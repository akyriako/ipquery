package api

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var testAddr = netip.MustParseAddr("1.2.3.4")

type testRecord struct {
	Test string `maxminddb:"test"`
}

// writeDB atomically installs content at path the same way geoipupdate does:
// write a temp file, then rename over the target.
func writeDB(t *testing.T, path string, content []byte) {
	t.Helper()

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		t.Fatalf("write temp db: %v", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatalf("rename db: %v", err)
	}
	// Make sure the new file has a distinguishable ModTime even on filesystems
	// with coarse timestamps.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func newTestDB(t *testing.T, value string) (*ReloadableGeoIPDB, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "Test.mmdb")
	writeDB(t, path, buildTestMMDB(value))

	db, err := NewReloadableGeoIPDB(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db, path
}

func mustLookup(t *testing.T, db *ReloadableGeoIPDB) string {
	t.Helper()

	var rec testRecord
	if err := db.Lookup(testAddr, &rec); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	return rec.Test
}

func TestOpensAtStartup(t *testing.T) {
	db, _ := newTestDB(t, "v1")

	if got := db.DatabaseType(); got != "Test" {
		t.Errorf("DatabaseType = %q, want %q", got, "Test")
	}
	if got := mustLookup(t, db); got != "v1" {
		t.Errorf("lookup = %q, want %q", got, "v1")
	}
}

func TestNoReloadWhenUnchanged(t *testing.T) {
	db, _ := newTestDB(t, "v1")

	before := db.db
	for i := 0; i < 3; i++ {
		if err := db.reloadIfChanged(); err != nil {
			t.Fatalf("reloadIfChanged: %v", err)
		}
	}
	if db.db != before {
		t.Error("reader was swapped even though the file did not change")
	}
}

func TestReloadPicksUpReplacedFile(t *testing.T) {
	db, path := newTestDB(t, "v1")

	writeDB(t, path, buildTestMMDB("v2"))
	if err := db.reloadIfChanged(); err != nil {
		t.Fatalf("reloadIfChanged: %v", err)
	}

	if got := mustLookup(t, db); got != "v2" {
		t.Errorf("lookup after reload = %q, want %q", got, "v2")
	}
}

func TestCorruptReplacementKeepsPreviousDatabase(t *testing.T) {
	db, path := newTestDB(t, "v1")

	writeDB(t, path, []byte("this is not an mmdb file"))
	if err := db.reloadIfChanged(); err == nil {
		t.Fatal("expected an error for a corrupt database")
	}

	if got := mustLookup(t, db); got != "v1" {
		t.Errorf("lookup after failed reload = %q, want %q", got, "v1")
	}

	// A subsequent valid replacement must reload again.
	writeDB(t, path, buildTestMMDB("v3"))
	if err := db.reloadIfChanged(); err != nil {
		t.Fatalf("reloadIfChanged after recovery: %v", err)
	}
	if got := mustLookup(t, db); got != "v3" {
		t.Errorf("lookup after recovery = %q, want %q", got, "v3")
	}
}

func TestConcurrentLookupsDuringReload(t *testing.T) {
	db, path := newTestDB(t, "v1")

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				var rec testRecord
				if err := db.Lookup(testAddr, &rec); err != nil {
					t.Errorf("lookup during reload: %v", err)
					return
				}
				if rec.Test == "" {
					t.Error("empty record during reload")
					return
				}
			}
		}()
	}

	for i := 0; i < 20; i++ {
		writeDB(t, path, buildTestMMDB("v"+string(rune('A'+i))))
		if err := db.reloadIfChanged(); err != nil {
			t.Errorf("reloadIfChanged: %v", err)
		}
	}

	close(stop)
	wg.Wait()
}

func TestWatcherStopsOnContextCancel(t *testing.T) {
	db, path := newTestDB(t, "v1")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		db.StartWatcher(ctx, 10*time.Millisecond)
		close(done)
	}()

	writeDB(t, path, buildTestMMDB("v2"))

	deadline := time.After(2 * time.Second)
	for {
		var rec testRecord
		if err := db.Lookup(testAddr, &rec); err != nil {
			t.Fatalf("lookup: %v", err)
		}
		if rec.Test == "v2" {
			break
		}
		select {
		case <-deadline:
			t.Fatal("watcher did not pick up the new database")
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop on context cancellation")
	}
}

func TestCloseReleasesResources(t *testing.T) {
	db, _ := newTestDB(t, "v1")

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("second close should be a no-op: %v", err)
	}

	var rec testRecord
	if err := db.Lookup(testAddr, &rec); err == nil {
		t.Error("lookup on a closed database should fail, not panic or succeed")
	}
	// Reload on a closed database must stay a no-op.
	if err := db.reloadIfChanged(); err != nil {
		t.Errorf("reloadIfChanged after close: %v", err)
	}
}
