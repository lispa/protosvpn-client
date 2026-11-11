package client

import "strings"

// parseOpenVPN deliberately accepts only a small, self-contained client grammar.
// Inline material is opaque, not certificate validation or executable engine input.
func parseOpenVPN(content string) (Profile, error) {
	p := Profile{Protocol: "openvpn", Name: "OpenVPN profile", Transport: "udp", Security: "tls", Warnings: []string{"Syntax validated only; certificates and network authentication have not been verified."}}
	block := ""
	blockContent := false
	seen := map[string]bool{}
	blocks := map[string]bool{"ca": true, "cert": true, "key": true, "tls-auth": true, "tls-crypt": true}
	client, tun, remote := false, false, false
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if block != "" {
			if line == "</"+block+">" {
				if !blockContent {
					return p, ErrInvalid
				}
				block = ""
				continue
			}
			if strings.HasPrefix(line, "<") {
				return p, ErrInvalid
			}
			if line != "" {
				blockContent = true
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "<") {
			tag := strings.TrimSuffix(strings.TrimPrefix(line, "<"), ">")
			if line != "<"+tag+">" || !blocks[tag] || seen["block:"+tag] {
				return p, ErrInvalid
			}
			seen["block:"+tag] = true
			block = tag
			blockContent = false
			continue
		}
		// No shell quoting, expansion, continuations or inline comments are interpreted.
		if strings.ContainsAny(line, "\x00\"'`\\") {
			return p, ErrInvalid
		}
		f := strings.Fields(line)
		key := f[0]
		args := f[1:]
		if seen[key] {
			return p, ErrInvalid
		}
		seen[key] = true
		switch key {
		case "client":
			if len(args) != 0 {
				return p, ErrInvalid
			}
			client = true
		case "dev":
			if len(args) != 1 || args[0] != "tun" {
				return p, ErrInvalid
			}
			tun = true
		case "remote":
			if len(args) != 2 || !validHost(args[0]) {
				return p, ErrInvalid
			}
			var e error
			p.Port, e = portNumber(args[1])
			if e != nil {
				return p, ErrInvalid
			}
			p.Host = args[0]
			remote = true
		case "proto":
			if len(args) != 1 {
				return p, ErrInvalid
			}
			switch args[0] {
			case "udp", "udp4", "udp6", "tcp-client", "tcp4-client", "tcp6-client":
				p.Transport = args[0]
			default:
				return p, ErrInvalid
			}
		case "auth-user-pass":
			if len(args) != 0 {
				return p, ErrInvalid
			}
			p.RequiresAuth = true
		case "nobind", "persist-key", "persist-tun", "remote-random", "pull":
			if len(args) != 0 {
				return p, ErrInvalid
			}
		case "resolv-retry":
			if len(args) != 1 || args[0] != "infinite" {
				return p, ErrInvalid
			}
		case "remote-cert-tls":
			if len(args) != 1 || args[0] != "server" {
				return p, ErrInvalid
			}
		case "key-direction":
			if len(args) != 1 || (args[0] != "0" && args[0] != "1") {
				return p, ErrInvalid
			}
		case "cipher", "auth", "data-ciphers", "data-ciphers-fallback", "tls-version-min", "verb":
			if len(args) != 1 || !safeOption(key, args[0]) {
				return p, ErrInvalid
			}
		default:
			return p, ErrInvalid
		}
	}
	if block != "" || !client || !tun || !remote {
		return p, ErrInvalid
	}
	return p, nil
}
func safeOption(key, value string) bool {
	allowed := map[string]string{
		"cipher":                "AES-128-GCM AES-256-GCM AES-128-CBC AES-256-CBC CHACHA20-POLY1305",
		"auth":                  "SHA256 SHA384 SHA512 SHA1",
		"data-ciphers-fallback": "AES-128-GCM AES-256-GCM AES-128-CBC AES-256-CBC CHACHA20-POLY1305",
		"tls-version-min":       "1.2 1.3", "verb": "0 1 2 3 4",
	}
	if key == "data-ciphers" {
		for _, v := range strings.Split(value, ":") {
			if !safeOption("cipher", v) {
				return false
			}
		}
		return value != ""
	}
	for _, v := range strings.Fields(allowed[key]) {
		if value == v {
			return true
		}
	}
	return false
}
