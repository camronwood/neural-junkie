package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func TestHandleUserRulesGetPut(t *testing.T) {
	chatHub = hub.NewHub()
	hubSessions = hub.NewSessionManager()
	sess := hubSessions.CreateSession("Camron", "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/user-rules", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	rec := httptest.NewRecorder()
	handleUserRules(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status %d: %s", rec.Code, rec.Body.String())
	}

	raw, _ := json.Marshal(map[string]string{"markdown": "Use concise bullets."})
	req = httptest.NewRequest(http.MethodPut, "/api/user-rules", bytes.NewReader(raw))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	rec = httptest.NewRecorder()
	handleUserRules(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/user-rules", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	rec = httptest.NewRecorder()
	handleUserRules(rec, req)
	var body struct {
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Markdown != "Use concise bullets." {
		t.Fatalf("got markdown %q", body.Markdown)
	}
}
