package slack

import (
	"testing"
)

func TestUserTokenStoreRoundTrip(t *testing.T) {
	useTempHomeDir(t)
	store, err := NewUserTokenStore()
	if err != nil {
		t.Fatal(err)
	}
	if store.HasToken() {
		t.Fatal("expected empty store")
	}
	if err := store.SaveToken("xoxp-test-token", "U1", "im:history,im:read"); err != nil {
		t.Fatal(err)
	}
	if !store.HasToken() {
		t.Fatal("expected token after save")
	}
	got, err := store.AccessToken()
	if err != nil {
		t.Fatal(err)
	}
	if got != "xoxp-test-token" {
		t.Fatalf("token = %q", got)
	}
	if err := store.Clear(); err != nil {
		t.Fatal(err)
	}
	if store.HasToken() {
		t.Fatal("expected cleared store")
	}
}
