package slack

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestParseSlackOAuthResponse(t *testing.T) {
	okJSON := `{"ok":true,"access_token":"xoxb-test"}`
	tok, _, err := ParseSlackOAuthResponse([]byte(okJSON))
	if err != nil || tok != "xoxb-test" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
	errJSON := `{"ok":false,"error":"invalid_code"}`
	if _, _, err := ParseSlackOAuthResponse([]byte(errJSON)); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewBridgeRequiresConfig(t *testing.T) {
	_, err := NewBridge(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
	cfg := config.DefaultConfig()
	_, err = NewBridge(cfg, nil, nil)
	if err == nil {
		t.Fatal("expected error when slack not ready")
	}
}
