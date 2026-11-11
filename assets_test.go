package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNativeAPIIsNeverHandledAsAnAsset(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "API") })
	assets := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "STATIC") })
	handler := newAPIMiddleware(api)(assets)
	for _, method := range []string{"GET", "POST", "PATCH", "DELETE"} {
		for _, path := range []string{"/api", "/api/state", "/api/import", "/api/profiles/123", "/api/diagnostics"} {
			t.Run(method+path, func(t *testing.T) {
				r := httptest.NewRecorder()
				handler.ServeHTTP(r, httptest.NewRequest(method, path, nil))
				if r.Body.String() != "API" {
					t.Fatalf("API request reached wrong handler: %q", r.Body.String())
				}
			})
		}
	}
}
func TestNativeAssetsAreNotExposedAsAPI(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("asset reached API") })
	assets := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "STATIC") })
	handler := newAPIMiddleware(api)(assets)
	for _, path := range []string{"/", "/index.html", "/assets/main.js", "/apiculture"} {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		if r.Body.String() != "STATIC" {
			t.Fatalf("asset not delivered: %s", path)
		}
	}
}
