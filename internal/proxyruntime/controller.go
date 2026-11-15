// Package proxyruntime keeps system proxy ownership in a worker process. The worker
// survives a GUI crash long enough to restore settings when its stdin pipe closes.
package proxyruntime

import (
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
	Status string `json:"status"`
	Code   string `json:"code,omitempty"`
}
type request struct {
	URI string `json:"uri"`
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
}

func New(executable string) *Controller {
	return &Controller{executable: executable, status: "disconnected"}
}
func corePath(executable string) string {
	name := "xray"
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
func (c *Controller) Connect(ctx context.Context, uri string) error {
	c.mu.Lock()
	if c.session != nil {
		c.mu.Unlock()
		return &client.ConnectionError{Code: "BUSY"}
	}
	cmd := exec.Command(c.executable, WorkerFlag)
	configureCommand(cmd)
	input, err := cmd.StdinPipe()
	if err != nil {
		c.mu.Unlock()
		return &client.ConnectionError{Code: "CONNECT_FAILED"}
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		c.mu.Unlock()
		return &client.ConnectionError{Code: "CONNECT_FAILED"}
	}
	cmd.Stderr = io.Discard
	s := &session{cmd: cmd, input: input, done: make(chan struct{}), ready: make(chan message, 1)}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		c.mu.Unlock()
		return &client.ConnectionError{Code: "CONNECT_FAILED"}
	}
	c.session = s
	c.status = "connecting"
	c.code = ""
	c.mu.Unlock()
	go c.monitor(s, output)
	if json.NewEncoder(input).Encode(request{URI: uri}) != nil {
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
		return &client.ConnectionError{Code: "CONNECT_FAILED"}
	}
}
func safeCode(code string) string {
	switch code {
	case "PROXY_FAILED", "RESTORE_FAILED", "BUSY", "UNSUPPORTED_PROFILE":
		return code
	}
	return "CONNECT_FAILED"
}
func (c *Controller) monitor(s *session, output io.ReadCloser) {
	decoder := json.NewDecoder(io.LimitReader(output, 65536))
	first := true
	stopped := false
	for {
		var msg message
		if decoder.Decode(&msg) != nil {
			break
		}
		c.mu.Lock()
		if c.session == s {
			switch msg.Status {
			case "connected":
				c.status = "connected"
				c.code = ""
			case "stopped":
				c.status = "disconnected"
				c.code = ""
				stopped = true
			case "error":
				c.status = "error"
				c.code = safeCode(msg.Code)
			}
		}
		c.mu.Unlock()
		if first {
			s.ready <- msg
			first = false
		}
	}
	_ = s.cmd.Wait()
	_ = s.input.Close()
	c.mu.Lock()
	if c.session == s {
		c.session = nil
		if !stopped && c.status != "error" {
			c.status = "error"
			c.code = "CONNECT_FAILED"
		}
	}
	c.mu.Unlock()
	if first {
		s.ready <- message{Status: "error", Code: "CONNECT_FAILED"}
	}
	close(s.done)
}
func (c *Controller) Disconnect() error {
	c.mu.Lock()
	s := c.session
	if s == nil {
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
