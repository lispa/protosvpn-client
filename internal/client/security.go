package client

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func (s *Service) validToken(values []string) bool {
	if len(values) != 1 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(values[0]), []byte(s.token)) == 1
}

// localRequest deliberately ignores forwarded headers. Only the actual Host
// and a corresponding exact Origin may authorize this local-only API.
func localRequest(r *http.Request) bool {
	host := r.Host
	name := host
	if strings.Contains(host, ":") {
		var port string
		var err error
		name, port, err = net.SplitHostPort(host)
		if err != nil {
			if host != "[::1]" {
				return false
			}
			name = "::1"
		} else if _, err = portNumber(port); err != nil {
			return false
		}
	}
	switch name {
	case "127.0.0.1", "localhost", "::1", "wails.localhost", "wails":
	default:
		return false
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origins := r.Header.Values("Origin")
	if len(origins) == 0 {
		return true
	}
	if len(origins) != 1 || origins[0] == "" {
		return false
	}
	origin := origins[0]
	u, err := url.Parse(origin)
	if err != nil || u.Host != host || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || origin != u.Scheme+"://"+host {
		return false
	}
	if u.Scheme == "http" {
		return true
	}
	return (u.Scheme == "wails" && name == "wails") || (u.Scheme == "https" && (name == "wails.localhost" || name == "wails"))
}
