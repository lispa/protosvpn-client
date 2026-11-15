package proxyengine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"time"
)

// Traffic contains cumulative bytes at the proxy outbound for this engine's lifetime.
// This is core accounting (including protocol overhead), not a network-interface meter.
type Traffic struct {
	UploadBytes   uint64
	DownloadBytes uint64
}

var ErrStatsUnavailable = errors.New("proxyengine: traffic statistics unavailable")

const maxStatsOutput = 4096
const statsPattern = "outbound>>>proxy>>>traffic>>>"

// Stats reads only the proxy outbound's cumulative counters; it never resets them.
// Errors deliberately contain no core output, configuration, credentials or paths.
func (e *Engine) Stats(ctx context.Context) (Traffic, error) {
	e.mu.Lock()
	endpoint, binary := e.statsEndpoint, e.binary
	available := e.ready && !e.closed
	e.mu.Unlock()
	if !available || endpoint == "" {
		return Traffic{}, ErrStatsUnavailable
	}
	select {
	case <-e.done:
		return Traffic{}, ErrStatsUnavailable
	default:
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	// Close and unexpected child exit also stop an outstanding stats subprocess.
	go func() {
		select {
		case <-e.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	cmd := exec.CommandContext(ctx, binary, "api", "statsquery", "--server="+endpoint, "-pattern", statsPattern, "-timeout", "3")
	configureProcess(cmd)
	output := &statsOutput{}
	cmd.Stdout = output
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil || ctx.Err() != nil || output.overflow {
		return Traffic{}, ErrStatsUnavailable
	}
	e.mu.Lock()
	closed := e.closed
	e.mu.Unlock()
	if closed {
		return Traffic{}, ErrStatsUnavailable
	}
	select {
	case <-e.done:
		return Traffic{}, ErrStatsUnavailable
	default:
	}
	return parseStats(output.Bytes())
}

// Drain excessive output without retaining it or exposing raw subprocess errors.
type statsOutput struct {
	buffer   bytes.Buffer
	overflow bool
}

func (w *statsOutput) Bytes() []byte { return w.buffer.Bytes() }

func (w *statsOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := maxStatsOutput - w.buffer.Len()
	if n > remaining {
		w.overflow = true
		p = p[:remaining]
	}
	_, _ = w.buffer.Write(p)
	return n, nil
}

// strictObject rejects duplicate fields, non-object values and trailing JSON.
func strictObject(data []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, ErrStatsUnavailable
	}
	fields := make(map[string]json.RawMessage)
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return nil, ErrStatsUnavailable
		}
		key, ok := token.(string)
		if !ok {
			return nil, ErrStatsUnavailable
		}
		if _, exists := fields[key]; exists {
			return nil, ErrStatsUnavailable
		}
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, ErrStatsUnavailable
		}
		fields[key] = value
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') {
		return nil, ErrStatsUnavailable
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, ErrStatsUnavailable
	}
	return fields, nil
}

func parseStats(data []byte) (Traffic, error) {
	bad := func() (Traffic, error) { return Traffic{}, ErrStatsUnavailable }
	if len(data) > maxStatsOutput {
		return bad()
	}
	root, err := strictObject(data)
	if err != nil || len(root) != 1 {
		return bad()
	}
	var counters []json.RawMessage
	if err := json.Unmarshal(root["stat"], &counters); err != nil || len(counters) != 2 {
		return bad()
	}
	var traffic Traffic
	seen := make(map[string]bool)
	for _, raw := range counters {
		fields, err := strictObject(raw)
		if err != nil || len(fields) < 1 || len(fields) > 2 {
			return bad()
		}
		for k := range fields {
			if k != "name" && k != "value" {
				return bad()
			}
		}
		var name string
		if err := json.Unmarshal(fields["name"], &name); err != nil || seen[name] {
			return bad()
		}
		seen[name] = true
		var value int64
		// Official CLI encoding/json omits value when the protobuf int64 is zero.
		if raw, exists := fields["value"]; exists {
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				return bad()
			}
			if err := json.Unmarshal(raw, &value); err != nil || value < 0 {
				return bad()
			}
		}
		switch name {
		case statsPattern + "uplink":
			traffic.UploadBytes = uint64(value)
		case statsPattern + "downlink":
			traffic.DownloadBytes = uint64(value)
		default:
			return bad()
		}
	}
	return traffic, nil
}
