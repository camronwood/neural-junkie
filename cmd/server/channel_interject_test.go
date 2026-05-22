package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func TestHandleChannelInterjectRoute(t *testing.T) {
	chatHub = hub.NewHub()
	chatHub.CreateChannel("interject-ch", "", "")
	body := strings.NewReader(`{"held_by":"tester"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channels/interject-ch/interject", body)
	rec := httptest.NewRecorder()
	handleChannelInterjectRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !chatHub.IsChannelHeld("interject-ch") {
		t.Fatal("expected channel held after interject")
	}
}
