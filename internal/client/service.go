package client

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"protosvpn.local/client/internal/profilevault"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 1 << 20

var ErrInvalid = errors.New("invalid or unsupported profile")
var ErrNotFound = errors.New("profile not found")

type Profile struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Protocol     string   `json:"protocol"`
	Host         string   `json:"host"`
	Port         int      `json:"port"`
	Transport    string   `json:"transport"`
	Security     string   `json:"security"`
	RequiresAuth bool     `json:"requiresAuth"`
	Warnings     []string `json:"warnings"`
}
type entry struct {
	profile Profile
	secret  string
}
type Service struct {
	store               Store
	storageStatus       string
	storageErrorCode    string
	loadFailed          bool
	connectMu           sync.Mutex
	connector           Connector
	activeID            string
	activeCompatibility bool
	mu                  sync.RWMutex
	entries             []entry
	selected            string
	token               string
}

func randomID() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("secure randomness unavailable")
	}
	return hex.EncodeToString(b[:])
}
func New() *Service { return &Service{token: randomID(), entries: []entry{}, storageStatus: "empty"} }

// Store exchanges only bounded, opaque profile documents. Implementations must protect
// credentials at rest and commit Save atomically.
type Store interface {
	Load() ([]byte, error)
	Save([]byte) error
}

var ErrStoreLoad = errors.New("STORE_LOAD_FAILED")
var ErrStoreSave = errors.New("STORE_SAVE_FAILED")

const maxProfiles = 128

type storedProfile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Content  string `json:"content"`
}
type storedProfiles struct {
	Profiles   []storedProfile `json:"profiles"`
	SelectedID string          `json:"selectedID"`
}

func NewPersistentWithConnector(c Connector) *Service { return NewWithStore(c, profilevault.Default()) }
func NewWithStore(c Connector, store Store) *Service {
	s := NewWithConnector(c)
	s.store = store
	if store == nil {
		return s
	}
	raw, err := store.Load()
	if os.IsNotExist(err) {
		return s
	}
	if err == nil {
		err = s.restore(raw)
	}
	if err != nil {
		s.loadFailed = true
		s.storageStatus = "error"
		s.storageErrorCode = "STORE_LOAD_FAILED"
	}
	return s
}
func parseStored(format, content string) (Profile, error) {
	if len(content) == 0 || len(content) > MaxBytes || !utf8.ValidString(content) {
		return Profile{}, ErrInvalid
	}
	switch format {
	case "vless":
		return parseVLESS(content)
	case "openvpn":
		return parseOpenVPN(content)
	}
	return Profile{}, ErrInvalid
}
func (s *Service) storageLoadFailed() bool { s.mu.RLock(); defer s.mu.RUnlock(); return s.loadFailed }

// uniqueJSON rejects duplicate object members at every nesting level.
func uniqueJSON(d *json.Decoder) bool {
	t, err := d.Token()
	if err != nil {
		return false
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return true
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return false
			}
			key, ok := k.(string)
			if !ok || seen[key] {
				return false
			}
			seen[key] = true
			if !uniqueJSON(d) {
				return false
			}
		}
		end, err := d.Token()
		return err == nil && end == json.Delim('}')
	case '[':
		for d.More() {
			if !uniqueJSON(d) {
				return false
			}
		}
		end, err := d.Token()
		return err == nil && end == json.Delim(']')
	}
	return false
}
func (s *Service) restore(raw []byte) error {
	if len(raw) == 0 || len(raw) > profilevault.MaxBytes || !utf8.Valid(raw) || !uniqueJSON(json.NewDecoder(bytes.NewReader(raw))) {
		return ErrStoreLoad
	}
	var doc storedProfiles
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&doc) != nil || d.Decode(new(any)) != io.EOF || doc.Profiles == nil || len(doc.Profiles) > maxProfiles {
		return ErrStoreLoad
	}
	entries := make([]entry, 0, len(doc.Profiles))
	seen := map[string]bool{}
	for _, r := range doc.Profiles {
		if len(r.ID) != 64 || strings.ToLower(r.ID) != r.ID || seen[r.ID] || !validLabel(r.Name) {
			return ErrStoreLoad
		}
		if _, err := hex.DecodeString(r.ID); err != nil {
			return ErrStoreLoad
		}
		seen[r.ID] = true
		p, err := parseStored(r.Protocol, r.Content)
		if err != nil {
			return ErrStoreLoad
		}
		p.ID = r.ID
		p.Name = r.Name
		entries = append(entries, entry{p, r.Content})
	}
	if (len(entries) > 0 && !seen[doc.SelectedID]) || (len(entries) == 0 && doc.SelectedID != "") {
		return ErrStoreLoad
	}
	s.entries = entries
	s.selected = doc.SelectedID
	if len(entries) > 0 {
		s.storageStatus = "ready"
	}
	return nil
}

