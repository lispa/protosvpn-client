//go:build windows

package client

import "golang.org/x/sys/windows"

var getUserUILanguage = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

// Use the user's UI language, not regional date/number formatting.
func systemLocale() string {
	language, _, _ := getUserUILanguage.Call()
	if language&0x3ff == 0x19 {
		return "ru"
	}
	return "en"
}
