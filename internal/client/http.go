package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func sendJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func sendError(w http.ResponseWriter, status int, code, message string) {
	sendJSON(w, status, struct {
		Error apiError `json:"error"`
	}{apiError{code, message}})
}
func serviceError(w http.ResponseWriter, e error) {
	if errors.Is(e, ErrNotFound) {
		sendError(w, 404, "NOT_FOUND", "Profile not found.")
	} else {
		sendError(w, 400, "INVALID_INPUT", "Invalid or unsupported profile data.")
	}
}
func decodeBody(w http.ResponseWriter, r *http.Request, out any, emptyOK bool) bool {
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBytes))
	if e != nil || !utf8.Valid(raw) {
		sendError(w, 400, "INVALID_INPUT", "Invalid request body or size limit exceeded.")
		return false
	}
	if emptyOK && len(raw) == 0 {
		return true
	}
	media, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil || media != "application/json" {
		sendError(w, 400, "INVALID_INPUT", "A JSON request body is required.")
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if !strictStringObject(raw) || d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		sendError(w, 400, "INVALID_INPUT", "Invalid JSON request body.")
		return false
	}
	return true
}
func strictStringObject(raw []byte) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	first, err := d.Token()
	if err != nil || first != json.Delim('{') {
		return false
	}
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return false
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return false
		}
		seen[key] = true
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return false
		}
		if len(value) == 0 || value[0] != '"' {
			return false
		}
	}
	last, err := d.Token()
	return err == nil && last == json.Delim('}') && d.Decode(new(any)) == io.EOF
}
func NewHandler(s *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if !localRequest(r) {
			sendError(w, 403, "FORBIDDEN", "Local request denied.")
			return
		}
		path := r.URL.Path
		method := ""
		id := ""
		switch path {
		case "/api/state", "/api/diagnostics":
			method = "GET"
		case "/api/import", "/api/select", "/api/connect", "/api/disconnect":
			method = "POST"
		default:
			if strings.HasPrefix(path, "/api/profiles/") {
				id = strings.TrimPrefix(path, "/api/profiles/")
				if id != "" && !strings.Contains(id, "/") {
					method = "PATCH, DELETE"
				}
			}
		}
		if method == "" {
			sendError(w, 404, "NOT_FOUND", "API route not found.")
			return
		}
		if r.Method != method && !(method == "PATCH, DELETE" && (r.Method == "PATCH" || r.Method == "DELETE")) {
			w.Header().Set("Allow", method)
			sendError(w, 405, "METHOD_NOT_ALLOWED", "Method not allowed.")
			return
		}
		if r.Method != "GET" && !s.validToken(r.Header.Values("X-ProtosVPN-Token")) {
			sendError(w, 403, "FORBIDDEN", "Local request denied.")
			return
		}
		switch path {
		case "/api/state":
			sendJSON(w, 200, s.State())
		case "/api/diagnostics":
			sendJSON(w, 200, s.Diagnostics())
		case "/api/import":
			var input struct {
				Format  string `json:"format"`
				Content string `json:"content"`
				Name    string `json:"name"`
			}
			if !decodeBody(w, r, &input, false) {
				return
			}
			p, e := s.Import(input.Format, input.Content, input.Name)
			if e != nil {
				serviceError(w, e)
				return
			}
			sendJSON(w, 201, map[string]any{"profile": p})
		case "/api/select", "/api/connect":
			var input struct {
				ID string `json:"id"`
			}
			if !decodeBody(w, r, &input, false) {
				return
			}
			if input.ID == "" {
				serviceError(w, ErrInvalid)
				return
			}
			if path == "/api/connect" {
				if !s.hasProfile(input.ID) {
					serviceError(w, ErrNotFound)
					return
				}
				engineUnavailable(w)
				return
			}
			if e := s.Select(input.ID); e != nil {
				serviceError(w, e)
				return
			}
			sendJSON(w, 200, map[string]bool{"ok": true})
		case "/api/disconnect":
			var input struct{}
			if !decodeBody(w, r, &input, true) {
				return
			}
			engineUnavailable(w)
		default:
			if r.Method == "PATCH" {
				var input struct {
					Name string `json:"name"`
				}
				if !decodeBody(w, r, &input, false) {
					return
				}
				p, e := s.Rename(id, input.Name)
				if e != nil {
					serviceError(w, e)
					return
				}
				sendJSON(w, 200, map[string]any{"profile": p})
			} else {
				var input struct{}
				if !decodeBody(w, r, &input, true) {
					return
				}
				if e := s.Delete(id); e != nil {
					serviceError(w, e)
					return
				}
				sendJSON(w, 200, map[string]bool{"ok": true})
			}
		}
	})
}
func engineUnavailable(w http.ResponseWriter) {
	sendError(w, 409, "ENGINE_UNAVAILABLE", "VPN engine and system helper are not installed in this build; no connection was attempted.")
}
func (s *Service) hasProfile(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.profile.ID == id {
			return true
		}
	}
	return false
}
