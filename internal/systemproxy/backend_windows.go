//go:build windows

package systemproxy

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	internetOptionPerConnection   = 75
	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37
	perConnFlags                  = 1
	perConnProxyServer            = 2
	perConnProxyBypass            = 3
	perConnAutoConfigURL          = 4
	perConnFlagsUI                = 10
)

// The native union contains DWORD, pointer and FILETIME. In particular it is
// EIGHT bytes on 386 too (not uintptr-sized). uint64 has the required alignment
// of 4 on Windows/386 and 8 on amd64/arm64, producing sizes 12 and 16.
// https://learn.microsoft.com/windows/win32/api/wininet/ns-wininet-internet_per_conn_optionw
type perConnOption struct {
	option uint32
	value  uint64
}

type perConnOptionList struct {
	size        uint32
	connection  *uint16 // nil means the default LAN connection, not RAS.
	count       uint32
	optionError uint32
	options     *perConnOption
}

// Cross-compilation checks native size and offsets, not just Go field types.
// Both array bounds must be nonnegative, so any ABI mismatch fails the build.
const nativePointerSize = unsafe.Sizeof(uintptr(0))

var (
	_ [unsafe.Sizeof(perConnOption{}) - (8 + nativePointerSize)]byte
	_ [(8 + nativePointerSize) - unsafe.Sizeof(perConnOption{})]byte
	_ [unsafe.Offsetof(perConnOption{}.value) - nativePointerSize]byte
	_ [nativePointerSize - unsafe.Offsetof(perConnOption{}.value)]byte
	_ [unsafe.Sizeof(perConnOptionList{}) - (8 + 3*nativePointerSize)]byte
	_ [(8 + 3*nativePointerSize) - unsafe.Sizeof(perConnOptionList{})]byte
)

type winINetBackend struct {
	queryOption, setOption, globalFree *windows.LazyProc
}

func newBackend() backend {
	inet := windows.NewLazySystemDLL("wininet.dll")
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	b := &winINetBackend{
		queryOption: inet.NewProc("InternetQueryOptionW"),
		setOption:   inet.NewProc("InternetSetOptionW"),
		globalFree:  kernel.NewProc("GlobalFree"),
	}
	for _, p := range []*windows.LazyProc{b.queryOption, b.setOption, b.globalFree} {
		if p.Find() != nil {
			return nil
		}
	}
	return b
}

func (b *winINetBackend) query() (snapshot, error) {
	// Microsoft recommends FLAGS_UI on IE8+ to preserve the user's auto-detect
	// selection even when detection has failed. Always restore using FLAGS.
	// Older systems require falling back to FLAGS.
	s, err := b.queryFlags(perConnFlagsUI)
	if err != nil {
		return b.queryFlags(perConnFlags)
	}
	return s, nil
}

func optionString(o *perConnOption) *uint16 {
	// Read the pointer member of the native union without a uintptr round trip.
	return *(**uint16)(unsafe.Pointer(&o.value))
}

func (b *winINetBackend) queryFlags(flagsOption uint32) (snapshot, error) {
	opts := [4]perConnOption{{option: flagsOption}, {option: perConnProxyServer}, {option: perConnProxyBypass}, {option: perConnAutoConfigURL}}
	list := perConnOptionList{count: uint32(len(opts)), options: &opts[0]}
	list.size = uint32(unsafe.Sizeof(list))
	size := list.size
	// WinINet allocates each queried string with GlobalAlloc. Free even on
	// failed/partially successful queries, before attempting FLAGS fallback.
	defer func() {
		for i := 1; i < len(opts); i++ {
			if p := optionString(&opts[i]); p != nil {
				b.globalFree.Call(uintptr(unsafe.Pointer(p)))
			}
		}
	}()
	ok, _, _ := b.queryOption.Call(0, internetOptionPerConnection, uintptr(unsafe.Pointer(&list)), uintptr(unsafe.Pointer(&size)))
	runtime.KeepAlive(&list)
	runtime.KeepAlive(&opts)
	if ok == 0 {
		return snapshot{}, errQuery
	}
	return snapshot{
		flags:  uint32(opts[0].value),
		server: windows.UTF16PtrToString(optionString(&opts[1])),
		bypass: windows.UTF16PtrToString(optionString(&opts[2])),
		pac:    windows.UTF16PtrToString(optionString(&opts[3])),
	}, nil
}

func (b *winINetBackend) set(s snapshot) error {
	strings := [3]string{s.server, s.bypass, s.pac}
	var buffers [3][]uint16
	opts := [4]perConnOption{{option: perConnFlags, value: uint64(s.flags)}, {option: perConnProxyServer}, {option: perConnProxyBypass}, {option: perConnAutoConfigURL}}
	for i, v := range strings {
		buf, err := windows.UTF16FromString(v)
		if err != nil {
			return errSet
		}
		buffers[i] = buf
		*(**uint16)(unsafe.Pointer(&opts[i+1].value)) = &buf[0]
	}
	list := perConnOptionList{count: uint32(len(opts)), options: &opts[0]}
	list.size = uint32(unsafe.Sizeof(list))
	ok, _, _ := b.setOption.Call(0, internetOptionPerConnection, uintptr(unsafe.Pointer(&list)), uintptr(list.size))
	// The union's uint64 fields are not GC pointer roots. Keep the backing
	// UTF-16 arrays alive through the synchronous native call explicitly.
	runtime.KeepAlive(buffers)
	runtime.KeepAlive(&opts)
	runtime.KeepAlive(&list)
	if ok == 0 {
		return errSet
	}
	return nil
}

func (b *winINetBackend) notify() error {
	// These are notifications AFTER the real per-connection WinINet write;
	// registry edits plus notification alone would not update effective settings.
	changed, _, _ := b.setOption.Call(0, internetOptionSettingsChanged, 0, 0)
	refreshed, _, _ := b.setOption.Call(0, internetOptionRefresh, 0, 0)
	// Attempt both even if the first notification fails. Manager retains its
	// snapshot until both succeed, allowing Restore to retry notification.
	if changed == 0 || refreshed == 0 {
		return errNotify
	}
	return nil
}
