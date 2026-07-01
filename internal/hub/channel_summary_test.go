package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestChannelSnapshot_summaryRoundTrip(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("dm-test", "test", "", protocol.ChannelTypeDM, "system")
	h.channelContext["dm-test"] = &ChannelContextState{
		Summary:   "User asked about CRISPR.",
		UpdatedAt: time.Now().UTC(),
	}
	snap := h.TakeSessionSnapshot()
	cs := snap.Channels["dm-test"]
	if cs == nil || cs.SessionSummary != "User asked about CRISPR." {
		t.Fatalf("snapshot summary missing: %+v", cs)
	}
}

func TestClearChannelHistory_clearsSummary(t *testing.T) {
	h := NewHub()
	name := "dm-clear-test"
	h.CreateChannelWithType(name, "test", "", protocol.ChannelTypeDM, "system")
	h.channelContext[name] = &ChannelContextState{Summary: "old facts", UserTurns: 2}
	if err := h.ClearChannelHistory(name); err != nil {
		t.Fatal(err)
	}
	if s := h.GetChannelSessionSummary(name); s != "" {
		t.Fatalf("summary should be cleared, got %q", s)
	}
}

func TestNoteChannelActivity_userTurnCounter(t *testing.T) {
	h := NewHub()
	name := "dm-turns"
	h.CreateChannelWithType(name, "test", "", protocol.ChannelTypeDM, "system")
	user := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	for i := 0; i < 2; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeChat, name, user, "question")
		_ = h.SendMessage(msg)
	}
	h.mu.RLock()
	st := h.channelContext[name]
	h.mu.RUnlock()
	if st == nil || st.UserTurns < 2 {
		t.Fatalf("expected UserTurns >= 2, got %+v", st)
	}
}

func TestNoteChannelActivity_summaryRefresh(t *testing.T) {
	h := NewHub()
	name := "dm-summary"
	h.CreateChannelWithType(name, "test", "", protocol.ChannelTypeDM, "system")
	h.SetChannelSummaryGenerator(func(transcript string) (string, error) {
		return "summary: " + transcript[:min(20, len(transcript))], nil
	}, "test-model")

	user := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	agent := protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeChat, name, user, "What is CRISPR?"))
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeAnswer, name, agent, "CRISPR is gene editing."))
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeChat, name, user, "follow up one"))
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeChat, name, user, "follow up two"))
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeChat, name, user, "follow up three"))
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeAnswer, name, agent, "More detail here."))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s := h.GetChannelSessionSummary(name); s != "" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("expected summary to be generated")
}

func TestScheduleImmediateSummaryRefresh(t *testing.T) {
	h := NewHub()
	name := "dm-approval"
	h.CreateChannelWithType(name, "test", "", protocol.ChannelTypeDM, "system")
	done := make(chan struct{}, 1)
	h.SetChannelSummaryGenerator(func(transcript string) (string, error) {
		done <- struct{}{}
		return "export approved", nil
	}, "test-model")

	user := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	agent := protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeChat, name, user, "save article to docs/test.md"))
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeAnswer, name, agent, "Proposing file change."))

	h.scheduleImmediateSummaryRefresh(name)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected immediate summary refresh on approval scheduling")
	}
	if s := h.GetChannelSessionSummary(name); s == "" {
		time.Sleep(100 * time.Millisecond)
		if s = h.GetChannelSessionSummary(name); s == "" {
			t.Fatal("expected summary after immediate refresh")
		}
	}
}

func TestNoteChannelActivity_publicChannelEligible(t *testing.T) {
	h := NewHub()
	name := "general"
	h.CreateChannelWithType(name, "General", "", protocol.ChannelTypePublic, "system")
	if !channelMaintainsSessionSummary(protocol.ChannelTypePublic, name) {
		t.Fatal("public channel should maintain session summary")
	}
	user := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	msg := protocol.NewMessage(protocol.MessageTypeChat, name, user, "hello team")
	h.noteChannelActivity(msg)
	h.mu.RLock()
	st := h.channelContext[name]
	h.mu.RUnlock()
	if st == nil || st.UserTurns != 1 {
		t.Fatalf("expected user turn counted on public channel, got %+v", st)
	}
}

func TestNoteChannelActivity_regressionChannelSkipped(t *testing.T) {
	h := NewHub()
	name := "implement-scenarios"
	h.CreateChannelWithType(name, "Implement regression", "", protocol.ChannelTypePublic, "system")
	called := make(chan struct{}, 1)
	h.SetChannelSummaryGenerator(func(transcript string) (string, error) {
		called <- struct{}{}
		return "should not run", nil
	}, "test-model")

	user := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	agent := protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	for i := 0; i < 4; i++ {
		_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeChat, name, user, "implement feature please"))
	}
	_ = h.SendMessage(protocol.NewMessage(protocol.MessageTypeAnswer, name, agent, "Implementation session complete."))

	select {
	case <-called:
		t.Fatal("regression harness channel should not schedule session summary")
	case <-time.After(200 * time.Millisecond):
	}
	h.mu.RLock()
	st := h.channelContext[name]
	h.mu.RUnlock()
	if st != nil && st.UserTurns > 0 {
		t.Fatalf("expected no user turn tracking on regression channel, got %+v", st)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
