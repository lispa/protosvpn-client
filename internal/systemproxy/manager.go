// Package systemproxy temporarily owns the current user's default LAN WinINet
// proxy. It does not configure WinHTTP, dial-up connections, or a TUN device.
// Managers must be kept alive and Restore called before their local proxy exits.
package systemproxy

import (
	"errors"
	"net"
	"net/netip"
	"strconv"
	"sync"
)

var (
	ErrUnsupported = errors.New("system proxy is unsupported on this platform")
	ErrConflict    = errors.New("system proxy settings changed externally; refusing to overwrite")
	errEndpoint    = errors.New("system proxy endpoint must be a literal loopback IP and port 1-65535")
	errQuery       = errors.New("could not query system proxy settings")
	errSet         = errors.New("could not update system proxy settings")
	errNotify      = errors.New("could not refresh system proxy settings")
	errActive      = errors.New("system proxy already owned; restore before enabling again")
)

type snapshot struct {
	flags               uint32
	server, bypass, pac string
}

type backend interface {
	query() (snapshot, error)
	set(snapshot) error
	notify() error
}

// Manager serializes its operations. Do not copy a Manager after first use.
// External applications are not synchronized by this mutex; WinINet offers no
// atomic compare-and-swap. Restore checks ownership immediately before writing.
// The zero value is unsupported; use New.
type Manager struct {
	mu       sync.Mutex
	backend  backend
	original *snapshot
	owned    snapshot
	// A failed multi-option WinINet write can leave a subset applied. Retain
	// both boundaries until readback succeeds, so cleanup remains retryable.
	uncertain *snapshot
}

func New() *Manager { return &Manager{backend: newBackend()} }

// Available reports whether this manager has a platform backend. It does not
// imply permission to override a policy enforced by the system administrator.
func (m *Manager) Available() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.backend != nil }

// Enable snapshots all four LAN connection options, then disables PAC and
// auto-detection while directing HTTP/HTTPS to endpoint. Only literal IPv4 or
// IPv6 loopback addresses are accepted (no DNS, URLs, credentials, or zones).
// Even on error, call Restore: a WinINet write/notification can partially succeed.
func (m *Manager) Enable(endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.backend == nil {
		return ErrUnsupported
	}
	if !validEndpoint(endpoint) {
		return errEndpoint
	}
	if m.original != nil {
		return errActive
	}
	old, err := m.backend.query()
	if err != nil {
		return errQuery
	}
	m.original = &old
	m.owned = old
	if err := m.write(snapshot{flags: 3, server: endpoint}); err != nil {
		return err
	}
	if err := m.backend.notify(); err != nil {
		return errNotify
	}
	return nil
}

// Restore restores the exact original flags and strings, but only while the
// current settings still match this manager's last write. Conflicts retain the
// snapshot for an explicit later retry; they never expose proxy/PAC strings.
// A successful Restore is idempotent.
func (m *Manager) Restore() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.backend == nil {
		return ErrUnsupported
	}
	if m.original == nil {
		return nil
	}
	current, err := m.backend.query()
	if err != nil {
		return errQuery
	}
	if m.uncertain != nil && transition(current, m.owned, *m.uncertain) {
		m.owned = current
		m.uncertain = nil
	}
	if current != m.owned {
		return ErrConflict
	}
	if current != *m.original {
		if err := m.write(*m.original); err != nil {
			return err
		}
	}
	if err := m.backend.notify(); err != nil {
		return errNotify
	}
	m.original = nil
	m.uncertain = nil
	return nil
}

// write records an exact readback after a failed call. Only values from the
// attempted transition may be considered ours; an unrelated value conflicts.
// WinINet has no transaction/CAS, so indistinguishable concurrent writes of the
// same values cannot be detected (including during this recovery window).
func (m *Manager) write(target snapshot) error {
	m.uncertain = &target
	if err := m.backend.set(target); err != nil {
		if observed, qerr := m.backend.query(); qerr == nil {
			if transition(observed, m.owned, target) {
				m.owned = observed
			}
			m.uncertain = nil
		}
		return errSet
	}
	m.owned = target
	m.uncertain = nil
	return nil
}

func transition(s, from, to snapshot) bool {
	return (s.flags == from.flags || s.flags == to.flags) &&
		(s.server == from.server || s.server == to.server) &&
		(s.bypass == from.bypass || s.bypass == to.bypass) &&
		(s.pac == from.pac || s.pac == to.pac)
}

func validEndpoint(endpoint string) bool {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil || net.JoinHostPort(host, port) != endpoint {
		return false
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || ip.Zone() != "" || !ip.IsLoopback() {
		return false
	}
	if port == "" {
		return false
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(port, 10, 16)
	return err == nil && n != 0
}
