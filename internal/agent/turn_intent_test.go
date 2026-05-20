package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClassifyTurnIntent_closureThanks(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "User"}, "ok thanks")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", nil); got != IntentClosure {
		t.Fatalf("got %v, want closure", got)
	}
}

func TestClassifyTurnIntent_alreadySaid(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "User"}, "I know you said that already")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", nil); got != IntentClosure {
		t.Fatalf("got %v, want closure", got)
	}
}

func TestClassifyTurnIntent_lowSignal(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "User"}, "nice")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", nil); got != IntentLowSignal {
		t.Fatalf("got %v, want low_signal", got)
	}
}

func TestClassifyTurnIntent_collabNotLowSignal(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "collab-1", protocol.AgentInfo{ID: "u", Name: "User"}, "ok")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeCollaboration, "goexpert", nil); got != IntentSubstantive {
		t.Fatalf("got %v, want substantive on collab", got)
	}
}

func TestClassifyTurnIntent_substantiveDistance(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "User"},
		"How far is it from Collinsville IL to St Louis MO?")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", nil); got != IntentSubstantive {
		t.Fatalf("got %v, want substantive", got)
	}
}

func TestMaxHistoryForIntent_withSummary(t *testing.T) {
	if maxHistoryForIntent(IntentSubstantive, true) != 4 {
		t.Fatal("expected 4 history rows when summary present")
	}
}
