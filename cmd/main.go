package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
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
	// LogLevel is one of debug, info, warn, error.
	LogLevel slog.Level `env:"LOG_LEVEL" envDefault:"info"`
	// LogFormat is either json (default, for log collectors) or text.
	LogFormat string `env:"LOG_FORMAT" envDefault:"json"`
}

func main() {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		slog.Error("parse env", "err", err)
		os.Exit(1)
	}

	slog.SetDefault(newLogger(cfg.LogFormat, cfg.LogLevel))
	slog.Info("starting ipquery server")

	trusted, err := parseCIDRs(cfg.TrustedProxyCIDRs)
	if err != nil {
		slog.Error("invalid TRUSTED_PROXY_CIDRS", "err", err)
		os.Exit(1)
	}

	slog.Info("trusted proxies configured", "cidrs", cfg.TrustedProxyCIDRs)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	asn, err := ipqapi.NewAsnReader(cfg.GeoLiteAsn)
	if err != nil {
		slog.Error("open asn database", "path", cfg.GeoLiteAsn, "err", err)
		os.Exit(1)
	}
	defer asn.Close()

	city, err := ipqapi.NewCityReader(cfg.GeoLiteCity)
	if err != nil {
		slog.Error("open city database", "path", cfg.GeoLiteCity, "err", err)
		os.Exit(1)
	}
	defer city.Close()

	slog.Info("starting geoip database watchers", "interval", cfg.GeoIpReloadInterval.String())
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

	// Buffered so the goroutine never blocks if nobody is listening anymore.
	srvErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
	}()

	listenFailed := false
	select {
	case err := <-srvErr:
		slog.Error("http server failed", "err", err)
		listenFailed = true
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	stop()
	slog.Info("shutting down ipquery server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown", "err", err)
	}

	if listenFailed {
		os.Exit(1)
	}
}

func newLogger(format string, level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.EqualFold(format, "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
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
