package mw

import (
	"log"
	"net/http"
	"strings"
)

// CORS allows requests from configured origins.
func CORS(allowed []string, next http.Handler) http.Handler {
	allowedOrigins := make(map[string]bool, len(allowed))
	for _, origin := range allowed {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowedOrigins[origin] = true
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		log.Printf(
			"CORS check: origin=%q allowed=%v method=%s",
			origin,
			allowedOrigins[origin],
			r.Method,
		)

		if allowedOrigins[origin] && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, DELETE, OPTIONS",
			)
			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Authorization, Content-Type",
			)
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