// commitLocked persists before publishing any mutation to readers or connectors.
func (s *Service) commitLocked(entries []entry, selected string) error {
	if s.loadFailed {
		return ErrStoreLoad
	}
	if len(entries) > maxProfiles {
		return ErrInvalid
	}
	if s.store != nil {
		doc := storedProfiles{Profiles: make([]storedProfile, 0, len(entries)), SelectedID: selected}
		for _, e := range entries {
			doc.Profiles = append(doc.Profiles, storedProfile{e.profile.ID, e.profile.Name, e.profile.Protocol, e.secret})
		}
		raw, err := json.Marshal(doc)
		if err != nil || len(raw) > profilevault.MaxBytes {
			return ErrInvalid
		}
		if s.store.Save(raw) != nil {
			s.storageStatus = "error"
			s.storageErrorCode = "STORE_SAVE_FAILED"
			return ErrStoreSave
		}
	}
	s.entries = entries
	s.selected = selected
	s.storageStatus = "empty"
	if len(entries) > 0 {
		s.storageStatus = "ready"
	}
	s.storageErrorCode = ""
	return nil
}
func (s *Service) storageModeLocked() string {
	if s.store != nil {
		return "protected"
	}
	return "memory"
}
func validLabel(s string) bool {
	if strings.TrimSpace(s) == "" || !utf8.ValidString(s) || utf8.RuneCountInString(s) > 80 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validHost(h string) bool {
	if net.ParseIP(h) != nil {
		return true
	}
	if len(h) == 0 || len(h) > 253 {
		return false
	}
	for _, part := range strings.Split(h, ".") {
		if len(part) == 0 || len(part) > 63 || part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
func portNumber(s string) (int, error) {
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, ErrInvalid
		}
	}
	p, e := strconv.Atoi(s)
	if e != nil || p < 1 || p > 65535 {
		return 0, ErrInvalid
	}
	return p, nil
}
func parseVLESS(content string) (Profile, error) {
	p := Profile{Protocol: "vless", Warnings: []string{"Syntax validated only; network authentication has not been tested."}}
	u, e := url.Parse(content)
	if e != nil || u.Scheme != "vless" || u.User == nil || !uuidPattern.MatchString(u.User.Username()) || u.Opaque != "" {
		return p, ErrInvalid
	}
	if _, set := u.User.Password(); set {
		return p, ErrInvalid
	}
	if _, _, err := net.SplitHostPort(u.Host); err != nil {
		return p, ErrInvalid
	}
	if u.Path != "" || u.RawPath != "" || !validHost(u.Hostname()) {
		return p, ErrInvalid
	}
	p.Host = u.Hostname()
	p.Port, e = portNumber(u.Port())
	if e != nil {
		return p, ErrInvalid
	}
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return p, ErrInvalid
	}
	allowed := map[string]bool{"security": true, "type": true, "encryption": true, "flow": true, "sni": true, "fp": true, "pbk": true, "sid": true, "spx": true, "path": true, "host": true, "serviceName": true, "alpn": true, "headerType": true, "authority": true, "mode": true}
	for k, v := range q {
		if !allowed[k] || len(v) != 1 {
			return p, ErrInvalid
		}
	}
	p.Security = q.Get("security")
	if p.Security == "" {
		p.Security = "none"
	}
	if p.Security != "none" && p.Security != "tls" && p.Security != "reality" {
		return p, ErrInvalid
	}
	p.Transport = q.Get("type")
	if p.Transport == "" {
		p.Transport = "tcp"
	}
	switch p.Transport {
	case "tcp", "ws", "grpc", "http", "httpupgrade", "xhttp":
	default:
		return p, ErrInvalid
	}
	if enc := q.Get("encryption"); enc != "" && enc != "none" {
		return p, ErrInvalid
	}
	p.Name = u.Fragment
	if p.Name == "" {
		p.Name = "VLESS profile"
	}
	if !validLabel(p.Name) {
		return p, ErrInvalid
	}
	return p, nil
}
func (s *Service) Import(format, content, name string) (Profile, error) {
	if s.storageLoadFailed() {
		return Profile{}, ErrStoreLoad
	}
	if len(content) == 0 || len(content) > MaxBytes || !utf8.ValidString(content) {
		return Profile{}, ErrInvalid
	}
	var p Profile
	var err error
	switch format {
	case "vless":
		p, err = parseVLESS(content)
	case "openvpn":
		p, err = parseOpenVPN(content)
	default:
		err = ErrInvalid
	}
	if err != nil {
		return Profile{}, ErrInvalid
	}
	if name != "" {
		if !validLabel(name) {
			return Profile{}, ErrInvalid
		}
		p.Name = strings.TrimSpace(name)
	}
	p.ID = randomID()
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := append(append([]entry(nil), s.entries...), entry{p, content})
	selected := s.selected
	if selected == "" {
		selected = p.ID
	}
	if err := s.commitLocked(entries, selected); err != nil {
		return Profile{}, err
	}
	return cloneProfile(p), nil
}
func cloneProfile(p Profile) Profile { p.Warnings = append([]string{}, p.Warnings...); return p }
