package proxyengine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOfficialStats(t *testing.T) {
	for _, core := range []string{"xray", "xray-legacy"} {
		t.Run(core, func(t *testing.T) {
			name := core
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			binary, _ := filepath.Abs(filepath.Join("..", "..", "build", "cores", runtime.GOOS+"-"+runtime.GOARCH, name))
			if _, err := os.Stat(binary); err != nil {
				t.Skip("official test core not installed")
			}
			port := unusedPort(t)
			startServer(t, binary, "raw", port)
			payload := strings.Repeat("x", 65536)
			target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if err != nil || string(b) != payload {
					t.Error("upload lost")
				}
				_, _ = io.WriteString(w, payload)
			}))
			defer target.Close()
			e := New(binary)
			defer e.Close()
			startup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			endpoint, err := e.Start(startup, "vless://"+testID+"@127.0.0.1:"+strconv.Itoa(port))
			cancel()
			if err != nil {
				t.Fatal(err)
			}
			before, err := e.Stats(context.Background())
			if err != nil {
				t.Fatal("initial stats:", err)
			}
			if before != (Traffic{}) {
				t.Fatal("readiness generated remote traffic")
			}
			proxy, _ := url.Parse("http://" + endpoint)
			roots := x509.NewCertPool()
			roots.AddCert(target.Certificate())
			transport := &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true, TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
			var previous Traffic
			for range 2 {
				response, err := client.Post(target.URL, "application/octet-stream", strings.NewReader(payload))
				if err != nil {
					t.Fatal("roundtrip failed")
				}
				b, err := io.ReadAll(response.Body)
				response.Body.Close()
				if err != nil || string(b) != payload {
					t.Fatal("download lost")
				}
				current, err := e.Stats(context.Background())
				deadline := time.Now().Add(2 * time.Second)
				// Core copy-loop accounting may publish the last uplink buffer after
				// the HTTP response is consumed; wait for that asynchronous update.
				for err == nil && (current.UploadBytes < previous.UploadBytes+uint64(len(payload)) || current.DownloadBytes < previous.DownloadBytes+uint64(len(payload))) && time.Now().Before(deadline) {
					time.Sleep(10 * time.Millisecond)
					current, err = e.Stats(context.Background())
				}
				if err != nil {
					t.Fatal("live stats:", err)
				}
				if current.UploadBytes < previous.UploadBytes+uint64(len(payload)) || current.DownloadBytes < previous.DownloadBytes+uint64(len(payload)) {
					t.Fatalf("counters failed to increase: before=%+v after=%+v", previous, current)
				}
				repeated, err := e.Stats(context.Background())
				if err != nil || repeated != current {
					t.Fatalf("query reset counters: %+v %+v %v", current, repeated, err)
				}
				previous = current
			}
			t.Logf("real cumulative outbound bytes: upload=%d download=%d", previous.UploadBytes, previous.DownloadBytes)
			ctx, cancelStats := context.WithCancel(context.Background())
			cancelStats()
			if _, err := e.Stats(ctx); err != ErrStatsUnavailable {
				t.Fatal("cancelled stats accepted")
			}
			var wg sync.WaitGroup
			for range 4 {
				wg.Add(1)
				go func() { defer wg.Done(); _, _ = e.Stats(context.Background()) }()
			}
			e.Close()
			wg.Wait()
			if _, err := e.Stats(context.Background()); err != ErrStatsUnavailable {
				t.Fatal("closed stats accepted")
			}
		})
	}
}
