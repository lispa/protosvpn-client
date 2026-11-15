package systemproxy

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestEndpointValidation(t *testing.T) {
	for _, s := range []string{"127.0.0.1:8080", "127.42.0.7:1", "[::1]:65535"} {
		if !validEndpoint(s) {
			t.Errorf("rejected loopback %q", s)
		}
	}
	for _, s := range []string{"localhost:80", "example.com:80", "0.0.0.0:80", "[::]:80", "192.168.1.1:80", "127.0.0.1:0", "127.0.0.1:65536", "127.0.0.1:+80", "127.0.0.1: 80", "http://127.0.0.1:80", "user:pass@127.0.0.1:80", "127.0.0.1:80;other:9", "[::1%lo]:80", "127.0.0.1:80\x00", "127.1:80", "[127.0.0.1]:80", "127.0.0.1:"} {
		b := &fakeBackend{}
		m := &Manager{backend: b}
		if err := m.Enable(s); err == nil {
			t.Errorf("accepted invalid endpoint %q", s)
		}
		if b.sets != 0 {
			t.Fatal("invalid endpoint changed settings")
		}
	}
}

func TestConflictForEachField(t *testing.T) {
	changes := []func(*snapshot){func(s *snapshot) { s.flags ^= 8 }, func(s *snapshot) { s.server = "external" }, func(s *snapshot) { s.bypass = "external" }, func(s *snapshot) { s.pac = "https://private/secret" }}
	for _, change := range changes {
		b := &fakeBackend{current: snapshot{flags: 9, pac: "original"}}
		m := &Manager{backend: b}
		if err := m.Enable("127.0.0.1:8080"); err != nil {
			t.Fatal(err)
		}
		ours := b.current
		change(&b.current)
		changed := b.current
		err := m.Restore()
		if !errors.Is(err, ErrConflict) || strings.Contains(err.Error(), "secret") {
			t.Fatal("missing sanitized conflict")
		}
		if b.current != changed || b.sets != 1 {
			t.Fatal("overwrote external settings")
		}
		b.current = ours
		if err := m.Restore(); err != nil {
			t.Fatal("conflict retry failed", err)
		}
	}
}

func TestFailedWritesCanRestore(t *testing.T) {
	for _, stage := range []string{"none", "some", "all", "readback-failure"} {
		t.Run(stage, func(t *testing.T) {
			original := snapshot{flags: 15, server: "old", bypass: "bypass", pac: "private"}
			b := &fakeBackend{current: original}
			m := &Manager{backend: b}
			b.setHook = func(s snapshot) error {
				switch stage {
				case "some":
					b.current.server = s.server
				case "all", "readback-failure":
					b.current = s
				}
				if stage == "readback-failure" {
					b.queryErr = secretError
				}
				return secretError
			}
			err := m.Enable("127.0.0.1:8080")
			if err == nil || strings.Contains(err.Error(), "secret") {
				t.Fatal("missing sanitized set failure")
			}
			b.setHook = nil
			b.queryErr = nil
			if err := m.Restore(); err != nil {
				t.Fatal("failed enable must remain recoverable", err)
			}
			if b.current != original {
				t.Fatal("lost original snapshot")
			}
		})
	}
}

func TestRestoreWriteFailureRetry(t *testing.T) {
	for _, partial := range []bool{false, true} {
		b := &fakeBackend{current: snapshot{flags: 13, server: "old", bypass: "local", pac: "pac"}}
		original := b.current
		m := &Manager{backend: b}
		if err := m.Enable("[::1]:8080"); err != nil {
			t.Fatal(err)
		}
		b.setHook = func(s snapshot) error {
			if partial {
				b.current.flags = s.flags
				b.current.pac = s.pac
			}
			return secretError
		}
		if err := m.Restore(); err == nil {
			t.Fatal("expected set failure")
		}
		b.setHook = nil
		if err := m.Restore(); err != nil {
			t.Fatal("restore retry failed", err)
		}
		if b.current != original {
			t.Fatal("lost snapshot")
		}
	}
}

func TestNotificationFailureRetry(t *testing.T) {
	b := &fakeBackend{current: snapshot{flags: 9, pac: "pac"}, notifyErr: secretError}
	original := b.current
	m := &Manager{backend: b}
	if err := m.Enable("127.0.0.1:8080"); err == nil {
		t.Fatal("expected notify failure")
	}
	if err := m.Restore(); err == nil {
		t.Fatal("expected restore notification failure")
	}
	if b.current != original {
		t.Fatal("restore did not write original")
	}
	b.notifyErr = nil
	if err := m.Restore(); err != nil {
		t.Fatal(err)
	}
	if m.original != nil {
		t.Fatal("restore did not complete")
	}
}

func TestQueryFailureRetryAndNoSecretErrors(t *testing.T) {
	b := &fakeBackend{queryErr: secretError}
	m := &Manager{backend: b}
	if err := m.Enable("127.0.0.1:8080"); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("query failure must be sanitized")
	}
	if m.original != nil || b.sets != 0 {
		t.Fatal("failed query claimed ownership")
	}
	b.queryErr = nil
	if err := m.Enable("127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if err := m.Enable("127.0.0.1:8081"); err == nil {
		t.Fatal("must not overwrite original snapshot")
	}
	b.queryErr = secretError
	if err := m.Restore(); err == nil {
		t.Fatal("query error swallowed")
	}
	b.queryErr = nil
	if err := m.Restore(); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentRestore(t *testing.T) {
	b := &fakeBackend{}
	m := &Manager{backend: b}
	if err := m.Enable("127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Restore(); err != nil {
				t.Error(err)
			}
			m.Available()
		}()
	}
	wg.Wait()
	if b.sets != 2 {
		t.Fatal("concurrent restore was not idempotent")
	}
}

func TestUnsupportedManager(t *testing.T) {
	m := &Manager{}
	if m.Available() {
		t.Fatal("zero manager available")
	}
	if !errors.Is(m.Enable("127.0.0.1:8080"), ErrUnsupported) || !errors.Is(m.Restore(), ErrUnsupported) {
		t.Fatal("missing ErrUnsupported")
	}
}
