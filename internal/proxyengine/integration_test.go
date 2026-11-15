package proxyengine

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func coreBinary(t *testing.T) string {
	t.Helper()
	p := os.Getenv("XRAY_TEST_BINARY")
	if p == "" {
		p = filepath.Join("..", "..", "build", "cores", runtime.GOOS+"-"+runtime.GOARCH, "xray")
		if runtime.GOOS == "windows" {
			p += ".exe"
		}
	}
	p, _ = filepath.Abs(p)
	if _, err := os.Stat(p); err != nil {
		t.Skip("official Xray not installed; set XRAY_TEST_BINARY")
	}
	return p
}
func unusedPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
func startServer(t *testing.T, binary, network string, port int) {
	t.Helper()
	stream := map[string]any{"network": network, "security": "none"}
	switch network {
	case "ws", "httpupgrade", "xhttp":
		stream[network+"Settings"] = map[string]any{"path": "/test"}
	case "grpc":
		stream["grpcSettings"] = map[string]any{"serviceName": "test"}
	}
	config := map[string]any{"log": map[string]any{"loglevel": "none", "access": "none", "error": "none"}, "inbounds": []any{map[string]any{"listen": "127.0.0.1", "port": port, "protocol": "vless", "settings": map[string]any{"clients": []any{map[string]any{"id": testID}}, "decryption": "none"}, "streamSettings": stream}}, "outbounds": []any{map[string]any{"protocol": "freedom", "settings": map[string]any{"finalRules": []any{map[string]any{"action": "allow", "ip": []string{"127.0.0.1/32"}}}}}}}
	b, _ := json.Marshal(config)
	cmd := exec.Command(binary, "run", "-config", "stdin:")
	configureProcess(cmd)
	cmd.Stdin = bytes.NewReader(b)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal("server launch failed")
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	t.Cleanup(func() { _ = cmd.Process.Kill(); <-done })
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			t.Fatal("server exited")
		default:
		}
		c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 100*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server not ready")
}
func TestOfficialCoreEndToEnd(t *testing.T) {
	binary := coreBinary(t)
	for _, network := range []string{"raw", "ws", "grpc", "httpupgrade", "xhttp"} {
		t.Run(network, func(t *testing.T) {
			var hits atomic.Int32
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(204) }))
			defer target.Close()
			port := unusedPort(t)
			startServer(t, binary, network, port)
			q := url.Values{"type": {network}, "security": {"none"}}
			switch network {
			case "ws", "httpupgrade", "xhttp":
				q.Set("path", "/test")
			case "grpc":
				q.Set("serviceName", "test")
			}
			engine := New(binary)
			defer engine.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			endpoint, err := engine.Start(ctx, "vless://"+testID+"@127.0.0.1:"+strconv.Itoa(port)+"?"+q.Encode())
			cancel()
			if err != nil {
				t.Fatal(err)
			}
			// Cancellation after startup must never kill the running process.
			time.Sleep(50 * time.Millisecond)
			select {
			case <-engine.Done():
				t.Fatal("startup cancellation killed process")
			default:
			}
			if err = Check(context.Background(), endpoint, target.URL); err != nil {
				t.Fatal("real VLESS round trip failed:", err)
			}
			if hits.Load() != 1 {
				t.Fatal("target was not reached exactly once")
			}
			tlsTarget := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
			tlsTarget.Config.ErrorLog = log.New(io.Discard, "", 0)
			tlsTarget.StartTLS()
			defer tlsTarget.Close()
			if Check(context.Background(), endpoint, tlsTarget.URL) == nil {
				t.Fatal("untrusted TLS certificate accepted")
			}
			if _, err = engine.Start(context.Background(), baseURI); err != ErrStarted {
				t.Fatal("engine reused")
			}
			// Kill the actual child, not Close, to exercise unexpected-exit notification.
			engine.mu.Lock()
			_ = engine.cmd.Process.Kill()
			engine.mu.Unlock()
			select {
			case <-engine.Done():
			case <-time.After(3 * time.Second):
				t.Fatal("unexpected exit not detected")
			}
			if Check(context.Background(), endpoint, target.URL) == nil {
				t.Fatal("dead proxy check succeeded")
			}
			if engine.Close() != nil || engine.Close() != nil {
				t.Fatal("close failed")
			}
		})
	}
}
func TestOfficialConfigVariants(t *testing.T) {
	binary := coreBinary(t)
	key := "7QE1tJGSJMDcRJbYvHWuHyWvGzUq4cST4M2SAgU6FW0"
	for _, q := range []string{"type=tcp&security=tls&sni=example.org&fp=chrome&flow=xtls-rprx-vision", "type=raw&security=reality&sni=example.org&fp=chrome&pbk=" + key + "&sid=abcd&flow=xtls-rprx-vision", "type=grpc&security=tls&sni=example.org&serviceName=test&authority=example.org&mode=multi", "type=xhttp&security=tls&sni=example.org&mode=stream-one&host=example.org"} {
		b, err := makeConfig(baseURI+"?"+q, unusedPort(t))
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binary, "run", "-test", "-config", "stdin:")
		configureProcess(cmd)
		cmd.Stdin = bytes.NewReader(b)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err = cmd.Run(); err != nil {
			t.Fatal("official core rejected configuration variant")
		}
	}
}
func TestInvalidURIsSanitized(t *testing.T) {
	for _, uri := range []string{"vless://secret:password@example.org:443", "vless://" + testID + "@example.org:0", baseURI + "?unknown=secret", baseURI + "?security=tls&sni=secret%0Ahost", baseURI + "?type=raw&pbk=secret"} {
		e := New("unused")
		_, err := e.Start(context.Background(), uri)
		if err != ErrInvalid || strings.Contains(err.Error(), "secret") {
			t.Fatal("unsafe validation")
		}
		select {
		case <-e.Done():
		default:
			t.Fatal("failed startup not completed")
		}
	}
}
