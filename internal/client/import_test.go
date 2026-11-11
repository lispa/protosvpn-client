package client

import (
	"strings"
	"testing"
)

const testOVPN = "client\ndev tun\nproto udp\nremote vpn.example.test 1194\nauth-user-pass\n<ca>\nSYNTHETIC CA ONLY\n</ca>\n<key>\nSYNTHETIC PRIVATE KEY ONLY\n</key>\n"

func TestImportOpenVPN(t *testing.T) {
	p, e := New().Import("openvpn", testOVPN, "")
	if e != nil || p.Protocol != "openvpn" || p.Port != 1194 || !p.RequiresAuth || p.Transport != "udp" {
		t.Fatalf("self-contained import failed: %+v %v", p, e)
	}
}
func TestUnsafeOpenVPNRejected(t *testing.T) {
	for _, line := range []string{"up /tmp/secret", "plugin danger.so", "config nested.conf", "ca /tmp/cert", "auth-user-pass credentials.txt", "management 127.0.0.1 1234", "log /tmp/output", "unknown-secret-directive value", "remote vpn.example.test 443", "<cert>\nsecret\n</key>", "setenv opt secret"} {
		t.Run(strings.Fields(line)[0], func(t *testing.T) {
			_, err := New().Import("openvpn", testOVPN+line+"\n", "")
			if err == nil {
				t.Fatal("unsafe directive accepted")
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("reflected sensitive input")
			}
		})
	}
}
func TestRejectAmbiguousIPv6Authority(t *testing.T) {
	uri := "vless://00000000-0000-4000-8000-000000000001@2001:db8::1:443?security=tls"
	if _, err := New().Import("vless", uri, ""); err == nil {
		t.Fatal("unbracketed IPv6 authority accepted")
	}
}
func TestMalformedVLESSRejected(t *testing.T) {
	for _, uri := range []string{strings.Replace(testURI, "vless:", "https:", 1), strings.Replace(testURI, "@", ":password@", 1), strings.Replace(testURI, "443", "0", 1), strings.Replace(testURI, "type=ws", "type=ws&type=tcp", 1), strings.Replace(testURI, "security=tls", "security=unsafe", 1), strings.Replace(testURI, "type=ws", "type=exec", 1), strings.Replace(testURI, "type=ws", "type=ws&unknown=secret", 1), strings.Replace(testURI, "00000000-0000-4000-8000-000000000001", "secret", 1), testURI + "%zz"} {
		if _, err := New().Import("vless", uri, ""); err == nil {
			t.Error("invalid URI accepted")
		}
	}
	p, e := New().Import("vless", "vless://00000000-0000-4000-8000-000000000001@[2001:db8::1]:443?security=reality&type=grpc&serviceName=demo%2Fservice#IPv6", "")
	if e != nil || p.Host != "2001:db8::1" {
		t.Fatal("IPv6 or escaped value rejected")
	}
}
