package agent

import (
	"testing"
	"time"
)

func TestCollabTaskRateLimitOK_NewDispatchTokenBypassesInterval(t *testing.T) {
	a := &Agent{}
	if !a.collabTaskRateLimitOK("c1", "t1", "token-a") {
		t.Fatal("first dispatch token should be allowed")
	}
	if a.collabTaskRateLimitOK("c1", "t1", "token-a") {
		t.Fatal("same token within interval should be rate-limited")
	}
	if !a.collabTaskRateLimitOK("c1", "t1", "token-b") {
		t.Fatal("new dispatch token should bypass interval (resume/watchdog)")
	}
}

func TestCollabTaskRateLimitOK_EmptyTokenUsesInterval(t *testing.T) {
	a := &Agent{}
	if !a.collabTaskRateLimitOK("c1", "t1", "") {
		t.Fatal("first empty-token call should be allowed")
	}
	if a.collabTaskRateLimitOK("c1", "t1", "") {
		t.Fatal("second empty-token call within interval should be blocked")
	}
	time.Sleep(collabTaskMinReplyInterval + 20*time.Millisecond)
	if !a.collabTaskRateLimitOK("c1", "t1", "") {
		t.Fatal("empty-token call after interval should be allowed")
	}
}
