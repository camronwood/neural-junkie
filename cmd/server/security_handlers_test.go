package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func TestHandleSystemSecurity(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "")
	t.Setenv("NEURAL_JUNKIE_LISTEN_ALL", "")

	req := httptest.NewRequest(http.MethodGet, "/api/system/security", nil)
	rec := httptest.NewRecorder()
	handleSystemSecurity(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
	var snap struct {
		HubTokenConfigured bool `json:"hub_token_configured"`
		AuthRequired        bool `json:"auth_required"`
		ListenAll           bool `json:"listen_all"`
		LoopbackOnly        bool `json:"loopback_only"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&snap); err != nil {
		t.Fatal(err)
	}
	if snap.HubTokenConfigured || snap.AuthRequired || snap.ListenAll {
		t.Fatalf("expected defaults off: %+v", snap)
	}
	if !snap.LoopbackOnly {
		t.Fatal("expected loopback_only true by default")
	}

	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "secret")
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	t.Setenv("NEURAL_JUNKIE_LISTEN_ALL", "1")
	if !hub.HubTokenConfigured() || !hub.AuthRequired() {
		t.Fatal("env setup failed")
	}
	rec2 := httptest.NewRecorder()
	handleSystemSecurity(rec2, req)
	if err := json.NewDecoder(rec2.Body).Decode(&snap); err != nil {
		t.Fatal(err)
	}
	if !snap.HubTokenConfigured || !snap.AuthRequired || !snap.ListenAll || snap.LoopbackOnly {
		t.Fatalf("expected strict env: %+v", snap)
	}
}
