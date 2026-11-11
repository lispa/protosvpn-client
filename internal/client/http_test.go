package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func call(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:18761"+path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("X-ProtosVPN-Token", token)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestHTTPRealLifecycle(t *testing.T) {
	s := New()
	h := NewHandler(s)
	w := call(h, "GET", "/api/state", "", "")
	if w.Code != 200 {
		t.Fatalf("bootstrap: %d", w.Code)
	}
	var state State
	if json.Unmarshal(w.Body.Bytes(), &state) != nil || state.CSRFToken == "" {
		t.Fatal("missing token")
	}
	token := state.CSRFToken
	body, _ := json.Marshal(map[string]string{"format": "vless", "content": testURI})
	w = call(h, "POST", "/api/import", string(body), token)
	if w.Code != 201 {
		t.Fatalf("import %d: %s", w.Code, w.Body.String())
	}
	var imported struct {
		Profile Profile `json:"profile"`
	}
	json.Unmarshal(w.Body.Bytes(), &imported)
	id := imported.Profile.ID
	for _, step := range []struct {
		method, path, body string
		code               int
	}{{"POST", "/api/select", `{"id":"` + id + `"}`, 200}, {"PATCH", "/api/profiles/" + id, `{"name":"Renamed"}`, 200}, {"POST", "/api/connect", `{"id":"` + id + `"}`, 409}, {"POST", "/api/disconnect", `{}`, 409}, {"GET", "/api/diagnostics", "", 200}, {"DELETE", "/api/profiles/" + id, "", 200}, {"DELETE", "/api/profiles/" + id, "", 404}, {"POST", "/api/select", `{"id":"missing"}`, 404}, {"POST", "/api/state", `{}`, 405}, {"GET", "/api/unknown", "", 404}} {
		w = call(h, step.method, step.path, step.body, token)
		if w.Code != step.code {
			t.Errorf("%s %s: %d %s", step.method, step.path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "00000000-") {
			t.Fatal("secret leak")
		}
	}
	if s.State().Connection.Status != "disconnected" || len(s.State().Profiles) != 0 {
		t.Fatal("dishonest state")
	}
}
