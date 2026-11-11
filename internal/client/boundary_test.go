package client

import (
 "encoding/json"
 "net/http/httptest"
 "strings"
 "testing"
)
func TestImportBoundaries(t *testing.T) {
 for _,tc:=range []struct{format,content,name string}{
 {"vless","",""},{"vless",strings.Repeat("a",MaxBytes+1),""},{"vless",testURI,strings.Repeat("я",81)},{"vless",testURI,"name\tvalue"},{"vless",testURI,"   "},{"vless",testURI+"%0A",""},{"subscription",testURI,""},
 {"vless",strings.Replace(testURI,"192.0.2.1","-invalid.test",1),""},{"vless",strings.Replace(testURI,":443",":65536",1),""},{"vless",strings.Replace(testURI,":443","",1),""},
 {"openvpn","client\ndev tap\nremote vpn.example.test 443\n",""},{"openvpn","dev tun\nremote vpn.example.test 443\n",""},{"openvpn",testOVPN+"<cert>\nunterminated\n",""},{"openvpn",testOVPN+"<cert>\n</cert>\n",""},{"openvpn",testOVPN+"proto tcp-server\n",""},
 } {s:=New();if _,err:=s.Import(tc.format,tc.content,tc.name);err==nil {t.Errorf("invalid %s input accepted",tc.format)};if len(s.State().Profiles)!=0 {t.Fatal("invalid import mutated state")}}
 s:=New();p,e:=s.Import("vless",testURI,strings.Repeat("я",80));if e!=nil||len([]rune(p.Name))!=80 {t.Fatal("Unicode limit counts bytes")}
 // Verify exact private input retention without exposing it through the exported API.
 if s.entries[0].secret!=testURI {t.Fatal("private input changed")}
}
func TestActualBodyByteLimit(t *testing.T) {
 s:=New();h:=NewHandler(s)
 body,_:=json.Marshal(map[string]string{"format":"openvpn","content":testOVPN+"#"+strings.Repeat("я",MaxBytes/2)})
 r:=httptest.NewRequest("POST","http://localhost/api/import",strings.NewReader(string(body)));r.ContentLength=1;r.Header.Set("Content-Type","application/json");r.Header.Set("X-ProtosVPN-Token",s.token)
 w:=httptest.NewRecorder();h.ServeHTTP(w,r);if w.Code!=400||len(s.State().Profiles)!=0 {t.Fatal("trusted declared length rather than actual bytes")}
}
func TestSecurityHeaderMultiplicity(t *testing.T) {
 s:=New();h:=NewHandler(s)
 for _,header:=range []string{"Origin","X-ProtosVPN-Token"} {
 r:=httptest.NewRequest("POST","http://localhost/api/disconnect",strings.NewReader("{}"));r.Header.Set("Content-Type","application/json");r.Header.Set("Origin","http://localhost");r.Header.Set("X-ProtosVPN-Token",s.token);r.Header.Add(header,r.Header.Get(header))
 w:=httptest.NewRecorder();h.ServeHTTP(w,r);if w.Code!=403 {t.Fatal("duplicate security header accepted")}
 }
}
func FuzzImport(f *testing.F) {
 f.Add("vless",testURI);f.Add("openvpn",testOVPN);f.Add("openvpn","<key>\nsecret\n</cert>")
 f.Fuzz(func(t *testing.T,format,content string){if len(content)>MaxBytes+1 {t.Skip()};s:=New();p,e:=s.Import(format,content,"");if e!=nil {if e!=ErrInvalid {t.Fatal("unexpected error")};if len(s.State().Profiles)!=0 {t.Fatal("failed import changed state")};return};if p.ID==""||!validLabel(p.Name)||!validHost(p.Host)||p.Port<1||p.Port>65535 {t.Fatal("invalid imported metadata")};if len(s.State().Profiles)!=1 {t.Fatal("missing imported profile")}})
}
