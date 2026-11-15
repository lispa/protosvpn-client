package systemproxy

import (
	"errors"
	"testing"
)

func TestConflictStillUsingLocalCore(t *testing.T) {
	b := &fakeBackend{current: snapshot{flags: 1}}
	m := &Manager{backend: b}
	if err := m.Enable("127.0.0.1:12345"); err != nil {
		t.Fatal(err)
	}
	b.current.bypass = "<local>"
	if !errors.Is(m.Restore(), ErrConflict) {
		t.Fatal("external bypass change was overwritten")
	}
	if active, err := m.OwnsEndpoint(); err != nil || !active {
		t.Fatal("helper must keep local core alive")
	}
	b.current.server = "other.example:8888"
	if active, err := m.OwnsEndpoint(); err != nil || active {
		t.Fatal("helper should release after external server change")
	}
	b.current.server = "127.0.0.1:12345"
	b.current.flags = 1
	if active, err := m.OwnsEndpoint(); err != nil || active {
		t.Fatal("disabled manual proxy should release the core")
	}
}
