package main

import (
	"net/http"
	"strings"
)

// newAPIMiddleware routes local application API requests before asset fallback.
func newAPIMiddleware(api http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
				api.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
