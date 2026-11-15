package client

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

type fakeConnector struct {
	mu           sync.Mutex
	status, code string
	secret       string
	fail         bool
}

func (f *fakeConnector) Available() bool { return true }
func (f *fakeConnector) Status() (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.status == "" {
		return "disconnected", ""
	}
	return f.status, f.code
}
func (f *fakeConnector) Connect(ctx context.Context, uri string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.secret = uri
	if f.fail {
		f.status = "error"
		f.code = "CONNECT_FAILED"
		return &ConnectionError{Code: "CONNECT_FAILED"}
	}
	f.status = "connected"
	return nil
}
func (f *fakeConnector) Disconnect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = "disconnected"
	f.secret = ""
	return nil
}
func TestProxyHTTPConnectionLifecycle(t *testing.T) {
	f := &fakeConnector{}
	s := NewWithConnector(f)
	p, err := s.Import("vless", testURI, "")
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(s)
	w := call(h, "POST", "/api/connect", `{"id":"`+p.ID+`"}`, s.token)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	state := s.State()
	if state.Connection.Status != "connected" || state.Connection.ProfileID != p.ID || state.Connection.Mode != "system-proxy" {
		t.Fatal(state.Connection)
	}
	if f.secret != testURI {
		t.Fatal("engine did not receive profile")
	}
	raw, _ := json.Marshal(state)
	if strings.Contains(string(raw), testURI) || strings.Contains(string(raw), "00000000-") {
		t.Fatal("secret leaked")
	}
	if !errors.Is(s.Delete(p.ID), ErrBusy) {
		t.Fatal("must not remove active profile")
	}
	w = call(h, "POST", "/api/disconnect", "{}", s.token)
	if w.Code != 200 || s.State().Connection.Status != "disconnected" {
		t.Fatal(w.Code, w.Body.String())
	}
	if err = s.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
}
func TestProxyConnectFailureDoesNotClaimConnected(t *testing.T) {
	f := &fakeConnector{fail: true}
	s := NewWithConnector(f)
	p, _ := s.Import("vless", testURI, "")
	w := call(NewHandler(s), "POST", "/api/connect", `{"id":"`+p.ID+`"}`, s.token)
	if w.Code != 502 || s.State().Connection.Status != "error" || !strings.Contains(w.Body.String(), "CONNECT_FAILED") {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestProxyCanDeleteFailedProfile(t *testing.T) {
	f := &fakeConnector{fail: true}
	s := NewWithConnector(f)
	p, _ := s.Import("vless", testURI, "")
	_ = s.Connect(context.Background(), p.ID)
	if err := s.Delete(p.ID); err != nil {
		t.Fatal("failed profile remained locked", err)
	}
}

func TestProxyRejectsOpenVPN(t *testing.T) {
	s := NewWithConnector(&fakeConnector{})
	s.entries = []entry{{profile: Profile{ID: "ovpn", Protocol: "openvpn"}}}
	if !errors.Is(s.Connect(context.Background(), "ovpn"), ErrUnsupportedProfile) {
		t.Fatal("OpenVPN must not be routed through VLESS engine")
	}
}
func TestProxyCloseRestores(t *testing.T) {
	f := &fakeConnector{}
	s := NewWithConnector(f)
	p, _ := s.Import("vless", testURI, "")
	if err := s.Connect(context.Background(), p.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if s.State().Connection.Status != "disconnected" {
		t.Fatal("close left proxy enabled")
	}
}
