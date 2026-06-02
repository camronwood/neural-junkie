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

func TestParseOAuthV2AccessUserToken(t *testing.T) {
	raw := `{
		"ok": true,
		"access_token": "xoxb-bot",
		"authed_user": {
			"id": "U1",
			"access_token": "xoxp-user",
			"scope": "im:history,im:read,chat:write,users:read"
		}
	}`
	var result OAuthV2AccessResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	if result.AuthedUser.AccessToken != "xoxp-user" {
		t.Fatalf("user token = %q", result.AuthedUser.AccessToken)
	}
	if result.AuthedUser.Scope != "im:history,im:read,chat:write,users:read" {
		t.Fatalf("user scope = %q", result.AuthedUser.Scope)
	}
}
