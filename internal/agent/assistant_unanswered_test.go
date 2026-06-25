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

func TestUnansweredMessageTracker_skipsMentionedMessages(t *testing.T) {
	mockAI := ai.NewMockProvider()
	hub := channelTypeHubStub{shouldRespondTestHub: shouldRespondTestHub{}, chType: protocol.ChannelTypePublic}
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, mockAI, hub)
	tracker := newUnansweredMessageTracker(ag)

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "@BackendEngineer add theme support")
	msg.Timestamp = time.Now()
	tracker.observe(msg)

	tracker.trackerMutex.RLock()
	count := len(tracker.trackedMessages)
	tracker.trackerMutex.RUnlock()
	if count != 0 {
		t.Fatalf("expected @mention message not tracked, got %d", count)
	}
}

func TestUnansweredMessageTracker_skipsClosureMessages(t *testing.T) {
	mockAI := ai.NewMockProvider()
	hub := channelTypeHubStub{shouldRespondTestHub: shouldRespondTestHub{}, chType: protocol.ChannelTypePublic}
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, mockAI, hub)
	tracker := newUnansweredMessageTracker(ag)

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "ok thanks")
	msg.Timestamp = time.Now()
	tracker.observe(msg)

	tracker.trackerMutex.RLock()
	count := len(tracker.trackedMessages)
	tracker.trackerMutex.RUnlock()
	if count != 0 {
		t.Fatalf("expected closure message not tracked, got %d", count)
	}
}

func TestUnansweredMessageTracker_skipsJoinAnnouncements(t *testing.T) {
	mockAI := ai.NewMockProvider()
	hub := channelTypeHubStub{shouldRespondTestHub: shouldRespondTestHub{}, chType: protocol.ChannelTypePublic}
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, mockAI, hub)
	tracker := newUnansweredMessageTracker(ag)

	join := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		"general",
		protocol.AgentInfo{ID: "human-camronwood", Name: "camronwood", Type: protocol.AgentTypeGeneral},
		"Camron has joined the chat",
	)
	join.Timestamp = time.Now()
	tracker.observe(join)

	tracker.trackerMutex.RLock()
	count := len(tracker.trackedMessages)
	tracker.trackerMutex.RUnlock()
	if count != 0 {
		t.Fatalf("expected join announcement not tracked, got %d", count)
	}
}

func TestShouldTrackUnansweredUserMessage_rejectsSystemInfo(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		"general",
		protocol.AgentInfo{ID: "u", Name: "User", Type: protocol.AgentTypeGeneral},
		"Camron has joined the chat",
	)
	if shouldTrackUnansweredUserMessage(msg) {
		t.Fatal("expected system_info join line not to be tracked")
	}
}

func TestUnansweredMessageTracker_marksChannelOnAgentReply(t *testing.T) {
	mockAI := ai.NewMockProvider()
	hub := channelTypeHubStub{shouldRespondTestHub: shouldRespondTestHub{}, chType: protocol.ChannelTypePublic}
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, mockAI, hub)
	tracker := newUnansweredMessageTracker(ag)

	user := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "anyone there?")
	user.Timestamp = time.Now()
	tracker.observe(user)

	reply := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "be", Name: "BackendEngineer", Type: protocol.AgentTypeBackend}, "yes")
	reply.Timestamp = time.Now()
	tracker.observe(reply)

	tracker.trackerMutex.RLock()
	tracked := tracker.trackedMessages[user.ID]
	tracker.trackerMutex.RUnlock()
	if tracked == nil || !tracked.HasResponse {
		t.Fatal("expected latest channel message marked responded after agent reply")
	}
}

type channelTypeHubStub struct {
	shouldRespondTestHub
	chType protocol.ChannelType
}

func (h channelTypeHubStub) GetChannelType(string) protocol.ChannelType {
	return h.chType
}
