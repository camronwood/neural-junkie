package slack

import (
	"encoding/json"
	"testing"
)

func TestParseOAuthV2AccessTeamFields(t *testing.T) {
	raw := `{
		"ok": true,
		"access_token": "xoxb-test",
		"team": {"id": "T123", "name": "Test Workspace"},
		"bot_user_id": "U456"
	}`
	var result OAuthV2AccessResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.AccessToken != "xoxb-test" {
		t.Fatalf("token: %+v", result)
	}
	if result.Team.ID != "T123" || result.Team.Name != "Test Workspace" {
		t.Fatalf("team: %+v", result.Team)
	}
	if result.BotUserID != "U456" {
		t.Fatalf("bot_user_id = %q", result.BotUserID)
	}
}
