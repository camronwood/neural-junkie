package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/testutil"
)

func resetSlackTestGlobals(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		appConfig = nil
		slackBridge = nil
	})
}

func TestHandleSlackStatus_disabled(t *testing.T) {
	resetSlackTestGlobals(t)
	appConfig = config.DefaultConfig()

	req := httptest.NewRequest(http.MethodGet, "/api/slack/status", nil)
	rec := httptest.NewRecorder()
	handleSlackStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code: got %d want 200", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"enabled", "configured", "oauth_configured", "connect_ready", "token_set"} {
		if _, ok := resp[key]; !ok {
			t.Fatalf("missing key %q in status JSON", key)
		}
	}
	if resp["enabled"] != false {
		t.Fatalf("enabled: got %v want false", resp["enabled"])
	}
	if resp["configured"] != false {
		t.Fatalf("configured: got %v want false", resp["configured"])
	}
}

func TestHandleSlackStatus_configuredNoBridge(t *testing.T) {
	resetSlackTestGlobals(t)
	appConfig = config.DefaultConfig()
	appConfig.Slack.Enabled = true
	appConfig.Slack.BotToken = "xoxb-test"
	appConfig.Slack.AppToken = "xapp-test"

	req := httptest.NewRequest(http.MethodGet, "/api/slack/status", nil)
	rec := httptest.NewRecorder()
	handleSlackStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code: got %d want 200", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["enabled"] != true {
		t.Fatalf("enabled: got %v want true", resp["enabled"])
	}
	if resp["configured"] != true {
		t.Fatalf("configured: got %v want true", resp["configured"])
	}
	if resp["token_set"] != true {
		t.Fatalf("token_set: got %v want true", resp["token_set"])
	}
}

func TestHandleSlackConfig_getPut(t *testing.T) {
	testutil.IsolateNeuralJunkieHome(t)
	resetSlackTestGlobals(t)
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "1")
	hubSessions = hub.NewSessionManager()
	appConfig = config.DefaultConfig()

	req := httptest.NewRequest(http.MethodGet, "/api/slack/config", nil)
	rec := httptest.NewRecorder()
	handleSlackConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET config: got %d %s", rec.Code, rec.Body.String())
	}
	var got map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["enabled"] != false {
		t.Fatalf("GET enabled: got %v", got["enabled"])
	}

	enabled := true
	body, _ := json.Marshal(map[string]interface{}{
		"enabled":        &enabled,
		"default_policy": "mention_only",
	})
	sess := hubSessions.CreateSession("test", "admin")
	req = loopbackRequest(http.MethodPut, "/api/slack/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NJ-Session", sess.Token)
	rec = httptest.NewRecorder()
	handleSlackConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT config: got %d %s", rec.Code, rec.Body.String())
	}
	if !appConfig.Slack.Enabled {
		t.Fatal("expected Slack.Enabled true after PUT")
	}
	if appConfig.Slack.DefaultPolicy != config.SlackPolicyMentionOnly {
		t.Fatalf("default_policy: got %q want mention_only", appConfig.Slack.DefaultPolicy)
	}
}

func TestHandleSlackSmokeRun_noConfig(t *testing.T) {
	resetSlackTestGlobals(t)
	req := httptest.NewRequest(http.MethodPost, "/api/slack/smoke/run", nil)
	rec := httptest.NewRecorder()
	handleSlackSmokeRun(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d want 503", rec.Code)
	}
}

func TestHandleSlackSmokeRun_syntheticOnly(t *testing.T) {
	testutil.IsolateNeuralJunkieHome(t)
	resetSlackTestGlobals(t)
	appConfig = config.DefaultConfig()
	body := []byte(`{"outbound":false}`)
	req := loopbackRequest(http.MethodPost, "/api/slack/smoke/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleSlackSmokeRun(rec, req)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusOK {
		t.Fatalf("status: got %d %s", rec.Code, rec.Body.String())
	}
	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if _, ok := result["checks"]; !ok {
		t.Fatalf("expected checks in response: %+v", result)
	}
}
