package proxyruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestControllerMissingLegacyNeverFallsBack(t *testing.T) {
	c := New("/nonexistent/ProtosVPN")
	err := c.ConnectCompatibility(context.Background(), "vless://11111111-1111-4111-8111-111111111111@example.com:80?security=none")
	if err == nil || err.Error() != "COMPATIBILITY_UNAVAILABLE" {
		t.Fatal(err)
	}
	status, code := c.Status()
	if status != "error" || code != "COMPATIBILITY_UNAVAILABLE" {
		t.Fatal(status, code)
	}
	for _, code := range []string{"COMPATIBILITY_UNAVAILABLE", "UNSUPPORTED_COMPATIBILITY"} {
		if safeCode(code) != code {
			t.Fatal("lost safe code", code)
		}
	}
}

func TestWorkerPassesExplicitCompatibilityToFactory(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		selected := !legacy
		var output bytes.Buffer
		e := &testEngine{done: make(chan struct{})}
		p := &testProxy{}
		payload, _ := json.Marshal(request{URI: "vless://11111111-1111-4111-8111-111111111111@example.com:80?security=none", Compatibility: legacy})
		runWorkerWithFactory(strings.NewReader(string(payload)+"\n"), &output, func(wantLegacy bool) engine { selected = wantLegacy; return e }, p, func(context.Context, string) error { return errors.New("stop before system proxy") })
		if selected != legacy {
			t.Fatal("lost explicit selection", legacy, selected)
		}
		if p.enabled != 0 {
			t.Fatal("proxy unexpectedly enabled")
		}
	}
}

func TestWorkerRejectsCompatibilityForTLSBeforeStartingCore(t *testing.T) {
	e := &testEngine{done: make(chan struct{})}
	p := &testProxy{}
	var output bytes.Buffer
	runWorker(strings.NewReader("{\"uri\":\"vless://11111111-1111-4111-8111-111111111111@example.com:443?security=tls\",\"compatibility\":true}\n"), &output, e, p, func(context.Context, string) error { return errors.New("synthetic stop") })
	var msg message
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Code != "UNSUPPORTED_COMPATIBILITY" {
		t.Fatal(msg)
	}
	if p.enabled != 0 {
		t.Fatal("proxy changed for rejected downgrade")
	}
}
