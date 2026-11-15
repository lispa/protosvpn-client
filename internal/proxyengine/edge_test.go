package proxyengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStartupCancelled(t *testing.T) {
	e := New(coreBinary(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.Start(ctx, baseURI); err == nil {
		t.Fatal("cancelled startup succeeded")
	}
	select {
	case <-e.Done():
	case <-time.After(time.Second):
		t.Fatal("cancellation leaked engine")
	}
}
func TestCloseConcurrent(t *testing.T) {
	e := New(coreBinary(t))
	if _, err := e.Start(context.Background(), baseURI); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if e.Close() != nil {
				t.Error("concurrent close failed")
			}
		})
	}
	wg.Wait()
	select {
	case <-e.Done():
	default:
		t.Fatal("process not reaped")
	}
}
func TestCheckHTTPStatusAndNoRedirect(t *testing.T) {
	// Check must actually use its configured proxy even for localhost targets;
	// this tiny HTTP fixture is the proxy, and validates absolute-form requests.
	var seen string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.RequestURI
		w.Header().Set("Location", "http://127.0.0.1:1/secret")
		w.WriteHeader(302)
	}))
	defer proxy.Close()
	if Check(context.Background(), strings.TrimPrefix(proxy.URL, "http://"), "http://localhost/test") == nil {
		t.Fatal("redirect accepted")
	}
	if seen != "http://localhost/test" {
		t.Fatal("proxy bypassed")
	}
}
func TestConfigRejectsMalformed(t *testing.T) {
	for _, uri := range []string{baseURI + "/path", baseURI + "?security=tls&security=none", baseURI + "?security=reality&sni=example.org&fp=chrome&pbk=AAAA", baseURI + "?security=tls&alpn=h3", baseURI + "?security=tls&fp=chrome&sid=abcd", baseURI + "?type=ws&mode=auto", baseURI + "?type=http", baseURI + "?type=xhttp&mode=invalid", baseURI + "?type=grpc&mode=invalid", baseURI + "?type=ws&host=bad%2Fhost", baseURI + "?security=tls&sni=bad%20host", baseURI + "?encryption=", baseURI + "?type=ws&path=relative", "vless://" + testID + "@2001:db8::1:443"} {
		if _, err := makeConfig(uri, 12345); err != ErrInvalid {
			t.Fatal("malformed config accepted")
		}
	}
	for _, uri := range []string{"vless://" + testID + "@[::1]:443?security=tls", baseURI + "?type=grpc&serviceName=demo%2Fservice", baseURI + "?type=tcp&headerType=none"} {
		if _, err := makeConfig(uri, 12345); err != nil {
			t.Fatal("valid config rejected")
		}
	}
}
