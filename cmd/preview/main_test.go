package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"protosvpn.local/client/internal/client"
	"strings"
	"testing"
)

func TestPreviewRoutes(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<!doctype html><title>ProtosVPN test</title>"), 0600); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dist, "app.js"), []byte("// local asset"), 0600)
	h := previewHandler(dist, client.New())
	for _, tc := range []struct {
		path     string
		status   int
		contains string
	}{{"/", 200, "ProtosVPN test"}, {"/profiles", 200, "ProtosVPN test"}, {"/app.js", 200, "local asset"}, {"/missing.js", 404, ""}, {"/api/state", 200, `"storageMode":"memory"`}, {"/api/unknown", 404, `"error"`}, {"/api", 404, `"error"`}} {
		r := httptest.NewRequest("GET", "http://127.0.0.1:18761"+tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.contains) {
			t.Errorf("%s: %d %s", tc.path, w.Code, w.Body.String())
		}
		if strings.HasPrefix(tc.path, "/api") && strings.Contains(w.Body.String(), "doctype") {
			t.Fatal("API SPA fallback")
		}
	}
	if previewAddress != "127.0.0.1:18761" {
		t.Fatal("nonloopback listener")
	}
}
