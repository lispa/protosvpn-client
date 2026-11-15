package proxyruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"protosvpn.local/client/internal/proxyengine"
	"runtime"
	"testing"
	"time"
)

// This is an offline startup/readiness test, not an external server connection.
func TestOfficialSelectedCorePublicPlaintextPolicy(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-host integration fixture")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "core"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"xray", "xray-legacy"} {
		src, err := filepath.Abs(filepath.Join("..", "..", "build", "cores", "linux-amd64", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = os.Stat(src); err != nil {
			t.Skip("official integration binaries not installed")
		}
		if err = os.Symlink(src, filepath.Join(dir, "core", name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, legacy := range []bool{false, true} {
		e := proxyengine.New(selectedCorePath(filepath.Join(dir, "ProtosVPN"), legacy))
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		endpoint, err := e.Start(ctx, "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:80?security=none&type=tcp")
		cancel()
		e.Close()
		if legacy {
			if err != nil || endpoint == "" {
				t.Fatal("explicit legacy startup failed", err)
			}
		} else if !errors.Is(err, proxyengine.ErrExited) {
			t.Fatal("modern unexpectedly accepted public plaintext", err)
		}
	}
}
