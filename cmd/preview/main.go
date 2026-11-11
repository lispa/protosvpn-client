// Command preview serves the real in-memory Stage 1 API and built frontend.
// Run from the workspace root: go run ./cmd/preview
package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"protosvpn.local/client/internal/client"
)

const previewAddress = "127.0.0.1:18761"

func previewHandler(dist string, s *client.Service) http.Handler {
	api := client.NewHandler(s)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed.", 405)
			return
		}
		// OpenRoot confines reads to dist, including symlink resolution. Never list directories.
		root, e := os.OpenRoot(dist)
		if e != nil {
			http.Error(w, "Frontend build unavailable.", 503)
			return
		}
		defer root.Close()
		clean := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if clean == "" || path.Ext(clean) == "" {
			clean = "index.html"
		}
		f, e := root.Open(clean)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, e := f.Stat()
		if e != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	})
}
func main() {
	dist := filepath.Join("frontend", "dist")
	if info, err := os.Stat(filepath.Join(dist, "index.html")); err != nil || !info.Mode().IsRegular() {
		log.Fatal("frontend/dist/index.html unavailable; build the frontend and run preview from the workspace root")
	}
	server := &http.Server{Addr: previewAddress, Handler: previewHandler(dist, client.New()), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	log.Printf("ProtosVPN preview: http://%s (memory-only; no VPN engine)", previewAddress)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("preview listener failed")
	}
}
