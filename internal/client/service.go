package client

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
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
	mu       sync.RWMutex
	entries  []entry
	selected string
	token    string
}

func randomID() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("secure randomness unavailable")
	}
	return hex.EncodeToString(b[:])
}
func New() *Service { return &Service{token: randomID(), entries: []entry{}} }
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
	s.entries = append(s.entries, entry{p, content})
	if s.selected == "" {
		s.selected = p.ID
	}
	return cloneProfile(p), nil
}
func cloneProfile(p Profile) Profile { p.Warnings = append([]string{}, p.Warnings...); return p }
