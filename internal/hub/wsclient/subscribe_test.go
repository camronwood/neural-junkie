package wsclient

import (
	"net/url"
	"testing"
	"time"
)

func TestChannelWSURL(t *testing.T) {
	got, err := channelWSURL("http://localhost:18765", "general", []string{"dev", "collab-1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "ws" || u.Host != "localhost:18765" || u.Path != "/ws" {
		t.Fatalf("url = %q", got)
	}
	q := u.Query()
	if q.Get("channel") != "general" {
		t.Fatalf("channel = %q", q.Get("channel"))
	}
	if q.Get("extra") != "dev,collab-1" {
		t.Fatalf("extra = %q", q.Get("extra"))
	}
}

func TestAgentWSURL(t *testing.T) {
	got, err := agentWSURL("https://hub.example.com", "agent-123")
	if err != nil {
		t.Fatal(err)
	}
	if got != "wss://hub.example.com/api/agents/ws?agent_id=agent-123" {
		t.Fatalf("got %q", got)
	}
}

func TestNextBackoff(t *testing.T) {
	if nextBackoff(defaultReconnectMin) != time.Second {
		t.Fatalf("expected 1s")
	}
	if nextBackoff(defaultReconnectMax) != defaultReconnectMax {
		t.Fatalf("cap exceeded")
	}
}
