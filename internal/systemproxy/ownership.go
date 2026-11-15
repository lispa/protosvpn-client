package systemproxy

// OwnsEndpoint reports whether WinINet still points at a loopback proxy involved
// in this manager's transaction. A flags/bypass/PAC conflict must not cause the
// helper to stop the local core while Windows still uses it. Query failures are
// returned rather than interpreted as permission to stop the core.
func (m *Manager) OwnsEndpoint() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.backend == nil {
		return false, ErrUnsupported
	}
	if m.original == nil {
		return false, nil
	}
	current, err := m.backend.query()
	if err != nil {
		return false, errQuery
	}
	if current.flags&2 == 0 {
		return false, nil
	}
	if validEndpoint(m.owned.server) && current.server == m.owned.server {
		return true, nil
	}
	if m.uncertain != nil && validEndpoint(m.uncertain.server) && current.server == m.uncertain.server {
		return true, nil
	}
	return false, nil
}
