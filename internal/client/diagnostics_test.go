package client

import (
	"context"
	"encoding/json"
	"testing"
)

type diagnosticConnector struct {
	fakeConnector
	code string
}

func (c *diagnosticConnector) Connect(context.Context, string) error {
	return &ConnectionError{Code: c.code}
}
func TestConnectionDiagnosticCodesSurviveHTTP(t *testing.T) {
	for _, code := range []string{"ENGINE_START_FAILED", "EGRESS_CHECK_FAILED", "HELPER_START_FAILED", "HELPER_EXITED", "HELPER_TIMEOUT"} {
		t.Run(code, func(t *testing.T) {
			s := NewWithConnector(&diagnosticConnector{code: code})
			p, err := s.Import("vless", testURI, "")
			if err != nil {
				t.Fatal(err)
			}
			w := call(NewHandler(s), "POST", "/api/connect", `{"id":"`+p.ID+`"}`, s.token)
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != 502 || body.Error.Code != code {
				t.Fatal(w.Code, body.Error.Code)
			}
		})
	}
}
