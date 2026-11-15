package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"protosvpn.local/client/internal/profilevault"
)

func TestPersistentProfilesRestart(t *testing.T) {
	store := profilevault.New(filepath.Join(t.TempDir(), "vault"))
	s := NewWithStore(nil, store)
	if s.State().StorageMode != "protected" || s.State().StorageStatus != "empty" {
		t.Fatal(s.State())
	}
	a, err := s.Import("vless", testURI, "First")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Import("vless", testURI, "Second")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Rename(a.ID, "Renamed"); err != nil {
		t.Fatal(err)
	}
	r := NewWithStore(nil, store)
	if !reflect.DeepEqual(r.State().Profiles, s.State().Profiles) || r.State().SelectedID != b.ID || r.State().StorageStatus != "ready" {
		t.Fatal("restart changed profiles")
	}
	if r.State().Connection.Status != "disconnected" || r.activeCompatibility {
		t.Fatal("connection restored")
	}
	if err = r.Delete(b.ID); err != nil {
		t.Fatal(err)
	}
	r = NewWithStore(nil, store)
	if len(r.State().Profiles) != 1 || r.State().SelectedID != a.ID {
		t.Fatal("delete not persisted")
	}
	state, _ := json.Marshal(r.State())
	diag, _ := json.Marshal(r.Diagnostics())
	if bytes.Contains(state, []byte(testURI)) || bytes.Contains(diag, []byte(testURI)) {
		t.Fatal("secret leaked")
	}
}

// Fault injection only at the I/O boundary; parsing and serialization remain real.
type faultStore struct {
	data             []byte
	loadErr, saveErr error
	saves            int
}

func (f *faultStore) Load() ([]byte, error) {
	if f.data == nil && f.loadErr == nil {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), f.data...), f.loadErr
}
func (f *faultStore) Save(b []byte) error {
	f.saves++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.data = append([]byte(nil), b...)
	return nil
}

func TestPersistentSaveRollback(t *testing.T) {
	f := &faultStore{}
	s := NewWithStore(nil, f)
	a, _ := s.Import("vless", testURI, "A")
	b, _ := s.Import("vless", testURI, "B")
	before := s.State()
	disk := append([]byte(nil), f.data...)
	f.saveErr = errors.New(testURI)
	operations := []func() error{func() error { _, err := s.Import("vless", testURI, "C"); return err }, func() error { return s.Select(b.ID) }, func() error { _, err := s.Rename(a.ID, "changed"); return err }, func() error { return s.Delete(a.ID) }}
	for _, op := range operations {
		if err := op(); !errors.Is(err, ErrStoreSave) {
			t.Fatalf("save error: %v", err)
		}
		after := s.State()
		if !reflect.DeepEqual(before.Profiles, after.Profiles) || before.SelectedID != after.SelectedID || !bytes.Equal(disk, f.data) {
			t.Fatal("partial mutation")
		}
	}
	f.saveErr = nil
	if err := s.Select(b.ID); err != nil {
		t.Fatal("recovery", err)
	}
}
