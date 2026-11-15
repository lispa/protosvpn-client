package client

import (
	"runtime"
	"strings"
)

type Connection struct {
	Metrics         *ConnectionMetrics `json:"metrics,omitempty"`
	Compatibility   bool               `json:"compatibility"`
	Mode            string             `json:"mode"`
	ErrorCode       string             `json:"errorCode,omitempty"`
	Status          string             `json:"status"`
	ProfileID       string             `json:"profileId"`
	EngineAvailable bool               `json:"engineAvailable"`
}
type State struct {
	SystemLocale     string     `json:"systemLocale"`
	Version          string     `json:"version"`
	Profiles         []Profile  `json:"profiles"`
	SelectedID       string     `json:"selectedId"`
	StorageStatus    string     `json:"storageStatus"`
	StorageErrorCode string     `json:"storageErrorCode,omitempty"`
	StorageMode      string     `json:"storageMode"`
	CSRFToken        string     `json:"csrfToken"`
	Connection       Connection `json:"connection"`
}
type Diagnostics struct {
	Version         string `json:"version"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	StorageMode     string `json:"storageMode"`
	EngineAvailable bool   `json:"engineAvailable"`
	ProfileCount    int    `json:"profileCount"`
}

func (s *Service) State() State {
	connection := s.connectionState()
	s.mu.RLock()
	defer s.mu.RUnlock()
	profiles := make([]Profile, 0, len(s.entries))
	for _, e := range s.entries {
		profiles = append(profiles, cloneProfile(e.profile))
	}
	return State{SystemLocale: systemLocale(), Version: "0.2.3", Profiles: profiles, SelectedID: s.selected, StorageMode: s.storageModeLocked(), StorageStatus: s.storageStatus, StorageErrorCode: s.storageErrorCode, CSRFToken: s.token, Connection: connection}
}
func (s *Service) Select(id string) error {
	if s.storageLoadFailed() {
		return ErrStoreLoad
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.profile.ID == id {
			return s.commitLocked(s.entries, id)
		}
	}
	return ErrNotFound
}
func (s *Service) Rename(id, name string) (Profile, error) {
	if s.storageLoadFailed() {
		return Profile{}, ErrStoreLoad
	}
	if !validLabel(name) {
		return Profile{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].profile.ID == id {
			entries := append([]entry(nil), s.entries...)
			entries[i].profile.Name = strings.TrimSpace(name)
			if err := s.commitLocked(entries, s.selected); err != nil {
				return Profile{}, err
			}
			return cloneProfile(entries[i].profile), nil
		}
	}
	return Profile{}, ErrNotFound
}
func (s *Service) Delete(id string) error {
	if s.storageLoadFailed() {
		return ErrStoreLoad
	}
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	connection := s.connectionState()
	if connection.ProfileID == id && connection.Status != "disconnected" {
		return ErrBusy
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.entries {
		if e.profile.ID == id {
			entries := append([]entry(nil), s.entries[:i]...)
			entries = append(entries, s.entries[i+1:]...)
			selected := s.selected
			if selected == id {
				selected = ""
				if len(entries) > 0 {
					selected = entries[0].profile.ID
				}
			}
			return s.commitLocked(entries, selected)
		}
	}
	return ErrNotFound
}
func (s *Service) Diagnostics() Diagnostics {
	available := s.connectionState().EngineAvailable
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Diagnostics{Version: "0.2.3", OS: runtime.GOOS, Arch: runtime.GOARCH, StorageMode: s.storageModeLocked(), EngineAvailable: available, ProfileCount: len(s.entries)}
}
