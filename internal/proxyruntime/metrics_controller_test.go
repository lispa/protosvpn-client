package proxyruntime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestLiveMetricsHelper(t *testing.T) {
	if os.Getenv("PROTOSVPN_TEST_LIVE_METRICS") != "1" {
		return
	}
	fmt.Println(`{"status":"connected","metrics":{"startedAt":"2026-01-01T00:00:00Z","latencyMs":12,"uploadBytes":777,"downloadBytes":888}}`)
	io.Copy(io.Discard, os.Stdin)
	fmt.Println(`{"status":"stopped"}`)
	os.Exit(0)
}
func TestControllerPublishesIPCMeasurementsAndClearsOnDisconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLiveMetricsHelper$")
	cmd.Env = append(os.Environ(), "PROTOSVPN_TEST_LIVE_METRICS=1")
	input, _ := cmd.StdinPipe()
	output, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	s := &session{cmd: cmd, input: input, ready: make(chan message, 1), done: make(chan struct{})}
	c := New("unused")
	c.session = s
	c.status = "connecting"
	go c.monitor(s, output)
	select {
	case m := <-s.ready:
		if m.Status != "connected" {
			t.Fatal(m)
		}
	case <-ctx.Done():
		t.Fatal("no ready")
	}
	m := c.Metrics()
	if m == nil || m.UploadBytes == nil || *m.UploadBytes != 777 || *m.LatencyMs != 12 {
		t.Fatal("IPC measurements missing", m)
	}
	*m.UploadBytes = 0
	if *c.Metrics().UploadBytes != 777 {
		t.Fatal("IPC snapshot aliases controller state")
	}
	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if c.Metrics() != nil {
		t.Fatal("stale session after disconnect")
	}
}
