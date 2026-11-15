package proxyruntime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type testEngine struct {
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

func (e *testEngine) Start(context.Context, string) (string, error) { return "127.0.0.1:12345", nil }
func (e *testEngine) Done() <-chan struct{}                         { return e.done }
func (e *testEngine) Close() error                                  { e.mu.Lock(); e.closed = true; e.mu.Unlock(); return nil }

type testProxy struct {
	mu                sync.Mutex
	enabled, restored int
}

func (p *testProxy) Enable(string) error { p.mu.Lock(); p.enabled++; p.mu.Unlock(); return nil }
func (p *testProxy) Restore() error      { p.mu.Lock(); p.restored++; p.mu.Unlock(); return nil }
func TestWorkerRestoresOnGUIEOFAndCoreExit(t *testing.T) {
	for _, crash := range []bool{false, true} {
		t.Run(map[bool]string{false: "GUI pipe EOF", true: "core exit"}[crash], func(t *testing.T) {
			input, w := io.Pipe()
			output, r := io.Pipe()
			e := &testEngine{done: make(chan struct{})}
			p := &testProxy{}
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				runWorker(input, r, e, p, func(context.Context, string) error { return nil })
				r.Close()
			}()
			if err := json.NewEncoder(w).Encode(request{URI: "synthetic"}); err != nil {
				t.Fatal(err)
			}
			decoder := json.NewDecoder(output)
			var msg message
			if err := decoder.Decode(&msg); err != nil || msg.Status != "connected" {
				t.Fatal(msg, err)
			}
			if crash {
				close(e.done)
			} else {
				w.Close()
			}
			if err := decoder.Decode(&msg); err != nil {
				t.Fatal(err)
			}
			if crash && msg.Status != "error" {
				t.Fatal(msg)
			}
			if !crash && msg.Status != "stopped" {
				t.Fatal(msg)
			}
			select {
			case <-finished:
			case <-time.After(3 * time.Second):
				t.Fatal("worker failed to stop")
			}
			w.Close()
			output.Close()
			p.mu.Lock()
			if p.enabled != 1 || p.restored != 1 {
				t.Fatal(p.enabled, p.restored)
			}
			p.mu.Unlock()
			e.mu.Lock()
			if !e.closed {
				t.Fatal("core not closed")
			}
			e.mu.Unlock()
		})
	}
}
func TestWorkerDoesNotSetProxyWhenRemoteCheckFails(t *testing.T) {
	input, w := io.Pipe()
	output, r := io.Pipe()
	e := &testEngine{done: make(chan struct{})}
	p := &testProxy{}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runWorker(input, r, e, p, func(context.Context, string) error { return errors.New("private upstream error must not escape") })
		r.Close()
	}()
	json.NewEncoder(w).Encode(request{URI: "synthetic"})
	var msg message
	if err := json.NewDecoder(output).Decode(&msg); err != nil || msg.Code != "EGRESS_CHECK_FAILED" {
		t.Fatal(msg, err)
	}
	w.Close()
	<-finished
	output.Close()
	if p.enabled != 0 {
		t.Fatal("system proxy enabled before egress passed")
	}
}
