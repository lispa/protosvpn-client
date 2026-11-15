package proxyruntime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

type failingEngine struct{ testEngine }

func (e *failingEngine) Start(context.Context, string) (string, error) {
	return "", errors.New("SECRET raw path or credential")
}
func TestWorkerReportsCoreStartupStage(t *testing.T) {
	input, w := io.Pipe()
	output, r := io.Pipe()
	e := &failingEngine{testEngine{done: make(chan struct{})}}
	p := &testProxy{}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runWorker(input, r, e, p, func(context.Context, string) error { t.Error("egress reached after core failure"); return nil })
		r.Close()
	}()
	_ = json.NewEncoder(w).Encode(request{URI: "synthetic"})
	var msg message
	err := json.NewDecoder(output).Decode(&msg)
	w.Close()
	<-finished
	output.Close()
	if err != nil || msg.Code != "ENGINE_START_FAILED" {
		t.Fatal(msg, err)
	}
	if p.enabled != 0 {
		t.Fatal("proxy changed on core startup failure")
	}
}
func TestControllerReportsHelperStartupStage(t *testing.T) {
	c := New(filepath.Join(t.TempDir(), "missing-client"))
	err := c.Connect(context.Background(), "SECRET URI")
	if err == nil || !strings.Contains(err.Error(), "HELPER_START_FAILED") {
		t.Fatal(err)
	}
	status, code := c.Status()
	if status != "error" || code != "HELPER_START_FAILED" {
		t.Fatal(status, code)
	}
}
func TestControllerPreservesDiagnosticCodes(t *testing.T) {
	for _, code := range []string{"ENGINE_START_FAILED", "EGRESS_CHECK_FAILED", "HELPER_START_FAILED", "HELPER_EXITED", "HELPER_TIMEOUT"} {
		if safeCode(code) != code {
			t.Errorf("lost diagnostic code %s", code)
		}
	}
	if safeCode("SECRET") != "CONNECT_FAILED" {
		t.Fatal("untrusted error escaped")
	}
}
