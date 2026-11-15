package client

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type compatibilityConnector struct {
	fakeConnector
	legacyCalls int
}

func (f *compatibilityConnector) ConnectCompatibility(ctx context.Context, uri string) error {
	f.legacyCalls++
	return f.Connect(ctx, uri)
}

const plaintextURI = "vless://11111111-1111-4111-8111-111111111111@example.com:80?security=none&type=tcp#Plaintext"

func TestCompatibilityCannotDowngradeTLS(t *testing.T) {
	f := &compatibilityConnector{}
	s := NewWithConnector(f)
	p, err := s.Import("vless", testURI, "")
	if err != nil {
		t.Fatal(err)
	}
	w := call(NewHandler(s), "POST", "/api/connect", `{"id":"`+p.ID+`","compatibility":true}`, s.token)
	if w.Code != 502 || !strings.Contains(w.Body.String(), "UNSUPPORTED_COMPATIBILITY") {
		t.Fatal(w.Code, w.Body.String())
	}
	if f.legacyCalls != 0 || f.secret != "" {
		t.Fatal("TLS reached legacy connector")
	}
}
func TestCompatibilityJSONRejectsAmbiguousConsent(t *testing.T) {
	f := &compatibilityConnector{}
	s := NewWithConnector(f)
	p, _ := s.Import("vless", plaintextURI, "")
	for _, tail := range []string{`"compatibility":null`, `"compatibility":"true"`, `"compatibility":1`, `"compatibility":{}`, `"compatibility":true,"compatibility":false`, `"id":true`, `"extra":true`} {
		w := call(NewHandler(s), "POST", "/api/connect", `{"id":"`+p.ID+`",`+tail+`}`, s.token)
		if w.Code != 400 {
			t.Fatal(tail, w.Code)
		}
	}
	if f.legacyCalls != 0 || f.secret != "" {
		t.Fatal("invalid request reached connector")
	}
}

func TestCompatibilityRequiresExplicitRequest(t *testing.T) {
	f := &compatibilityConnector{}
	s := NewWithConnector(f)
	p, err := s.Import("vless", plaintextURI, "")
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(s)
	normal := call(h, "POST", "/api/connect", `{"id":"`+p.ID+`"}`, s.token)
	if normal.Code != 200 {
		t.Fatal(normal.Code)
	}
	if f.legacyCalls != 0 {
		t.Fatal("implicit downgrade")
	}
	_ = s.Disconnect()
	legacy := call(h, "POST", "/api/connect", `{"id":"`+p.ID+`","compatibility":true}`, s.token)
	if legacy.Code != 200 {
		t.Fatal(legacy.Code, legacy.Body.String())
	}
	if f.legacyCalls != 1 {
		t.Fatal("explicit legacy choice lost")
	}
	raw, _ := json.Marshal(s.State())
	if !strings.Contains(string(raw), `"compatibility":true`) {
		t.Fatal("active compatibility warning unavailable")
	}
	if strings.Contains(string(raw), plaintextURI) {
		t.Fatal("URI leaked")
	}
}
