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
	runWorker(os.Stdin, os.Stdout, proxyengine.New(corePath(executable)), systemproxy.New(), checkEgress)
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
	lostParent := make(chan struct{})
	go func() { _, _ = io.Copy(io.Discard, reader); close(lostParent) }()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	startupDone := make(chan struct{})
	defer close(startupDone)
	go func() {
		select {
		case <-lostParent:
			cancel()
		case <-startupDone:
		}
	}()
	endpoint, err := e.Start(ctx, request.URI)
	request.URI = ""
	if err != nil {
		code := "CONNECT_FAILED"
		if errors.Is(err, proxyengine.ErrInvalid) {
			code = "UNSUPPORTED_PROFILE"
		}
		send("error", code)
		_ = e.Close()
		return
	}
	defer e.Close()
	if check(ctx, endpoint) != nil {
		send("error", "CONNECT_FAILED")
		return
	}
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
	if enc.Encode(message{Status: "connected"}) != nil {
		restoreUntilReleased(p, send)
		return
	}
	coreExited := false
	select {
	case <-lostParent:
	case <-e.Done():
		coreExited = true
	}
	restoreUntilReleased(p, send)
	if coreExited {
		send("error", "CONNECT_FAILED")
	} else {
		send("stopped", "")
	}
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
