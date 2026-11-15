package proxyengine

import (
	"encoding/json"
	"strings"
	"testing"
)

const testID = "00000000-0000-4000-8000-000000000001"
const baseURI = "vless://" + testID + "@127.0.0.1:443"

func TestConfigStrict(t *testing.T) {
	b, err := makeConfig(baseURI+"?security=tls&type=ws&sni=example.org&path=%2Fsocket&host=example.org", 12345)
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	if json.Unmarshal(b, &c) != nil {
		t.Fatal("invalid JSON")
	}
	if !strings.Contains(string(b), `"allowInsecure":false`) || !strings.Contains(string(b), `"listen":"127.0.0.1"`) {
		t.Fatal("unsafe config")
	}
	for _, q := range []string{"security=unsafe", "security=tls&allowInsecure=1", "type=bogus", "type=ws&type=tcp", "security=reality", "security=none&sni=example.org", "type=tcp&path=/bad", "type=grpc&flow=xtls-rprx-vision", "type=ws&serviceName=x", "security=tls&fp=bogus", "encryption=foo", "headerType=http"} {
		if _, err := makeConfig(baseURI+"?"+q, 12345); err == nil {
			t.Errorf("accepted %s", q)
		}
	}
}
