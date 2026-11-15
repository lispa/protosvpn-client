//go:build windows

package profilevault

import (
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

// DPAPI is current-user scoped (never CRYPTPROTECT_LOCAL_MACHINE), with no UI
// and no plaintext fallback. The protected directory grants only this user access.
func (v *Vault) prepare() error {
	if err := os.MkdirAll(v.dir, 0700); err != nil {
		return ErrProtection
	}
	info, err := os.Lstat(v.dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ErrProtection
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return ErrProtection
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return ErrProtection
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return ErrProtection
	}
	return windows.SetNamedSecurityInfo(v.dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}
func dpapi(data []byte, encrypt bool) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrProtection
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	var err error
	if encrypt {
		err = windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	if err != nil {
		return nil, ErrProtection
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	if out.Size > MaxBytes+1024 {
		return nil, ErrProtection
	}
	return append([]byte(nil), unsafe.Slice(out.Data, int(out.Size))...), nil
}
func (v *Vault) protect(data []byte) ([]byte, error) {
	b, err := dpapi(data, true)
	if err != nil {
		return nil, err
	}
	return append([]byte("PVD1"), b...), nil
}
func (v *Vault) unprotect(raw []byte) ([]byte, error) {
	if len(raw) < 5 || string(raw[:4]) != "PVD1" {
		return nil, ErrProtection
	}
	if err := v.prepare(); err != nil {
		return nil, ErrProtection
	}
	return dpapi(raw[4:], false)
}
