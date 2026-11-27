package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ipqapi "github.com/akyriako/ipquery/api"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Config struct {
	TrustedProxyCIDRs []string `env:"TRUSTED_PROXY_CIDRS" envSeparator:"," envDefault:"127.0.0.1/32,::1/128"`
	ListenAddr        string   `env:"LISTEN_ADDR" envDefault:":8080"`
	GeoLiteAsn        string   `env:"GEOLITE2_ASN" envDefault:"./geolite/GeoLite2-ASN.mmdb"`
	GeoLiteCity       string   `env:"GEOLITE2_CITY" envDefault:"./geolite/GeoLite2-City.mmdb"`
	AbuseIpDbApiKey   *string  `env:"ABUSEIPDB_API_KEY"`
	// GeoIpReloadInterval controls how often the mmdb files are checked for
	// replacement by geoipupdate.
	GeoIpReloadInterval time.Duration `env:"GEOIP_RELOAD_INTERVAL" envDefault:"60s"`
}

func main() {
	log.Print("starting ipquery server")

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("parse env: %v", err)
	}

	trusted, err := parseCIDRs(cfg.TrustedProxyCIDRs)
	if err != nil {
		log.Fatalf("invalid TRUSTED_PROXY_CIDRS: %v", err)
	}

	log.Printf("trustedProxies: %v", trusted)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	asn, err := ipqapi.NewAsnReader(cfg.GeoLiteAsn)
	if err != nil {
		log.Fatalf("asn reader error: %v", err)
	}
	defer asn.Close()

	city, err := ipqapi.NewCityReader(cfg.GeoLiteCity)
	if err != nil {
		log.Fatalf("city reader error: %v", err)
	}
	defer city.Close()

	log.Printf("geoip reload interval: %s", cfg.GeoIpReloadInterval)
	go asn.StartWatcher(ctx, cfg.GeoIpReloadInterval)
	go city.StartWatcher(ctx, cfg.GeoIpReloadInterval)

	lc := &ipqapi.LookupClient{TrustedProxies: trusted, AsnReader: asn, CityReader: city}
	if cfg.AbuseIpDbApiKey != nil {
		risk := ipqapi.NewAbuseIpDbChecker(*cfg.AbuseIpDbApiKey)
		lc.RiskChecker = risk
	}
	apis := ipqapi.Server{LookupClient: lc}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(ipqapi.AccessLogger(apis.GetClientIP))

	r.Get("/", apis.Index())

	r.Get("/own", apis.GetOwnIP)
	r.Get("/own/all", apis.GetOwnIPAll)
	r.Get("/lookup/{ip}", apis.LookupIPAll)
	r.Get("/health", apis.GetHealth)

	srv := &http.Server{Addr: cfg.ListenAddr, Handler: r}

	go func() {
		log.Printf("listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Print("shutting down ipquery server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown: %v", err)
	}
}

func parseCIDRs(items []string) ([]*net.IPNet, error) {
	var out []*net.IPNet
	for _, raw := range items {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		_, n, err := net.ParseCIDR(raw)
		if err != nil {
			return nil, fmt.Errorf("bad cidr %q: %w", raw, err)
		}
		out = append(out, n)
	}
	return out, nil
}
