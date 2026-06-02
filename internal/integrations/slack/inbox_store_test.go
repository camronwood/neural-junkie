package slack

import (
	"encoding/json"
	"testing"
)

func TestOAuthV2AccessResponseAuthedUser(t *testing.T) {
	raw := `{
		"ok": true,
		"access_token": "xoxb-test",
		"bot_user_id": "B1",
		"authed_user": {"id": "UOWNER", "name": "camron"},
		"team": {"id": "T1", "name": "Test"}
	}`
	var resp OAuthV2AccessResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.AuthedUser.ID != "UOWNER" {
		t.Fatalf("owner id %q", resp.AuthedUser.ID)
	}
}

func TestSeedInboxFromInstall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := SeedInboxFromInstall("U1", "Camron"); err != nil {
		t.Fatal(err)
	}
	store, err := NewInboxStore()
	if err != nil {
		t.Fatal(err)
	}
	cfg := store.Get()
	if cfg.OwnerSlackUserID != "U1" {
		t.Fatalf("owner %q", cfg.OwnerSlackUserID)
	}
	if cfg.NJChannel != "slack:inbox:U1" {
		t.Fatalf("channel %q", cfg.NJChannel)
	}
	if len(cfg.ForwardRules) == 0 {
		t.Fatal("expected default forward rules")
	}
}

func TestInboxStoreSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	store, err := NewInboxStore()
	if err != nil {
		t.Fatal(err)
	}
	saved, err := store.Save(InboxConfig{
		Enabled:          true,
		OwnerSlackUserID: "U1",
		AgentID:          "a1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.NJChannel != "slack:inbox:U1" {
		t.Fatalf("channel %q", saved.NJChannel)
	}
	if !saved.ReplyInThread {
		t.Fatal("expected reply in thread")
	}
}
