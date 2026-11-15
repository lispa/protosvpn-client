//go:build windows

package systemproxy

import (
	"testing"
	"unsafe"
)

// This test never reads or changes host proxy settings.
func TestWinINetABI(t *testing.T) {
	var o perConnOption
	var l perConnOptionList
	ptr := unsafe.Sizeof(uintptr(0))
	wantOption, wantValue, wantList := uintptr(12), uintptr(4), uintptr(20)
	if ptr == 8 {
		wantOption, wantValue, wantList = 16, 8, 32
	}
	if unsafe.Sizeof(o) != wantOption || unsafe.Offsetof(o.value) != wantValue || unsafe.Sizeof(l) != wantList {
		t.Fatal("WinINet ABI mismatch")
	}
	if unsafe.Offsetof(l.connection) != ptr || unsafe.Offsetof(l.options) != wantList-ptr {
		t.Fatal("WinINet list alignment mismatch")
	}
	if !New().Available() {
		t.Fatal("Windows backend unavailable")
	}
}
