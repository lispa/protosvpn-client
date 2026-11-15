package proxyruntime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestMetricsStreamHelper(t *testing.T) {
	if os.Getenv("PROTOSVPN_TEST_METRICS_HELPER") != "1" {
		return
	}
	fmt.Println(`{"status":"connected"}`)
	for i := 0; i < 1600; i++ {
		fmt.Println(`{"status":"metrics","metrics":{"startedAt":"2026-01-01T00:00:00Z","sampledAt":"2026-01-01T00:00:02Z","uploadBytes":1000,"downloadBytes":2000}}`)
	}
	fmt.Println(`{"status":"stopped"}`)
	os.Exit(0)
}
func TestMonitorSupportsLongLivedMetricsBeyond64KiB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMetricsStreamHelper$")
	cmd.Env = append(os.Environ(), "PROTOSVPN_TEST_METRICS_HELPER=1")
	input, _ := cmd.StdinPipe()
	output, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	s := &session{cmd: cmd, input: input, ready: make(chan message, 1), done: make(chan struct{})}
	c := New("unused")
	c.session = s
	c.status = "connecting"
	c.monitor(s, output)
	status, code := c.Status()
	if status != "disconnected" || code != "" {
		t.Fatal("long session broke IPC", status, code)
	}
	if ctx.Err() != nil {
		t.Fatal("monitor stalled on total stream size cap")
	}
}
