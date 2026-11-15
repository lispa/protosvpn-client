package systemproxy

import (
	"errors"
	"testing"
)

var secretError = errors.New("https://user:password@private/pac?token=secret")

type fakeBackend struct {
	current             snapshot
	sets                int
	queryErr, notifyErr error
	setHook             func(snapshot) error
}

func (b *fakeBackend) query() (snapshot, error) { return b.current, b.queryErr }
func (b *fakeBackend) set(s snapshot) error {
	b.sets++
	if b.setHook != nil {
		return b.setHook(s)
	}
	b.current = s
	return nil
}
func (b *fakeBackend) notify() error { return b.notifyErr }

func TestExactRestore(t *testing.T) {
	original := snapshot{flags: 15, server: "http=old:1;https=other:2", bypass: "<local>;*.private", pac: "https://private/pac?token=secret"}
	b := &fakeBackend{current: original}
	m := &Manager{backend: b}
	if err := m.Enable("127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if b.current.flags != 3 || b.current.server != "127.0.0.1:8080" || b.current.pac != "" || b.current.bypass != "" {
		t.Fatalf("unexpected enabled settings")
	}
	if err := m.Restore(); err != nil {
		t.Fatal(err)
	}
	if b.current != original {
		t.Fatal("snapshot was not restored exactly")
	}
	if err := m.Restore(); err != nil {
		t.Fatal(err)
	}
	if b.sets != 2 {
		t.Fatalf("idempotent restore wrote settings: %d", b.sets)
	}
}
