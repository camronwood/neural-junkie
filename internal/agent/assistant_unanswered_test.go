package agent

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMessageAsksAboutNJPlatformHelp_chatKeywords(t *testing.T) {
	if !messageAsksAboutNJPlatformHelp("how do I create a repo agent?") {
		t.Fatal("expected platform help keyword match")
	}
	if messageAsksAboutNJPlatformHelp("what do you think about rust?") {
		t.Fatal("did not expect casual opinion to match platform help")
	}
}

func TestAssistantPublicShouldRespond_mentionOnly(t *testing.T) {
	mockAI := ai.NewMockProvider()
	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, mockAI, hub)
	ag.Info.ID = "asst-1"

	help := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "how do I mention an agent?")
	if !assistantPublicShouldRespond(ag, help) {
		t.Fatal("expected NJ platform help on public channel")
	}

	casual := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "what do you think about go vs rust?")
	if assistantPublicShouldRespond(ag, casual) {
		t.Fatal("expected casual public chat to stay silent")
	}

	mentioned := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "@Assistant hello")
	mentioned.Mentions = []string{"asst-1"}
	if !assistantPublicShouldRespond(ag, mentioned) {
		t.Fatal("expected mention response")
	}
}

func TestUnansweredMessageTracker_tracksUserMessage(t *testing.T) {
	mockAI := ai.NewMockProvider()
	hub := channelTypeHubStub{shouldRespondTestHub: shouldRespondTestHub{}, chType: protocol.ChannelTypePublic}
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, mockAI, hub)
	tracker := newUnansweredMessageTracker(ag)

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "anyone there?")
	msg.Timestamp = time.Now()
	tracker.observe(msg)

	tracker.trackerMutex.RLock()
	_, ok := tracker.trackedMessages[msg.ID]
	tracker.trackerMutex.RUnlock()
	if !ok {
		t.Fatal("expected user message to be tracked")
	}
}

type channelTypeHubStub struct {
	shouldRespondTestHub
	chType protocol.ChannelType
}

func (h channelTypeHubStub) GetChannelType(string) protocol.ChannelType {
	return h.chType
}
