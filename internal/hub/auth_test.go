package hub

import (
	"net/http/httptest"
	"testing"
)

func TestCanUserAccessDMChannel(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("dm-camron-assistant", "dm", "", "dm", "camron")
	if !h.CanUserAccessChannel("Camron", "dm-camron-assistant") {
		t.Fatal("owner should access dm")
	}
	if h.CanUserAccessChannel("Eve", "dm-camron-assistant") {
		t.Fatal("other user should not access dm")
	}
}

func TestCanUserAccessSlackMirrorChannel(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("slack:C01234", "Slack: #test", "", "custom", "slack-bridge")
	if !h.CanUserAccessChannel("Camron", "slack:C01234") {
		t.Fatal("any user should access slack mirror channel")
	}
	if !h.CanUserAccessChannel("eve", "slack:C01234") {
		t.Fatal("slack mirror channel is not private to creator")
	}
}

func TestCanUserAccessPrivateCustomChannel(t *testing.T) {
	h := NewHub()
	ch := h.CreateChannelWithType("secret-proj", "x", "", "custom", "alice")
	ch.HumanMembers = []string{"bob"}
	if !h.CanUserAccessChannel("alice", "secret-proj") {
		t.Fatal("creator")
	}
	if !h.CanUserAccessChannel("bob", "secret-proj") {
		t.Fatal("member")
	}
	if h.CanUserAccessChannel("eve", "secret-proj") {
		t.Fatal("non-member denied")
	}
}

func TestSessionManagerCreateValidate(t *testing.T) {
	sm := NewSessionManager()
	s := sm.CreateSession("Camron Wood")
	if s.Token == "" {
		t.Fatal("token")
	}
	r := httptest.NewRequest("GET", "/api/auth/session", nil)
	r.Header.Set("X-NJ-Session", s.Token)
	got := SessionFromRequest(r, sm)
	if got == nil || got.Username != "Camron Wood" {
		t.Fatalf("session %+v", got)
	}
}
