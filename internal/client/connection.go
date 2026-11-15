package client

import (
	"context"
	"errors"
)

var ErrEngineUnavailable = errors.New("engine unavailable")
var ErrUnsupportedProfile = errors.New("unsupported profile")
var ErrBusy = errors.New("connection busy")

// Connector owns the privileged-free worker; never receives profile metadata over stdout.
type Connector interface {
	Available() bool
	Connect(context.Context, string) error
	Disconnect() error
	Status() (status, code string)
}

type ConnectionError struct{ Code string }

func (e *ConnectionError) Error() string { return e.Code }

func NewWithConnector(c Connector) *Service { s := New(); s.connector = c; return s }

// Serialize mutations while starting/stopping a connection without blocking state reads.
func (s *Service) Connect(ctx context.Context, id string) error {
	if !s.connectMu.TryLock() {
		return ErrBusy
	}
	defer s.connectMu.Unlock()
	if s.connector == nil || !s.connector.Available() {
		return ErrEngineUnavailable
	}
	s.mu.RLock()
	var found *entry
	for _, e := range s.entries {
		if e.profile.ID == id {
			copy := e
			found = &copy
			break
		}
	}
	s.mu.RUnlock()
	if found == nil {
		return ErrNotFound
	}
	if found.profile.Protocol != "vless" {
		return ErrUnsupportedProfile
	}
	status, _ := s.connector.Status()
	if status == "connected" || status == "connecting" || status == "disconnecting" {
		return ErrBusy
	}
	s.mu.Lock()
	s.activeID = id
	s.mu.Unlock()
	return s.connector.Connect(ctx, found.secret)
}
func (s *Service) Disconnect() error {
	if !s.connectMu.TryLock() {
		return ErrBusy
	}
	defer s.connectMu.Unlock()
	if s.connector == nil || !s.connector.Available() {
		return ErrEngineUnavailable
	}
	err := s.connector.Disconnect()
	if err == nil {
		s.mu.Lock()
		s.activeID = ""
		s.mu.Unlock()
	}
	return err
}
func (s *Service) Close() error {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	if s.connector == nil {
		return nil
	}
	return s.connector.Disconnect()
}
func (s *Service) connectionState() Connection {
	c := Connection{Status: "disconnected", Mode: "system-proxy"}
	if s.connector == nil {
		return c
	}
	c.EngineAvailable = s.connector.Available()
	c.Status, c.ErrorCode = s.connector.Status()
	s.mu.RLock()
	if c.Status != "disconnected" && (c.Status != "error" || c.ErrorCode == "RESTORE_FAILED") {
		c.ProfileID = s.activeID
	}
	s.mu.RUnlock()
	return c
}
