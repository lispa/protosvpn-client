package client

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestMemoryCRUD(t *testing.T) {
	s := New()
	initial := s.State()
	if initial.Profiles == nil || len(initial.Profiles) != 0 || initial.Connection.Status != "disconnected" || initial.Connection.EngineAvailable || initial.CSRFToken == "" || initial.StorageMode != "memory" {
		t.Fatalf("invalid initial state: %+v", initial)
	}
	a, _ := s.Import("vless", testURI, "")
	b, _ := s.Import("openvpn", testOVPN, "Second")
	if s.State().SelectedID != a.ID {
		t.Fatal("first not selected")
	}
	if err := s.Select(b.ID); err != nil {
		t.Fatal(err)
	}
	p, e := s.Rename(a.ID, "Профиль")
	if e != nil || p.Name != "Профиль" {
		t.Fatal("rename failed")
	}
	if _, e = s.Rename(a.ID, strings.Repeat("я", 81)); e == nil {
		t.Fatal("long label accepted")
	}
	for _, bad := range []string{" ", "bad\nlabel"} {
		if _, e = s.Rename(a.ID, bad); e == nil {
			t.Fatal("invalid label accepted")
		}
	}
	if e = s.Delete(b.ID); e != nil || s.State().SelectedID != a.ID {
		t.Fatal("delete selection fallback failed")
	}
	if s.Diagnostics().ProfileCount != 1 {
		t.Fatal("diagnostic count wrong")
	}
	state := s.State()
	state.Profiles[0].Name = "mutated"
	state.Profiles[0].Warnings[0] = "mutated"
	if s.State().Profiles[0].Name == "mutated" || s.State().Profiles[0].Warnings[0] == "mutated" {
		t.Fatal("mutable snapshot")
	}
	raw, _ := json.Marshal(s.State())
	if strings.Contains(string(raw), "00000000-") || strings.Contains(string(raw), "SYNTHETIC") {
		t.Fatal("secret exposure")
	}
	if s.Select("missing") != ErrNotFound || s.Delete("missing") != ErrNotFound {
		t.Fatal("missing profile not reported")
	}
	s.Delete(a.ID)
	if len(s.State().Profiles) != 0 || s.State().SelectedID != "" || len(s.entries) != 0 {
		t.Fatal("not removed")
	}
	if len(New().State().Profiles) != 0 {
		t.Fatal("state persisted")
	}
}
func TestConcurrentCRUD(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				p, e := s.Import("vless", testURI, "")
				if e != nil {
					t.Error(e)
					return
				}
				s.Select(p.ID)
				s.Rename(p.ID, "Rename")
				s.State()
				s.Diagnostics()
				s.Delete(p.ID)
			}
		}()
	}
	wg.Wait()
	if len(s.State().Profiles) != 0 {
		t.Fatal("profiles left")
	}
}
