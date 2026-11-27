package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang/v2"
)

// DefaultReloadInterval is used when no explicit interval is configured.
const DefaultReloadInterval = 60 * time.Second

// fileSignature is the cheap, portable fingerprint we use to detect that an
// mmdb file on disk has been replaced. geoipupdate writes a temporary file and
// renames it over the target, so the path must be re-stat'ed instead of
// relying on the already open file descriptor.
type fileSignature struct {
	modTime time.Time
	size    int64
}

func statSignature(path string) (fileSignature, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return fileSignature{}, err
	}
	return fileSignature{modTime: fi.ModTime(), size: fi.Size()}, nil
}

// ReloadableGeoIPDB keeps a single open maxminddb.Reader that is shared by all
// lookups and swaps it for a freshly opened one whenever the underlying file
// changes on disk. It is safe for concurrent use.
type ReloadableGeoIPDB struct {
	path string

	mu sync.RWMutex
	db *maxminddb.Reader

	sig fileSignature
}

// NewReloadableGeoIPDB opens path and returns a database handle. The watcher is
// not started automatically; call StartWatcher.
func NewReloadableGeoIPDB(path string) (*ReloadableGeoIPDB, error) {
	db, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}

	// A failing stat here is not fatal: the next check cycle will simply see a
	// different signature and attempt a reload.
	sig, _ := statSignature(path)

	return &ReloadableGeoIPDB{path: path, db: db, sig: sig}, nil
}

// Path returns the file this database was opened from.
func (d *ReloadableGeoIPDB) Path() string { return d.path }

// DatabaseType returns the mmdb metadata database type of the current reader.
func (d *ReloadableGeoIPDB) DatabaseType() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return ""
	}
	return d.db.Metadata.DatabaseType
}

// Lookup resolves addr against the currently active reader and decodes the
// record into out. The reader is held under a read lock for the whole decode,
// which is what makes it safe to close a replaced reader under the write lock.
func (d *ReloadableGeoIPDB) Lookup(addr netip.Addr, out any) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return fmt.Errorf("geoip database %s is closed", d.path)
	}
	return d.db.Lookup(addr).Decode(out)
}

// StartWatcher polls the file for changes until ctx is cancelled. It blocks, so
// callers normally run it in its own goroutine.
func (d *ReloadableGeoIPDB) StartWatcher(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultReloadInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.reloadIfChanged(); err != nil {
				slog.ErrorContext(ctx, "failed to reload GeoLite2 database, continuing with previous database",
					"path", d.path, "err", err)
			}
		}
	}
}

// reloadIfChanged is a single watcher iteration, exposed for tests. It is a
// no-op (and silent) while the file signature is unchanged.
func (d *ReloadableGeoIPDB) reloadIfChanged() error {
	sig, err := statSignature(d.path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	d.mu.RLock()
	unchanged := sig == d.sig
	closed := d.db == nil
	d.mu.RUnlock()

	if closed || unchanged {
		return nil
	}

	// Open the replacement first: if it is corrupt we keep serving from the
	// current reader and retry on the next cycle.
	next, err := maxminddb.Open(d.path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	d.mu.Lock()
	// Re-check under the write lock: the Open above runs unlocked, so a
	// concurrent Close may have set d.db to nil meanwhile. Swapping next in
	// anyway would revive the database after Close reported success, leaving a
	// reader nobody closes. Harmless if the process exits right away, but not
	// when Close is followed by more work in the same process.
	if d.db == nil {
		d.mu.Unlock()
		_ = next.Close()
		return nil
	}
	prev := d.db
	d.db = next
	d.sig = sig
	// Holding the write lock guarantees no lookup is inside prev, and no new
	// lookup can enter it, so closing here is safe.
	err = prev.Close()
	d.mu.Unlock()

	if err != nil {
		slog.Warn("GeoLite2 database reloaded, but closing the previous reader failed",
			"path", d.path, "err", err)
		return nil
	}

	slog.Info("GeoLite2 database reloaded", "path", d.path)
	return nil
}

// Close releases the active reader. It is idempotent.
func (d *ReloadableGeoIPDB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil
	}
	db := d.db
	d.db = nil
	return db.Close()
}
