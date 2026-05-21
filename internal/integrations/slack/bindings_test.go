package slack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestNJChannelName(t *testing.T) {
	if got := NJChannelName("CABC"); got != "slack:CABC" {
		t.Fatalf("got %q", got)
	}
}

func TestBindingStoreUpsertDeleteReload(t *testing.T) {
	useTempHomeDir(t)
	s1, err := NewBindingStore()
	if err != nil {
		t.Fatal(err)
	}
	_, err = s1.Upsert(Binding{
		SlackChannelID: "C1",
		AgentID:        "a1",
		Policy:         config.SlackPolicyMentionOnly,
		Enabled:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s1.GetBySlackChannel("C1"); !ok {
		t.Fatal("expected binding after upsert")
	}
	updated, err := s1.Upsert(Binding{
		SlackChannelID: "C1",
		AgentID:        "a2",
		AgentName:      "GoExpert",
		Policy:         config.SlackPolicyAlways,
		Enabled:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.AgentID != "a2" {
		t.Fatalf("agent id %q", updated.AgentID)
	}
	s2, err := NewBindingStore()
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.List()) != 1 {
		t.Fatalf("reload list len %d", len(s2.List()))
	}
	b, ok := s2.GetByNJChannel("slack:C1")
	if !ok || b.AgentID != "a2" {
		t.Fatalf("nj lookup: ok=%v %+v", ok, b)
	}
	if err := s2.Delete("C1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.GetBySlackChannel("C1"); ok {
		t.Fatal("expected deleted")
	}
	if err := s2.Delete("missing"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestBindingStoreValidation(t *testing.T) {
	useTempHomeDir(t)
	s, err := NewBindingStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Upsert(Binding{AgentID: "a1"}); err == nil {
		t.Fatal("expected slack_channel_id required")
	}
	if _, err := s.Upsert(Binding{SlackChannelID: "C1"}); err == nil {
		t.Fatal("expected agent_id required")
	}
}

func TestBindingStoreDisabledNotReturned(t *testing.T) {
	useTempHomeDir(t)
	s, err := NewBindingStore()
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Upsert(Binding{
		SlackChannelID: "C9",
		AgentID:        "a9",
		Enabled:        false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetBySlackChannel("C9"); ok {
		t.Fatal("disabled binding should not match GetBySlackChannel")
	}
}

func TestBindingsPersistedFile(t *testing.T) {
	useTempHomeDir(t)
	s, err := NewBindingStore()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.Upsert(Binding{SlackChannelID: "C1", AgentID: "a1", Enabled: true})
	p, err := bindingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("bindings file missing: %v", err)
	}
	if filepath.Base(p) != "bindings.json" {
		t.Fatalf("path %q", p)
	}
}
