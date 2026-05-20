package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestListChannelsOmitsOrphanDM(t *testing.T) {
	h := NewHub()
	ag := &protocol.AgentInfo{
		ID:     "go-1",
		Name:   "GoExpert",
		Type:   protocol.AgentTypeBackend,
		Status: "active",
	}
	if err := h.RegisterAgent(ag); err != nil {
		t.Fatal(err)
	}
	ch, err := h.CreateDMChannel("camron", "go-1")
	if err != nil {
		t.Fatal(err)
	}

	names := channelNames(h.ListChannels())
	if !contains(names, ch.Name) {
		t.Fatalf("expected DM %q in list, got %v", ch.Name, names)
	}

	if err := h.UnregisterAgent("go-1"); err != nil {
		t.Fatal(err)
	}

	names = channelNames(h.ListChannels())
	if contains(names, ch.Name) {
		t.Fatalf("orphan DM %q should be omitted after unregister, got %v", ch.Name, names)
	}
}

func channelNames(channels []*protocol.Channel) []string {
	out := make([]string, 0, len(channels))
	for _, ch := range channels {
		out = append(out, ch.Name)
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
