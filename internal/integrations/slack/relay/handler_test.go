package relay

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	slackint "github.com/camronwood/neural-junkie/internal/integrations/slack"
)

func TestHandler_forwardToLoopback(t *testing.T) {
	local := "http://127.0.0.1:18765/api/slack/oauth/callback"
	state := slackint.FormatOAuthState("nonce1", local)
	req := httptest.NewRequest(http.MethodGet, slackint.OAuthCallbackPath+"?code=abc&state="+url.QueryEscape(state), nil)
	rec := httptest.NewRecorder()
	NewHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc == "" || loc[:4] != "http" {
		t.Fatalf("location = %q", loc)
	}
}

func TestHandler_rejectsMissingState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, slackint.OAuthCallbackPath+"?code=abc", nil)
	rec := httptest.NewRecorder()
	NewHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
