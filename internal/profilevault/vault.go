// Package profilevault stores opaque profile records encrypted at rest.
package profilevault

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

const MaxBytes = 16 << 20
const FileName = "profiles.vault"

var ErrProtection = errors.New("profile storage protection failed")

type Vault struct {
	dir     string
	initErr error
}

func New(dir string) *Vault { return &Vault{dir: dir} }
func Default() *Vault {
	dir, err := os.UserConfigDir()
	if err != nil {
		return &Vault{initErr: ErrProtection}
	}
	return New(filepath.Join(dir, "ProtosVPN"))
}
func (v *Vault) Load() ([]byte, error) {
	if v.initErr != nil {
		return nil, ErrProtection
	}
	raw, err := readBounded(filepath.Join(v.dir, FileName), MaxBytes+1024)
	if err != nil {
		return nil, err
	}
	if err := v.restrictFile(filepath.Join(v.dir, FileName)); err != nil {
		return nil, ErrProtection
	}
	return v.unprotect(raw)
}
func (v *Vault) Save(data []byte) error {
	if v.initErr != nil || len(data) > MaxBytes {
		return ErrProtection
	}
	if err := v.prepare(); err != nil {
		return ErrProtection
	}
	raw, err := v.protect(data)
	if err != nil {
		return ErrProtection
	}
	return atomicWrite(filepath.Join(v.dir, FileName), raw)
}
func readBounded(path string, max int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrProtection
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(b)) > max {
		return nil, ErrProtection
	}
	return b, nil
}
func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".profiles-*")
	if err != nil {
		return ErrProtection
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	if f.Chmod(0600) != nil {
		return ErrProtection
	}
	if _, err = f.Write(data); err != nil {
		return ErrProtection
	}
	if f.Sync() != nil {
		return ErrProtection
	}
	if f.Close() != nil {
		return ErrProtection
	}
	if err = os.Rename(name, path); err != nil {
		return ErrProtection
	}
	return nil
}
