//go:build !windows

package systemproxy

import (
	"errors"
	"testing"
)

func TestNonWindows(t *testing.T) {
	m := New()
	if m.Available() {
		t.Fatal("non-Windows backend available")
	}
	if !errors.Is(m.Enable("127.0.0.1:8080"), ErrUnsupported) {
		t.Fatal("Enable did not report unsupported")
	}
	if !errors.Is(m.Restore(), ErrUnsupported) {
		t.Fatal("Restore did not report unsupported")
	}
}
