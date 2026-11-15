// Package proxyruntime keeps system proxy ownership in a worker process. The worker
// survives a GUI crash long enough to restore settings when its stdin pipe closes.
package proxyruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"protosvpn.local/client/internal/client"
)

const WorkerFlag = "--protosvpn-proxy-worker"

type message struct {
	Status  string                    `json:"status"`
	Code    string                    `json:"code,omitempty"`
	Metrics *client.ConnectionMetrics `json:"metrics,omitempty"`
}
type request struct {
	URI           string `json:"uri"`
	Compatibility bool   `json:"compatibility"`
}
type session struct {
	cmd   *exec.Cmd
	input io.WriteCloser
	done  chan struct{}
	ready chan message
}
type Controller struct {
	mu           sync.Mutex
	executable   string
	session      *session
	status, code string
	metrics      *client.ConnectionMetrics
}

func New(executable string) *Controller {
	return &Controller{executable: executable, status: "disconnected"}
}
func corePath(executable string) string {
	return selectedCorePath(executable, false)
}
func selectedCorePath(executable string, compatibility bool) string {
	name := "xray"
	if compatibility {
		name += "-legacy"
	}
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(executable), "core", name)
}
func (c *Controller) Available() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	info, err := os.Stat(corePath(c.executable))
	return err == nil && !info.IsDir()
}
func (c *Controller) Status() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status, c.code
}
func (c *Controller) Metrics() *client.ConnectionMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return client.CloneConnectionMetrics(c.metrics)
}
func (c *Controller) Connect(ctx context.Context, uri string) error {
	return c.connect(ctx, uri, false)
}
func (c *Controller) ConnectCompatibility(ctx context.Context, uri string) error {
	return c.connect(ctx, uri, true)
}
func (c *Controller) connect(ctx context.Context, uri string, compatibility bool) error {
	c.mu.Lock()
	if c.session != nil {
		c.mu.Unlock()
		return &client.ConnectionError{Code: "BUSY"}
	}
	if compatibility {
		code := ""
		if !compatibilityAllowed(uri) {
			code = "UNSUPPORTED_COMPATIBILITY"
		} else if info, err := os.Stat(selectedCorePath(c.executable, true)); err != nil || info.IsDir() {
			code = "COMPATIBILITY_UNAVAILABLE"
		}
		if code != "" {
			c.status, c.code = "error", code
			c.mu.Unlock()
			return &client.ConnectionError{Code: code}
		}
	}
	cmd := exec.Command(c.executable, WorkerFlag)
	configureCommand(cmd)
	input, err := cmd.StdinPipe()
	if err != nil {
		c.status, c.code = "error", "HELPER_START_FAILED"
		c.mu.Unlock()
		return &client.ConnectionError{Code: "HELPER_START_FAILED"}
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		c.status, c.code = "error", "HELPER_START_FAILED"
		c.mu.Unlock()
		return &client.ConnectionError{Code: "HELPER_START_FAILED"}
	}
	cmd.Stderr = io.Discard
	s := &session{cmd: cmd, input: input, done: make(chan struct{}), ready: make(chan message, 1)}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		c.status, c.code = "error", "HELPER_START_FAILED"
		c.mu.Unlock()
		return &client.ConnectionError{Code: "HELPER_START_FAILED"}
	}
	c.session = s
	c.metrics = nil
	c.status = "connecting"
	c.code = ""
	c.mu.Unlock()
	go c.monitor(s, output)
	if json.NewEncoder(input).Encode(request{URI: uri, Compatibility: compatibility}) != nil {
		input.Close()
		return &client.ConnectionError{Code: "CONNECT_FAILED"}
	}
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case msg := <-s.ready:
		if msg.Status == "connected" {
			return nil
		}
		input.Close()
		return &client.ConnectionError{Code: safeCode(msg.Code)}
	case <-ctx.Done():
		input.Close()
		return &client.ConnectionError{Code: "CONNECT_FAILED"}
	case <-timer.C:
		input.Close()
		return &client.ConnectionError{Code: "HELPER_TIMEOUT"}
	}
}
func safeCode(code string) string {
	switch code {
	case "PROXY_FAILED", "RESTORE_FAILED", "BUSY", "UNSUPPORTED_PROFILE", "ENGINE_START_FAILED", "EGRESS_CHECK_FAILED", "HELPER_START_FAILED", "HELPER_EXITED", "HELPER_TIMEOUT", "COMPATIBILITY_UNAVAILABLE", "UNSUPPORTED_COMPATIBILITY":
		return code
	}
	return "CONNECT_FAILED"
}
func (c *Controller) monitor(s *session, output io.ReadCloser) {
	// Bound each message, not the total lifetime stream: metrics are continuous.
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), 16<<10)
	first := true
	stopped := false
	for scanner.Scan() {
		var msg message
		if json.Unmarshal(scanner.Bytes(), &msg) != nil {
			break
		}
		c.mu.Lock()
		if c.session == s {
			switch msg.Status {
			case "connected":
				c.status = "connected"
				c.code = ""
				c.metrics = client.CloneConnectionMetrics(msg.Metrics)
			case "metrics":
				if c.status == "connected" {
					c.metrics = client.CloneConnectionMetrics(msg.Metrics)
				}
			case "stopped":
				c.status = "disconnected"
				c.code = ""
				stopped = true
				c.metrics = nil
			case "error":
				c.status = "error"
				c.code = safeCode(msg.Code)
			}
		}
		c.mu.Unlock()
		if first && (msg.Status == "connected" || msg.Status == "error") {
			s.ready <- msg
			first = false
		}
	}
	// Unblock the worker and its parent-liveness watcher before waiting.
	_ = output.Close()
	_ = s.input.Close()
	_ = s.cmd.Wait()
	c.mu.Lock()
	if c.session == s {
		c.session = nil
		c.metrics = nil
		if !stopped && c.status != "error" {
			c.status = "error"
			c.code = "HELPER_EXITED"
		}
	}
	c.mu.Unlock()
	if first {
		s.ready <- message{Status: "error", Code: "HELPER_EXITED"}
	}
	close(s.done)
}
func (c *Controller) Disconnect() error {
	c.mu.Lock()
	s := c.session
	if s == nil {
		c.metrics = nil
		c.status = "disconnected"
		c.code = ""
		c.mu.Unlock()
		return nil
	}
	c.status = "disconnecting"
	c.mu.Unlock()
	_ = s.input.Close()
	timer := time.NewTimer(12 * time.Second)
	defer timer.Stop()
	select {
	case <-s.done:
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.status == "error" && c.code == "RESTORE_FAILED" {
			return &client.ConnectionError{Code: "RESTORE_FAILED"}
		}
		c.status = "disconnected"
		c.code = ""
		return nil
	case <-timer.C:
		c.mu.Lock()
		if c.session == s {
			c.status = "error"
			c.code = "RESTORE_FAILED"
		}
		c.mu.Unlock()
		// Never terminate a worker that may still own proxy settings. It keeps retrying.
		return &client.ConnectionError{Code: "RESTORE_FAILED"}
	}
}
