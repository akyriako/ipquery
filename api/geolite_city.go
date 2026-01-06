package api

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/oschwald/maxminddb-golang/v2"
)

type CityReader struct {
	db *maxminddb.Reader
}

func NewCityReader(path string) (*CityReader, error) {
	db, err := maxminddb.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open city mmdb: %w", err)
	}

	log.Printf("city mmdb type: %s", db.Metadata.DatabaseType)

	return &CityReader{db: db}, nil
}

func (c *CityReader) Close() error { return c.db.Close() }

// GeoLite2-City record shape (only fields we need)
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
	if err := c.db.Lookup(addr).Decode(&rec); err != nil {
		return err
	}

	// Prefer English names
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

	// localtime
	if tz := rec.Location.TimeZone; tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			out.Location.Localtime = time.Now().In(loc).Format(time.RFC3339)
		}
	}

	return nil
}
