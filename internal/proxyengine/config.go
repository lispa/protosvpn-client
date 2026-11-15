// Package proxyengine runs an official Xray core without persisting proxy credentials.
package proxyengine

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var ErrInvalid = errors.New("proxyengine: invalid or unsupported VLESS configuration")
var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validHost(s string) bool {
	if net.ParseIP(s) != nil {
		return true
	}
	if len(s) == 0 || len(s) > 253 {
		return false
	}
	for _, part := range strings.Split(strings.TrimSuffix(s, "."), ".") {
		if len(part) == 0 || len(part) > 63 || part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}
func makeConfig(uri string, port int) ([]byte, error) {
	return buildConfig(uri, port, 0)
}

func makeConfigWithStats(uri string, port, statsPort int) ([]byte, error) {
	if statsPort < 1 || statsPort > 65535 || statsPort == port {
		return nil, ErrInvalid
	}
	return buildConfig(uri, port, statsPort)
}

func buildConfig(uri string, port, statsPort int) ([]byte, error) {
	bad := func() ([]byte, error) { return nil, ErrInvalid }
	if len(uri) > 65536 || strings.IndexFunc(uri, unicode.IsControl) >= 0 {
		return bad()
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "vless" || u.User == nil || !uuid.MatchString(u.User.Username()) || u.Opaque != "" || u.Path != "" || u.RawPath != "" {
		return bad()
	}
	if _, yes := u.User.Password(); yes {
		return bad()
	}
	if _, _, err = net.SplitHostPort(u.Host); err != nil || !validHost(u.Hostname()) {
		return bad()
	}
	remote, err := strconv.Atoi(u.Port())
	if err != nil || remote < 1 || remote > 65535 || port < 1 || port > 65535 {
		return bad()
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return bad()
	}
	allowed := map[string]bool{}
	for _, k := range strings.Fields("security type encryption flow sni fp pbk sid spx path host serviceName alpn headerType authority mode") {
		allowed[k] = true
	}
	for k, v := range q {
		if !allowed[k] || len(v) != 1 || v[0] == "" || strings.IndexFunc(v[0], unicode.IsControl) >= 0 {
			return bad()
		}
	}
	get := q.Get
	if v := get("encryption"); v != "" && v != "none" {
		return bad()
	}
	security := get("security")
	if security == "" {
		security = "none"
	}
	if security != "none" && security != "tls" && security != "reality" {
		return bad()
	}
	network := get("type")
	if network == "" || network == "tcp" {
		network = "raw"
	}
	switch network {
	case "raw", "ws", "grpc", "httpupgrade", "xhttp":
	default:
		return bad()
	}
	permitted := map[string]bool{"security": true, "type": true, "encryption": true}
	permit := func(keys string) {
		for _, k := range strings.Fields(keys) {
			permitted[k] = true
		}
	}
	stream := map[string]any{"network": network, "security": security}
	if security != "none" {
		permit("sni fp alpn")
		sni := get("sni")
		if sni == "" {
			sni = u.Hostname()
		}
		if !validHost(sni) {
			return bad()
		}
		fp := get("fp")
		if fp != "" {
			switch fp {
			case "chrome", "firefox", "safari", "ios", "android", "edge", "360", "qq", "random", "randomized":
			default:
				return bad()
			}
		}
		if security == "tls" {
			tls := map[string]any{"serverName": sni, "allowInsecure": false}
			if fp != "" {
				tls["fingerprint"] = fp
			}
			if get("alpn") != "" {
				a := strings.Split(get("alpn"), ",")
				for _, v := range a {
					if v != "h2" && v != "http/1.1" {
						return bad()
					}
				}
				tls["alpn"] = a
			}
			stream["tlsSettings"] = tls
		} else {
			permit("pbk sid spx")
			delete(permitted, "alpn")
			if get("sni") == "" || fp == "" || network != "raw" && network != "grpc" && network != "xhttp" {
				return bad()
			}
			key, e := base64.RawURLEncoding.DecodeString(get("pbk"))
			if e != nil || len(key) != 32 {
				return bad()
			}
			nonzero := false
			for _, v := range key {
				nonzero = nonzero || v != 0
			}
			if !nonzero {
				return bad()
			}
			sid := get("sid")
			if len(sid) > 16 || len(sid)%2 != 0 {
				return bad()
			}
			if _, e := hex.DecodeString(sid); e != nil {
				return bad()
			}
			r := map[string]any{"serverName": sni, "fingerprint": fp, "publicKey": get("pbk"), "shortId": sid}
			if get("spx") != "" {
				if !strings.HasPrefix(get("spx"), "/") {
					return bad()
				}
				r["spiderX"] = get("spx")
			}
			stream["realitySettings"] = r
		}
	}
	flow := get("flow")
	if flow != "" {
		permit("flow")
		if flow != "xtls-rprx-vision" || network != "raw" || security == "none" {
			return bad()
		}
	}
	switch network {
	case "raw":
		permit("headerType")
		if get("headerType") != "" && get("headerType") != "none" {
			return bad()
		}
	case "ws", "httpupgrade", "xhttp":
		permit("path host")
		p := get("path")
		if p == "" {
			p = "/"
		}
		if !strings.HasPrefix(p, "/") {
			return bad()
		}
		s := map[string]any{"path": p}
		if h := get("host"); h != "" {
			if !validHost(h) {
				return bad()
			}
			if network == "ws" {
				s["headers"] = map[string]string{"Host": h}
			} else {
				s["host"] = h
			}
		}
		if network == "xhttp" {
			permit("mode")
			m := get("mode")
			if m == "" {
				m = "auto"
			}
			switch m {
			case "auto", "packet-up", "stream-up", "stream-one":
			default:
				return bad()
			}
			s["mode"] = m
		}
		stream[network+"Settings"] = s
	case "grpc":
		permit("serviceName authority mode")
		s := map[string]any{"serviceName": get("serviceName")}
		if get("authority") != "" {
			if !validHost(get("authority")) {
				return bad()
			}
			s["authority"] = get("authority")
		}
		if get("mode") != "" && get("mode") != "gun" && get("mode") != "multi" {
			return bad()
		}
		s["multiMode"] = get("mode") == "multi"
		stream["grpcSettings"] = s
	}
	for k := range q {
		if !permitted[k] {
			return bad()
		}
	}
	user := map[string]any{"id": u.User.Username(), "encryption": "none"}
	if flow != "" {
		user["flow"] = flow
	}
	config := map[string]any{"log": map[string]any{"loglevel": "none", "access": "none", "error": "none"}, "inbounds": []any{map[string]any{"listen": "127.0.0.1", "port": port, "protocol": "http", "settings": map[string]any{}}}, "outbounds": []any{map[string]any{"protocol": "vless", "settings": map[string]any{"vnext": []any{map[string]any{"address": u.Hostname(), "port": remote, "users": []any{user}}}}, "streamSettings": stream}}}
	if statsPort != 0 {
		config["outbounds"].([]any)[0].(map[string]any)["tag"] = "proxy"
		config["api"] = map[string]any{"tag": "stats-api", "listen": net.JoinHostPort("127.0.0.1", strconv.Itoa(statsPort)), "services": []string{"StatsService"}}
		config["stats"] = map[string]any{}
		config["policy"] = map[string]any{"system": map[string]any{"statsOutboundUplink": true, "statsOutboundDownlink": true}}
	}
	b, err := json.Marshal(config)
	if err != nil {
		return bad()
	}
	return b, nil
}
