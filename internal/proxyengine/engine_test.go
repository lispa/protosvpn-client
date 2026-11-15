package proxyengine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLifecycleFailure(t *testing.T) {
	e := New("/nonexistent/xray-secret")
	if _, err := e.Start(context.Background(), baseURI); err == nil || errors.Is(err, ErrInvalid) {
		t.Fatal("missing process accepted")
	}
	select {
	case <-e.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed")
	}
	if _, err := e.Start(context.Background(), baseURI); err == nil {
		t.Fatal("second start accepted")
	}
	if e.Close() != nil || e.Close() != nil {
		t.Fatal("close not idempotent")
	}
	e = New("unused")
	if e.Close() != nil {
		t.Fatal("close before start")
	}
	if _, err := e.Start(context.Background(), baseURI); err == nil {
		t.Fatal("closed engine started")
	}
}
func TestCheckRejectsUnsafe(t *testing.T) {
	for _, ep := range []string{"example.org:80", "127.0.0.1:0", "http://127.0.0.1:80", "[::]:80"} {
		if Check(context.Background(), ep, "https://example.org") == nil {
			t.Fatal("unsafe endpoint accepted")
		}
	}
}
