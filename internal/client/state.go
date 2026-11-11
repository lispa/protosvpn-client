package client

import (
	"runtime"
	"strings"
)

type Connection struct {
	Status          string `json:"status"`
	ProfileID       string `json:"profileId"`
	EngineAvailable bool   `json:"engineAvailable"`
}
type State struct {
	SystemLocale string     `json:"systemLocale"`
	Version      string     `json:"version"`
	Profiles     []Profile  `json:"profiles"`
	SelectedID   string     `json:"selectedId"`
	StorageMode  string     `json:"storageMode"`
	CSRFToken    string     `json:"csrfToken"`
	Connection   Connection `json:"connection"`
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	profiles := make([]Profile, 0, len(s.entries))
	for _, e := range s.entries {
		profiles = append(profiles, cloneProfile(e.profile))
	}
	return State{SystemLocale: systemLocale(), Version: "0.1.2", Profiles: profiles, SelectedID: s.selected, StorageMode: "memory", CSRFToken: s.token, Connection: Connection{Status: "disconnected"}}
}
func (s *Service) Select(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.profile.ID == id {
			s.selected = id
			return nil
		}
	}
	return ErrNotFound
}
func (s *Service) Rename(id, name string) (Profile, error) {
	if !validLabel(name) {
		return Profile{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].profile.ID == id {
			s.entries[i].profile.Name = strings.TrimSpace(name)
			return cloneProfile(s.entries[i].profile), nil
		}
	}
	return Profile{}, ErrNotFound
}
func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.entries {
		if e.profile.ID == id {
			copy(s.entries[i:], s.entries[i+1:])
			s.entries[len(s.entries)-1] = entry{}
			s.entries = s.entries[:len(s.entries)-1]
			if s.selected == id {
				s.selected = ""
				if len(s.entries) > 0 {
					s.selected = s.entries[0].profile.ID
				}
			}
			return nil
		}
	}
	return ErrNotFound
}
func (s *Service) Diagnostics() Diagnostics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Diagnostics{Version: "0.1.2", OS: runtime.GOOS, Arch: runtime.GOARCH, StorageMode: "memory", ProfileCount: len(s.entries)}
}
