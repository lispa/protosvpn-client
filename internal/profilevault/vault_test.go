package profilevault

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEncryptedDiskRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	v := New(dir)
	if _, err := v.Load(); !os.IsNotExist(err) {
		t.Fatalf("first load: %v", err)
	}
	secret := []byte(`{"content":"vless://secret-credential@example.com:443"}`)
	if err := v.Save(secret); err != nil {
		t.Fatal(err)
	}
	got, err := New(dir).Load()
	if err != nil || !bytes.Equal(got, secret) {
		t.Fatalf("roundtrip: %v", err)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(b, secret) || bytes.Contains(b, []byte("vless://")) {
			t.Fatal("plaintext on disk")
		}
		info, _ := f.Info()
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
			t.Fatalf("file mode %v", info.Mode())
		}
	}
	info, _ := os.Stat(dir)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0700 {
		t.Fatalf("directory mode %v", info.Mode())
	}
}
