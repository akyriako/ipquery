package api

import (
	"log/slog"
	"net/http"
	"time"
)

type ClientIpFunc func(r *http.Request) string

type ResponseWriterWithStatusCode struct {
	http.ResponseWriter
	status int
}

func (w *ResponseWriterWithStatusCode) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func AccessLogger(getClientIp ClientIpFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			ww := &ResponseWriterWithStatusCode{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(ww, r)

			slog.LogAttrs(r.Context(), slog.LevelInfo, "access",
				slog.String("ip", getClientIp(r)),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.status),
				// slog encodes a Duration as nanoseconds in JSON, which is
				// awkward to read and to threshold on in queries.
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
				slog.String("agent", r.UserAgent()),
			)
		})
	}
}
