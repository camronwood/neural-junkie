package slack

import "testing"

func TestClassifyHumanDMChannel(t *testing.T) {
	botDM := "D_BOT"
	owner := "U_OWNER"
	tests := []struct {
		channelID, peerUserID string
		self                  bool
		wantKind              string
		skipped               bool
	}{
		{"D_BOT", "U_BOT", false, "bot", true},
		{"D1", owner, true, "note_to_self", true},
		{"D1b", owner, false, "peer", false},
		{"D2", "USLACKBOT", false, "slackbot", true},
		{"D3", "U_PEER", false, "peer", false},
	}
	for _, tc := range tests {
		kind, skipped := classifyHumanDMChannel(tc.channelID, tc.peerUserID, botDM, owner, tc.self)
		if kind != tc.wantKind || skipped != tc.skipped {
			t.Fatalf("%s/%s: got kind=%q skipped=%v want kind=%q skipped=%v", tc.channelID, tc.peerUserID, kind, skipped, tc.wantKind, tc.skipped)
		}
	}
}

func TestUserTokenStoreReload(t *testing.T) {
	useTempHomeDir(t)
	store, err := NewUserTokenStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveToken("xoxp-one", "U1", "im:read"); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.data.AccessToken = ""
	store.mu.Unlock()
	if store.HasToken() {
		t.Fatal("expected cleared in-memory token")
	}
	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	if !store.HasToken() {
		t.Fatal("expected token after reload")
	}
	got, err := store.AccessToken()
	if err != nil || got != "xoxp-one" {
		t.Fatalf("token after reload: %q err=%v", got, err)
	}
}
