package client

import (
	"encoding/json"
	"strings"
	"testing"
)

const testURI = "vless://00000000-0000-4000-8000-000000000001@192.0.2.1:443?security=tls&type=ws&path=%2Fdemo#Demo%20VPN"

func TestImportVLESS(t *testing.T) {
	s := New()
	p, err := s.Import("vless", testURI, "")
	if err != nil || p.Host != "192.0.2.1" || p.Port != 443 || p.Name != "Demo VPN" || p.Protocol != "vless" || p.ID == "" || p.Transport != "ws" {
		t.Fatalf("valid import failed: %+v %v", p, err)
	}
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), "00000000-") || strings.Contains(string(b), "path") {
		t.Fatal("secret leaked")
	}
}
