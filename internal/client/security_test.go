package client

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPGuards(t *testing.T) {
	s := New()
	h := NewHandler(s)
	for _, tc := range []struct {
		host, origin, token, fetch string
		want                       int
	}{
		{"127.0.0.1:18761", "", "", "", 403}, {"127.0.0.1:18761", "", "wrong", "", 403},
		{"evil.test", "", s.token, "", 403}, {"127.0.0.1.evil.test", "", s.token, "", 403}, {"localhost:18761", "http://evil.test", s.token, "", 403},
		{"localhost:18761", "null", s.token, "", 403}, {"localhost:18761", "http://localhost:18762", s.token, "", 403}, {"localhost:18761", "http://localhost:18761/path", s.token, "", 403},
		{"localhost:18761", "http://localhost:18761", s.token, "cross-site", 403},
		{"127.0.0.1:18761", "http://127.0.0.1:18761", s.token, "same-origin", 409}, {"localhost:18761", "http://localhost:18761", s.token, "", 409},
		{"[::1]:18761", "http://[::1]:18761", s.token, "", 409}, {"wails.localhost", "http://wails.localhost", s.token, "", 409}, {"wails", "wails://wails", s.token, "", 409},
	} {
		r := httptest.NewRequest("POST", "http://127.0.0.1/api/disconnect", strings.NewReader("{}"))
		r.Host = tc.host
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("X-ProtosVPN-Token", tc.token)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Sec-Fetch-Site", tc.fetch)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("host=%s origin=%s: got %d want %d", tc.host, tc.origin, w.Code, tc.want)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "" || strings.Contains(w.Body.String(), s.token) || strings.Contains(w.Body.String(), "evil.test") {
			t.Fatal("reflected data or CORS")
		}
	}
	for _, path := range []string{"/api/state", "/api/diagnostics", "/api/unknown"} {
		w := call(h, "GET", path, "", "")
		if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("missing response protection")
		}
	}
	for _, origin := range []string{"http://evil.test", "null"} {
		r := httptest.NewRequest("GET", "http://localhost/api/state", nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("bootstrap accessible to hostile origin")
		}
	}
}
func TestHTTPBodyAndSecretSafety(t *testing.T) {
	s := New()
	h := NewHandler(s)
	for _, body := range []string{`{"format":"vless","content":"secret-invalid-uri"}`, `{"format":"vless","content":"secret","extra":1}`, `{"format":"vless"} {}`, strings.Repeat(" ", MaxBytes+1)} {
		w := call(h, "POST", "/api/import", body, s.token)
		if w.Code != 400 {
			t.Errorf("invalid body accepted %d", w.Code)
		}
		if strings.Contains(w.Body.String(), "secret") {
			t.Fatal("reflected input")
		}
	}
	for _, body := range []string{`null`, `[]`, `{"format":"vless","format":"openvpn","content":"secret"}`, `{"format":null,"content":"secret"}`} {
		w := call(h, "POST", "/api/import", body, s.token)
		if w.Code != 400 {
			t.Errorf("ambiguous JSON accepted %d", w.Code)
		}
	}
	w := call(h, "POST", "/api/disconnect", `null`, s.token)
	if w.Code != 400 {
		t.Fatal("non-object body accepted")
	}
	w = call(h, "POST", "/api/select", `{}`, s.token)
	if w.Code != 400 {
		t.Fatal("missing ID not invalid input")
	}
	w = call(h, "POST", "/api/import", `{}`, "")
	if w.Code != 403 {
		t.Fatal("missing token allowed")
	}
}
