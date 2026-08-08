package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"
)

type CityReader struct {
	db *ReloadableGeoIPDB
}

func NewCityReader(path string) (*CityReader, error) {
	db, err := NewReloadableGeoIPDB(path)
	if err != nil {
		return nil, fmt.Errorf("open city mmdb: %w", err)
	}

	slog.Info("city mmdb opened", "path", path, "type", db.DatabaseType())

	return &CityReader{db: db}, nil
}

// StartWatcher polls the mmdb file and hot reloads it on change. Blocks until
// ctx is cancelled.
func (c *CityReader) StartWatcher(ctx context.Context, interval time.Duration) {
	c.db.StartWatcher(ctx, interval)
}

func (c *CityReader) Close() error { return c.db.Close() }

type cityRecord struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`

	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`

	Subdivisions []struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`

	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`

	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
}

func (c *CityReader) Enrich(ip net.IP, out *LookupResult) error {
	addr, ok := netIPToNetipAddr(ip)
	if !ok {
		return nil
	}

	var rec cityRecord
	if err := c.db.Lookup(addr, &rec); err != nil {
		return err
	}

	out.Location.Country = rec.Country.Names["en"]
	out.Location.CountryCode = rec.Country.ISOCode
	out.Location.City = rec.City.Names["en"]
	if len(rec.Subdivisions) > 0 {
		out.Location.State = rec.Subdivisions[0].Names["en"]
		if out.Location.State == "" {
			out.Location.State = rec.Subdivisions[0].ISOCode
		}
	}
	out.Location.Zipcode = rec.Postal.Code
	out.Location.Latitude = rec.Location.Latitude
	out.Location.Longitude = rec.Location.Longitude
	out.Location.Timezone = rec.Location.TimeZone

	if tz := rec.Location.TimeZone; tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			out.Location.Localtime = time.Now().In(loc).Format(time.RFC3339)
		}
	}

	return nil
}
