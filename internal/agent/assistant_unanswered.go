package agent

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// unansweredMessageTracker tracks user messages in public channels and nudges
// when no agent responds within the timeout window.
type unansweredMessageTracker struct {
	agent           *Agent
	trackedMessages map[string]*MessageTracker
	trackerMutex    sync.RWMutex
	stopTracking    chan struct{}
}

// MessageTracker tracks a user message awaiting an agent reply.
type MessageTracker struct {
	MessageID   string
	Timestamp   time.Time
	HasResponse bool
	Channel     string
	FromUser    bool
}

func newUnansweredMessageTracker(agent *Agent) *unansweredMessageTracker {
	return &unansweredMessageTracker{
		agent:           agent,
		trackedMessages: make(map[string]*MessageTracker),
		stopTracking:    make(chan struct{}),
	}
}

func (t *unansweredMessageTracker) start(ctx context.Context) {
	go t.monitorTimeouts(ctx)
}

func (t *unansweredMessageTracker) stop() {
	select {
	case <-t.stopTracking:
	default:
		close(t.stopTracking)
	}
}

func (t *unansweredMessageTracker) observe(msg *protocol.Message) {
	if msg == nil || t.agent == nil {
		return
	}
	if isUserLikeInboundMessage(msg) {
		t.trackUserMessage(msg)
	}
	if isAgentInboundMessage(msg) && msg.ReplyTo != "" {
		t.markAsResponded(msg.ReplyTo)
	}
}

func isUserLikeInboundMessage(msg *protocol.Message) bool {
	return msg.From.Type == "" || msg.From.Type == protocol.AgentTypeGeneral ||
		protocol.IsUserLikeSender(msg.From)
}

func isAgentInboundMessage(msg *protocol.Message) bool {
	return msg.From.Type != "" && msg.From.Type != protocol.AgentTypeGeneral &&
		!protocol.IsUserLikeSender(msg.From)
}

func (t *unansweredMessageTracker) trackUserMessage(msg *protocol.Message) {
	if strings.HasPrefix(msg.Content, "/") {
		return
	}
	if t.agent.Hub != nil && t.agent.effectiveChannelType(msg.Channel) != protocol.ChannelTypePublic {
		return
	}

	t.trackerMutex.Lock()
	defer t.trackerMutex.Unlock()

	t.trackedMessages[msg.ID] = &MessageTracker{
		MessageID:   msg.ID,
		Timestamp:   msg.Timestamp,
		HasResponse: false,
		Channel:     msg.Channel,
		FromUser:    true,
	}

	log.Printf("[%s] Tracking user message: %s (channel: %s)", t.agent.Info.Name, msg.ID, msg.Channel)
}

func (t *unansweredMessageTracker) markAsResponded(messageID string) {
	t.trackerMutex.Lock()
	defer t.trackerMutex.Unlock()

	if tracker, exists := t.trackedMessages[messageID]; exists {
		tracker.HasResponse = true
		log.Printf("[%s] Message %s received response", t.agent.Info.Name, messageID)
	}
}

func (t *unansweredMessageTracker) monitorTimeouts(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopTracking:
			return
		case <-ticker.C:
			t.checkTimeouts(ctx)
		}
	}
}

func (t *unansweredMessageTracker) checkTimeouts(ctx context.Context) {
	t.trackerMutex.Lock()
	defer t.trackerMutex.Unlock()

	now := time.Now()
	toDelete := []string{}

	for msgID, tracker := range t.trackedMessages {
		elapsed := now.Sub(tracker.Timestamp)

		if !tracker.HasResponse && elapsed >= 20*time.Second {
			log.Printf("[%s] No response for message %s after 20s, stepping in", t.agent.Info.Name, msgID)
			go t.respondToUnanswered(ctx, tracker)
			toDelete = append(toDelete, msgID)
		}

		if elapsed > 5*time.Minute {
			toDelete = append(toDelete, msgID)
		}
	}

	for _, msgID := range toDelete {
		delete(t.trackedMessages, msgID)
	}
}

func (t *unansweredMessageTracker) respondToUnanswered(ctx context.Context, tracker *MessageTracker) {
	if t.agent == nil {
		return
	}
	_ = ctx
	response := protocol.NewMessage(
		protocol.MessageTypeChat,
		tracker.Channel,
		t.agent.Info,
		"👋 I noticed no agents responded to your question. This chat is designed for development and technical discussions. "+
			"If you need help with the Neural Junkie app or chat features, mention @Assistant or try /help-assistant.\n"+
			"• Available commands (type /help)\n"+
			"• How to mention agents (@name or @type)\n"+
			"• Creating repo or custom expert agents\n\n"+
			"For technical questions, try mentioning specific agent types like @backend, @frontend, or @devops.",
	)
	response.ReplyTo = tracker.MessageID

	if err := t.agent.Hub.SendMessage(response); err != nil {
		log.Printf("[%s] Failed to send unanswered message response: %v", t.agent.Info.Name, err)
	}
}
