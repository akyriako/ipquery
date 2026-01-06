package api

import (
	"log"
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

			log.Printf(
				`ip=%s method=%s path=%s status=%d duration=%s agent=%q`,
				getClientIp(r),
				r.Method,
				r.URL.Path,
				ww.status,
				time.Since(start),
				r.UserAgent(),
			)
		})
	}
}
