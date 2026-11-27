package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

type AsnRecord struct {
	ASN uint   `maxminddb:"autonomous_system_number"`
	Org string `maxminddb:"autonomous_system_organization"`
}

type AsnReader struct {
	db *ReloadableGeoIPDB
}

func NewAsnReader(path string) (*AsnReader, error) {
	db, err := NewReloadableGeoIPDB(path)
	if err != nil {
		return nil, fmt.Errorf("open asn mmdb: %w", err)
	}

	log.Printf("asn mmdb type: %s", db.DatabaseType())

	return &AsnReader{db: db}, nil
}

// StartWatcher polls the mmdb file and hot reloads it on change. Blocks until
// ctx is cancelled.
func (a *AsnReader) StartWatcher(ctx context.Context, interval time.Duration) {
	a.db.StartWatcher(ctx, interval)
}

func (a *AsnReader) Close() error { return a.db.Close() }

func (a *AsnReader) Enrich(ip net.IP, out *LookupResult) error {
	addr, ok := netIPToNetipAddr(ip)
	if !ok {
		return nil
	}

	var rec AsnRecord
	if err := a.db.Lookup(addr, &rec); err != nil {
		return err
	}

	if rec.ASN == 0 && rec.Org == "" {
		return nil
	}

	out.ISP.ASN = fmt.Sprintf("AS%d", rec.ASN)
	out.ISP.Org = rec.Org
	out.ISP.ISP = rec.Org
	return nil
}
