package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"protosvpn.local/client/internal/profilevault"
	"strings"
	"testing"
)

func TestPersistentLoadFailureBlocksEveryMutationAndHTTPIsSafe(t *testing.T) {
	f := &faultStore{loadErr: errors.New(testURI)}
	s := NewWithStore(nil, f)
	if s.State().StorageStatus != "error" || s.State().StorageErrorCode != "STORE_LOAD_FAILED" {
		t.Fatal(s.State())
	}
	ops := []func() error{func() error { _, e := s.Import("vless", testURI, ""); return e }, func() error { return s.Select("unknown") }, func() error { _, e := s.Rename("unknown", "New"); return e }, func() error { return s.Delete("unknown") }}
	for _, op := range ops {
		if err := op(); !errors.Is(err, ErrStoreLoad) {
			t.Fatalf("load guard: %v", err)
		}
	}
	if f.saves != 0 {
		t.Fatal("failed load overwritten")
	}
	body, _ := json.Marshal(map[string]string{"format": "vless", "content": testURI})
	w := call(NewHandler(s), "POST", "/api/import", string(body), s.State().CSRFToken)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "STORE_LOAD_FAILED") || strings.Contains(w.Body.String(), testURI) {
		t.Fatal(w.Code, w.Body.String())
	}
	f = &faultStore{}
	s = NewWithStore(nil, f)
	f.saveErr = errors.New(testURI)
	w = call(NewHandler(s), "POST", "/api/import", string(body), s.State().CSRFToken)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "STORE_SAVE_FAILED") || strings.Contains(w.Body.String(), testURI) {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestPersistentCorruptDiskNeverOverwritten(t *testing.T) {
	dir := t.TempDir()
	v := profilevault.New(dir)
	s := NewWithStore(nil, v)
	if _, err := s.Import("vless", testURI, ""); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, profilevault.FileName)
	raw, _ := os.ReadFile(path)
	raw[len(raw)-1] ^= 1
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s = NewWithStore(nil, v)
	if _, err := s.Import("vless", testURI, ""); !errors.Is(err, ErrStoreLoad) {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(raw, after) {
		t.Fatal("corrupt file overwritten")
	}
}

func TestPersistentStrictDocumentValidation(t *testing.T) {
	id := strings.Repeat("a", 64)
	good := storedProfile{ID: id, Name: "A", Protocol: "vless", Content: testURI}
	tests := map[string]storedProfiles{
		"duplicate IDs":     {Profiles: []storedProfile{good, good}, SelectedID: id},
		"invalid ID":        {Profiles: []storedProfile{{ID: "bad", Name: "A", Protocol: "vless", Content: testURI}}, SelectedID: "bad"},
		"invalid name":      {Profiles: []storedProfile{{ID: id, Name: "\n", Protocol: "vless", Content: testURI}}, SelectedID: id},
		"invalid content":   {Profiles: []storedProfile{{ID: id, Name: "A", Protocol: "vless", Content: "vless://secret"}}, SelectedID: id},
		"invalid selection": {Profiles: []storedProfile{good}, SelectedID: "missing"},
	}
	for name, doc := range tests {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(doc)
			f := &faultStore{data: raw}
			s := NewWithStore(nil, f)
			if s.State().StorageErrorCode != "STORE_LOAD_FAILED" || len(s.State().Profiles) != 0 {
				t.Fatal("invalid document accepted")
			}
		})
	}
	raw, _ := json.Marshal(storedProfiles{Profiles: []storedProfile{good}, SelectedID: id})
	raw = bytes.Replace(raw, []byte(`"name":"A"`), []byte(`"name":"X","name":"A"`), 1)
	s := NewWithStore(nil, &faultStore{data: raw})
	if s.State().StorageStatus != "error" {
		t.Fatal("duplicate JSON key accepted")
	}
}
