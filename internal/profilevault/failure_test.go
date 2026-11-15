package profilevault

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTamperDoesNotDecrypt(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	if err := v.Save([]byte("sensitive profile")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	b, _ := os.ReadFile(path)
	b[len(b)-1] ^= 1
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Load(); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
}
func TestLostKeyIsNotRegenerated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("DPAPI has no application key file")
	}
	dir := t.TempDir()
	v := New(dir)
	if err := v.Save([]byte("credential")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	before, _ := os.ReadFile(path)
	os.Remove(filepath.Join(dir, "profiles.key"))
	if _, err := v.Load(); err == nil {
		t.Fatal("missing key accepted")
	}
	if err := v.Save([]byte("replacement")); err == nil {
		t.Fatal("lost key replaced")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("vault overwritten")
	}
}
func TestOversizeSaveDoesNotReplaceFile(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	if err := v.Save([]byte("before")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	before, _ := os.ReadFile(path)
	if err := v.Save(make([]byte, MaxBytes+1)); err == nil {
		t.Fatal("oversize accepted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed save changed file")
	}
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if f.Name() != FileName && f.Name() != "profiles.key" {
			t.Fatal("temporary file leaked")
		}
	}
}
func TestDefaultPathIndependentOfWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", t.TempDir())
	} else {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	v := Default()
	if v.dir != filepath.Join(dir, "ProtosVPN") {
		t.Fatal(v.dir)
	}
}

func TestExistingFileProtectionRestoredOnLoad(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ACL checked on native Windows separately")
	}
	dir := t.TempDir()
	v := New(dir)
	if err := v.Save([]byte("protected")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Load(); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("vault readable by other users")
	}
}
