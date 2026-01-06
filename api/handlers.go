package api

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	*LookupClient
}

func (s *Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) GetOwnIP(w http.ResponseWriter, r *http.Request) {
	ipStr := s.GetClientIP(r)
	if ipStr == "" {
		http.Error(w, "unable to determine client ip", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(ipStr))
}

func (s *Server) GetOwnIPAll(w http.ResponseWriter, r *http.Request) {
	ipStr := s.GetClientIP(r)
	if ipStr == "" {
		http.Error(w, "unable to determine client ip", http.StatusBadRequest)
		return
	}

	s.lookupIPAll(w, r, ipStr)
}

func (s *Server) LookupIPAll(w http.ResponseWriter, r *http.Request) {
	ipStr := chi.URLParam(r, "ip")
	if ipStr == "" {
		http.Error(w, "missing ip parameter", http.StatusBadRequest)
		return
	}

	s.lookupIPAll(w, r, ipStr)
}

func (s *Server) lookupIPAll(w http.ResponseWriter, r *http.Request, ipStr string) {
	ipNet := net.ParseIP(ipStr)
	if ipNet == nil {
		http.Error(w, "invalid ip", http.StatusBadRequest)
		return
	}

	res := LookupResult{IP: ipNet.String()}

	if err := s.AsnReader.Enrich(ipNet, &res); err != nil {
		http.Error(w, "asn lookup failed", http.StatusInternalServerError)
		return
	}
	if err := s.CityReader.Enrich(ipNet, &res); err != nil {
		http.Error(w, "city lookup failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}
