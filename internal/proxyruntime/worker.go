package proxyruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"protosvpn.local/client/internal/client"
	"protosvpn.local/client/internal/proxyengine"
	"protosvpn.local/client/internal/systemproxy"
)

type engine interface {
	Start(context.Context, string) (string, error)
	Close() error
	Done() <-chan struct{}
}
type proxyManager interface {
	Enable(string) error
	Restore() error
}

// MaybeRun must run before Wails initialises a window; no secrets appear in args.
func MaybeRun() bool {
	if len(os.Args) != 2 || os.Args[1] != WorkerFlag {
		return false
	}
	executable, err := os.Executable()
	if err != nil {
		return true
	}
	runWorkerWithFactory(os.Stdin, os.Stdout, func(compatibility bool) engine {
		return proxyengine.New(selectedCorePath(executable, compatibility))
	}, systemproxy.New(), checkEgress)
	return true
}
func checkEgress(ctx context.Context, endpoint string) error {
	proxyURL := &url.URL{Scheme: "http", Host: endpoint}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	h := &http.Client{Transport: transport, Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.gstatic.com/generate_204", nil)
	if err != nil {
		return err
	}
	res, err := h.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 204 {
		return errors.New("egress check failed")
	}
	return nil
}
func runWorker(input io.Reader, output io.Writer, e engine, p proxyManager, check func(context.Context, string) error) {
	runWorkerWithFactory(input, output, func(bool) engine { return e }, p, check)
}
func runWorkerWithFactory(input io.Reader, output io.Writer, newEngine func(bool) engine, p proxyManager, check func(context.Context, string) error) {
	enc := json.NewEncoder(output)
	send := func(status, code string) { _ = enc.Encode(message{Status: status, Code: code}) }
	reader := bufio.NewReader(io.LimitReader(input, (1<<20)+1))
	line, err := reader.ReadBytes('\n')
	if err != nil || len(line) > (1<<20) {
		send("error", "CONNECT_FAILED")
		return
	}
	var request request
	if json.Unmarshal(line, &request) != nil || len(request.URI) == 0 {
		send("error", "CONNECT_FAILED")
		return
	}
	if request.Compatibility && !compatibilityAllowed(request.URI) {
		send("error", "UNSUPPORTED_COMPATIBILITY")
		return
	}
	e := newEngine(request.Compatibility)
	lostParent := make(chan struct{})
	lifetimeCtx, lifetimeCancel := context.WithCancel(context.Background())
	defer lifetimeCancel()
	go func() { _, _ = io.Copy(io.Discard, reader); close(lostParent) }()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	startupDone := make(chan struct{})
	defer close(startupDone)
	go func() {
		select {
		case <-lostParent:
			cancel()
			lifetimeCancel()
		case <-startupDone:
		}
	}()
	endpoint, err := e.Start(ctx, request.URI)
	request.URI = ""
	if err != nil {
		code := "ENGINE_START_FAILED"
		if errors.Is(err, proxyengine.ErrInvalid) {
			code = "UNSUPPORTED_PROFILE"
		}
		send("error", code)
		_ = e.Close()
		return
	}
	defer e.Close()
	checkStarted := time.Now()
	if check(ctx, endpoint) != nil {
		send("error", "EGRESS_CHECK_FAILED")
		return
	}
	latency := time.Since(checkStarted).Milliseconds()
	if ctx.Err() != nil {
		send("error", "CONNECT_FAILED")
		return
	}
	if p.Enable(endpoint) != nil {
		// Enable may fail after partial application; always attempt to restore.
		restoreUntilReleased(p, send)
		send("error", "PROXY_FAILED")
		return
	}
	// Always unwind settings on GUI EOF, core exit or an output-pipe failure.
	baseMetrics := &client.ConnectionMetrics{StartedAt: time.Now().UTC().Format(time.RFC3339Nano), LatencyMs: &latency}
	var sampler trafficSampler
	readMetrics := func() *client.ConnectionMetrics {
		if provider, ok := e.(interface {
			Stats(context.Context) (proxyengine.Traffic, error)
		}); ok {
			statsCtx, statsCancel := context.WithTimeout(lifetimeCtx, time.Second)
			traffic, statsErr := provider.Stats(statsCtx)
			statsCancel()
			if statsErr == nil {
				return sampler.sample(baseMetrics, traffic.UploadBytes, traffic.DownloadBytes, time.Now())
			}
		}
		return client.CloneConnectionMetrics(baseMetrics)
	}
	if enc.Encode(message{Status: "connected", Metrics: readMetrics()}) != nil {
		restoreUntilReleased(p, send)
		return
	}
	coreExited := false
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
connected:
	for {
		select {
		case <-lostParent:
			break connected
		case <-e.Done():
			coreExited = true
			break connected
		case <-ticker.C:
			if enc.Encode(message{Status: "metrics", Metrics: readMetrics()}) != nil {
				break connected
			}
		}
	}
	restoreUntilReleased(p, send)
	if coreExited {
		send("error", "CONNECT_FAILED")
	} else {
		send("stopped", "")
	}
}
func compatibilityAllowed(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "vless" {
		return false
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return false
	}
	return (q.Get("security") == "" || q.Get("security") == "none") && (q.Get("encryption") == "" || q.Get("encryption") == "none")
}
func restoreUntilReleased(p proxyManager, send func(string, string)) {
	notified := false
	for {
		err := p.Restore()
		if err == nil || errors.Is(err, systemproxy.ErrUnsupported) {
			return
		}
		if errors.Is(err, systemproxy.ErrConflict) {
			// A bypass/PAC-only external edit may still point to our local core.
			// Do not kill it just because restoration refuses the external change.
			if owner, ok := p.(interface{ OwnsEndpoint() (bool, error) }); ok {
				active, queryErr := owner.OwnsEndpoint()
				if queryErr == nil && !active {
					return
				}
			}
		}
		if !notified {
			send("error", "RESTORE_FAILED")
			notified = true
		}
		// A surviving helper keeps the local proxy alive while retrying restoration.
		time.Sleep(2 * time.Second)
	}
}
