//go:build windows

package profilevault

import "golang.org/x/sys/windows"

// Enforce a protected, current-user-only DACL on an existing vault file too.
// Chmod(0600) alone does not establish equivalent Windows ACL permissions.
func (v *Vault) restrictFile(path string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return ErrProtection
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return ErrProtection
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return ErrProtection
	}
	if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		return ErrProtection
	}
	return nil
}
