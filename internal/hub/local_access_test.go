package hub

import (
	"net/http/httptest"
	"testing"
)

func TestAllowHubRequest_loopback(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/channels", nil)
	r.RemoteAddr = "127.0.0.1:54321"
	if !AllowHubRequest(r) {
		t.Fatal("expected loopback allowed")
	}
}

func TestAllowHubRequest_rejectsRemoteWithoutToken(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	r := httptest.NewRequest("GET", "/api/channels", nil)
	r.RemoteAddr = "192.168.1.50:1234"
	if AllowHubRequest(r) {
		t.Fatal("expected remote denied without token")
	}
}

func TestAllowHubRequest_token(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "test-secret-token")
	r := httptest.NewRequest("POST", "/api/send", nil)
	r.RemoteAddr = "10.0.0.5:9999"
	r.Header.Set("X-NJ-Hub-Token", "test-secret-token")
	if !AllowHubRequest(r) {
		t.Fatal("expected valid token")
	}
	r.Header.Set("X-NJ-Hub-Token", "wrong")
	if AllowHubRequest(r) {
		t.Fatal("expected invalid token denied")
	}
}

func TestAllowHubRequest_queryTokenWebSocketOnly(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "ws-secret")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")

	// Plain HTTP API: query token rejected in non-relaxed mode.
	r := httptest.NewRequest("GET", "/api/channels?hub_token=ws-secret", nil)
	r.RemoteAddr = "10.0.0.5:9999"
	if AllowHubRequest(r) {
		t.Fatal("expected HTTP query hub_token rejected without relaxed local")
	}

	// WebSocket upgrade: query token still accepted (browsers cannot set headers).
	ws := httptest.NewRequest("GET", "/ws?hub_token=ws-secret", nil)
	ws.RemoteAddr = "10.0.0.5:9999"
	ws.Header.Set("Upgrade", "websocket")
	if !AllowHubRequest(ws) {
		t.Fatal("expected WebSocket query hub_token allowed")
	}
}

func TestAllowHubRequest_queryTokenRelaxed(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "ws-secret")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "1")
	r := httptest.NewRequest("GET", "/api/channels?hub_token=ws-secret", nil)
	r.RemoteAddr = "10.0.0.5:9999"
	if !AllowHubRequest(r) {
		t.Fatal("expected query hub_token allowed in relaxed local")
	}
}

func TestIsLoopbackRemoteAddr(t *testing.T) {
	if !IsLoopbackRemoteAddr("127.0.0.1:18765") {
		t.Fatal("127.0.0.1")
	}
	if !IsLoopbackRemoteAddr("[::1]:18765") {
		t.Fatal("::1")
	}
	if IsLoopbackRemoteAddr("203.0.113.1:18765") {
		t.Fatal("public IP should not be loopback")
	}
}
