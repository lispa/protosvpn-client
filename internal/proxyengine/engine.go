package proxyengine

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

var (
	ErrStarted = errors.New("proxyengine: engine already used or closed")
	ErrLaunch  = errors.New("proxyengine: core could not be started")
	ErrExited  = errors.New("proxyengine: core stopped unexpectedly")
	ErrStartup = errors.New("proxyengine: local proxy did not become ready")
	ErrCheck   = errors.New("proxyengine: connectivity check failed")
)

// Engine is a single-use process owner. The binary must be a trusted official Xray executable.
// All errors are intentionally static: subprocess output, URIs, paths and credentials are never returned.
type Engine struct {
	mu           sync.Mutex
	binary       string
	used, closed bool
	cmd          *exec.Cmd
	done         chan struct{}
	once         sync.Once
}

func New(binary string) *Engine { return &Engine{binary: binary, done: make(chan struct{})} }

// Done closes after the process exits, or startup fails, or Close is called before Start.
func (e *Engine) Done() <-chan struct{} { return e.done }
func (e *Engine) finish()               { e.once.Do(func() { close(e.done) }) }

// Start validates configuration and checks the local HTTP proxy protocol. It does not
// claim remote connectivity; use Check for that. ctx only controls startup, never
// the successfully started child. Startup has an additional 10-second upper bound.
func (e *Engine) Start(ctx context.Context, uri string) (string, error) {
	e.mu.Lock()
	if e.used || e.closed {
		e.mu.Unlock()
		return "", ErrStarted
	}
	e.used = true
	e.mu.Unlock()
	ok := false
	defer func() {
		if !ok {
			_ = e.Close()
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return "", ErrStartup
	}
	// Reserve then release: Xray has no inherited-listener API. A bind race fails
	// closed (Xray exits) and is never treated as successful merely on TCP connect.
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", ErrStartup
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	config, err := makeConfig(uri, port)
	if err != nil {
		return "", err
	}
	cmd := exec.Command(e.binary, "run", "-config", "stdin:")
	configureProcess(cmd)
	cmd.Stdin = bytes.NewReader(config)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return "", ErrStarted
	}
	if err = cmd.Start(); err != nil {
		e.mu.Unlock()
		return "", ErrLaunch
	}
	e.cmd = cmd
	e.mu.Unlock()
	go func() { _ = cmd.Wait(); e.finish() }()
	endpoint := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ErrStartup
		case <-e.done:
			return "", ErrExited
		default:
		}
		if localReady(ctx, endpoint) {
			// Observe that the child, not merely a listener, remains alive across readiness.
			select {
			case <-ctx.Done():
				return "", ErrStartup
			case <-e.done:
				return "", ErrExited
			case <-ticker.C:
			}
			e.mu.Lock()
			closed := e.closed
			e.mu.Unlock()
			if closed {
				return "", ErrExited
			}
			select {
			case <-e.done:
				return "", ErrExited
			default:
			}
			ok = true
			return endpoint, nil
		}
		select {
		case <-ctx.Done():
			return "", ErrStartup
		case <-e.done:
			return "", ErrExited
		case <-ticker.C:
		}
	}
}
func localReady(ctx context.Context, endpoint string) bool {
	d := net.Dialer{Timeout: 200 * time.Millisecond}
	conn, err := d.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return false
	}
	defer conn.Close()
	deadline := time.Now().Add(200 * time.Millisecond)
	if end, ok := ctx.Deadline(); ok && end.Before(deadline) {
		deadline = end
	}
	_ = conn.SetDeadline(deadline)
	// Xray HTTP inbound rejects origin-form requests with HTTP 400 and
	// Proxy-Connection: close before dispatching anything to the VLESS server.
	if _, err = io.WriteString(conn, "GET / HTTP/1.1\r\nHost: readiness.invalid\r\nConnection: close\r\n\r\n"); err != nil {
		return false
	}
	response, err := http.ReadResponse(bufio.NewReader(io.LimitReader(conn, 8192)), nil)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == 400 && response.Header.Get("Proxy-Connection") == "close"
}

// Close kills and reaps the child; it is safe concurrently with Start and itself.
func (e *Engine) Close() error {
	e.mu.Lock()
	e.closed = true
	cmd := e.cmd
	if cmd == nil {
		e.finish()
		e.mu.Unlock()
		return nil
	}
	_ = cmd.Process.Kill()
	e.mu.Unlock()
	<-e.done
	return nil
}

// Check makes a certificate-verified GET through this loopback HTTP proxy. It is
// bounded to 20 seconds, accepts only 2xx, and never follows redirects (which
// could change the intended test target). No environment proxy or direct fallback.
func Check(ctx context.Context, endpoint, target string) error {
	host, p, err := net.SplitHostPort(endpoint)
	if err != nil || host != "127.0.0.1" {
		return ErrCheck
	}
	port, err := strconv.Atoi(p)
	if err != nil || port < 1 || port > 65535 {
		return ErrCheck
	}
	u, err := url.Parse(target)
	if err != nil || u.User != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ErrCheck
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	transport := &http.Transport{Proxy: http.ProxyURL(&url.URL{Scheme: "http", Host: endpoint}), DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return ErrCheck
	}
	response, err := client.Do(req)
	if err != nil {
		return ErrCheck
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return ErrCheck
	}
	return nil
}
