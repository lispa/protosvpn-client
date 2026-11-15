//go:build !windows

package profilevault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"os"
	"path/filepath"
)

// The local key is file-permission protected, not an OS keyring.
func (v *Vault) restrictFile(path string) error { return os.Chmod(path, 0600) }
func (v *Vault) prepare() error {
	if err := os.MkdirAll(v.dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(v.dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ErrProtection
	}
	return os.Chmod(v.dir, 0700)
}
func (v *Vault) key(create bool) ([]byte, error) {
	if err := v.prepare(); err != nil {
		return nil, err
	}
	path := filepath.Join(v.dir, "profiles.key")
	b, err := readBounded(path, 32)
	if err == nil {
		if len(b) != 32 || os.Chmod(path, 0600) != nil {
			return nil, ErrProtection
		}
		return b, nil
	}
	if !os.IsNotExist(err) || !create {
		return nil, ErrProtection
	}
	// Never replace a lost key while an encrypted vault exists.
	if _, err := os.Lstat(filepath.Join(v.dir, FileName)); !os.IsNotExist(err) {
		return nil, ErrProtection
	}
	b = make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return nil, ErrProtection
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, ErrProtection
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(path)
		}
	}()
	if _, err = f.Write(b); err != nil {
		return nil, ErrProtection
	}
	if f.Sync() != nil || f.Close() != nil {
		return nil, ErrProtection
	}
	ok = true
	return b, nil
}
func (v *Vault) aead(create bool) (cipher.AEAD, error) {
	key, err := v.key(create)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrProtection
	}
	return cipher.NewGCM(block)
}
func (v *Vault) protect(data []byte) ([]byte, error) {
	a, err := v.aead(true)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, ErrProtection
	}
	out := append([]byte("PVG1"), nonce...)
	return a.Seal(out, nonce, data, []byte("PVG1")), nil
}
func (v *Vault) unprotect(raw []byte) ([]byte, error) {
	a, err := v.aead(false)
	if err != nil {
		return nil, ErrProtection
	}
	n := a.NonceSize()
	if len(raw) < 4+n+a.Overhead() || string(raw[:4]) != "PVG1" {
		return nil, ErrProtection
	}
	b, err := a.Open(nil, raw[4:4+n], raw[4+n:], raw[:4])
	if err != nil {
		return nil, ErrProtection
	}
	return b, nil
}
