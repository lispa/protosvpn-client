package proxyruntime

import (
	"context"
	"encoding/json"
	"io"
	"protosvpn.local/client/internal/proxyengine"
	"testing"
	"time"
)

type measuredEngine struct {
	testEngine
	samples uint64
}

func (e *measuredEngine) Stats(context.Context) (proxyengine.Traffic, error) {
	e.samples++
	return proxyengine.Traffic{UploadBytes: e.samples * 1000, DownloadBytes: e.samples * 2000}, nil
}
func TestWorkerPublishesMeasuredSessionAndTraffic(t *testing.T) {
	input, parent := io.Pipe()
	reader, output := io.Pipe()
	e := &measuredEngine{testEngine: testEngine{done: make(chan struct{})}}
	p := &testProxy{}
	done := make(chan struct{})
	go func() {
		runWorker(input, output, e, p, func(context.Context, string) error { time.Sleep(15 * time.Millisecond); return nil })
		output.Close()
		close(done)
	}()
	if err := json.NewEncoder(parent).Encode(request{URI: "secret"}); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(reader)
	var ready message
	if err := decoder.Decode(&ready); err != nil {
		t.Fatal(err)
	}
	if ready.Status != "connected" || ready.Metrics == nil || ready.Metrics.StartedAt == "" || ready.Metrics.LatencyMs == nil || *ready.Metrics.LatencyMs < 10 || ready.Metrics.UploadBytes == nil || *ready.Metrics.UploadBytes != 1000 {
		t.Fatal("actual initial measurements absent", ready)
	}
	var sample message
	if err := decoder.Decode(&sample); err != nil {
		t.Fatal(err)
	}
	if sample.Status != "metrics" || sample.Metrics == nil || *sample.Metrics.UploadBytes != 2000 || sample.Metrics.UploadBps == nil || *sample.Metrics.UploadBps <= 0 {
		t.Fatal("real traffic update absent", sample)
	}
	parent.Close()
	var stopped message
	if err := decoder.Decode(&stopped); err != nil {
		t.Fatal(err)
	}
	if stopped.Status != "stopped" {
		t.Fatal(stopped)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("metrics prevented EOF restoration")
	}
	if p.restored != 1 || !e.closed {
		t.Fatal("cleanup broken", p.restored, e.closed)
	}
}
