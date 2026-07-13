package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (a *Agent) Start(ctx context.Context, channel string) error {
	a.Context.CurrentChannel = channel
	if err := a.AddChannel(ctx, channel); err != nil {
		return err
	}

	if !a.DisableChannelDiscovery {
		// Periodic discovery: pick up channels this agent was joined to after Start.
		go a.discoverChannels(ctx)
	}

	return nil
}

// StartMultiChannel starts the agent listening on multiple channels
func (a *Agent) StartMultiChannel(ctx context.Context, channels []string) error {
	if len(channels) == 0 {
		return fmt.Errorf("at least one channel is required")
	}
	a.Context.CurrentChannel = channels[0]

	for _, ch := range channels {
		if err := a.AddChannel(ctx, ch); err != nil {
			log.Printf("[%s] Warning: failed to subscribe to channel %s: %v", a.Info.Name, ch, err)
		}
	}

	if !a.DisableChannelDiscovery {
		go a.discoverChannels(ctx)
	}
	return nil
}

// AddChannel subscribes the agent to an additional channel dynamically
func (a *Agent) AddChannel(ctx context.Context, channel string) error {
	a.channelMu.Lock()
	if cancel, exists := a.activeChannels[channel]; exists {
		cancel()
		delete(a.activeChannels, channel)
	}

	subCh, err := a.Hub.Subscribe(channel)
	if err != nil {
		a.channelMu.Unlock()
		return fmt.Errorf("failed to subscribe to channel %s: %w", channel, err)
	}

	history, err := a.bootstrapChannelHistory(channel)
	if err == nil {
		a.replaceChannelHistory(channel, history)
	}

	// Listener lifetime must not follow short-lived caller contexts (e.g. HTTP handlers).
	listenerCtx, listenerCancel := context.WithCancel(context.Background())
	a.activeChannels[channel] = listenerCancel
	a.channelMu.Unlock()

	log.Printf("[%s] Agent listening on channel: %s", a.Info.Name, channel)

	// Replay any messages that arrived before this subscription (including a
	// delayed pass for messages that land during listener startup).
	go a.replayUnrespondedHistory(listenerCtx, channel)

	go func() {
		for {
			select {
			case <-listenerCtx.Done():
				return
			case <-a.stopCh:
				return
			case msg := <-subCh:
				if msg == nil {
					return
				}
				// Cancel in-flight generation only for closure turns (e.g. "ok thanks")
				// so collab recaps and slash commands are not aborted mid-flight.
				if shouldAbortInFlightForUserMessage(msg) {
					a.AbortChannel(msg.Channel)
				}
				go a.handleMessage(listenerCtx, msg)
			}
		}
	}()

	return nil
}

// shouldAbortInFlightForUserMessage reports whether a new user line should cancel
// in-flight generations on the same channel. Limited to conversational closure so
// collab recaps and hub slash commands are not disrupted.
func shouldAbortInFlightForUserMessage(msg *protocol.Message) bool {
	if msg == nil || !protocol.IsUserLikeSender(msg.From) {
		return false
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return false
	}
	if content[0] == '/' {
		return false
	}
	return classifyConversationalClosure(content) != ClosureNone
}

const unrespondedHistoryReplayDelay = 250 * time.Millisecond

// replayUnrespondedHistory scans channel history for user messages missed before
// subscription and re-scans once after a short delay (messages can land during AddChannel).
func (a *Agent) replayUnrespondedHistory(ctx context.Context, channel string) {
	a.processUnrespondedHistory(ctx, channel)
	timer := time.NewTimer(unrespondedHistoryReplayDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-a.stopCh:
		return
	case <-timer.C:
	}
	a.processUnrespondedHistory(ctx, channel)
}

// processUnrespondedHistory scans recent history for actionable messages that
// this agent may have missed between channel join and subscription readiness.
func (a *Agent) processUnrespondedHistory(ctx context.Context, channel string) {
	history, err := a.bootstrapChannelHistory(channel)
	if err != nil || len(history) == 0 {
		return
	}

	var pending []*protocol.Message
	for i := 0; i < len(history); i++ {
		candidate := history[i]
		if candidate == nil {
			continue
		}
		if candidate.From.ID == a.Info.ID || candidate.From.Name == a.Info.Name {
			continue
		}
		// Replay missed collaboration turn handoffs (system prompts with @mention).
		if candidate.IsFromSystem() && candidate.Type == protocol.MessageTypeCollabDiscussion &&
			candidate.GetCollaborationID() != "" && isCollabTurnHandoffContent(candidate.Content) {
			a.backfillMentionsFromContent(candidate)
			if a.shouldRespond(candidate) && !messageTooOldForUnansweredReplay(candidate) {
				pending = append(pending, candidate)
			}
			continue
		}
		if !protocol.IsUserLikeSender(candidate.From) {
			continue
		}
		if candidate.Type == protocol.MessageTypeAgentStatus ||
			candidate.Type == protocol.MessageTypeAgentJoin ||
			candidate.Type == protocol.MessageTypeAgentLeave ||
			candidate.Type == protocol.MessageTypeSystemInfo {
			continue
		}
		a.backfillMentionsFromContent(candidate)
		if !a.shouldRespond(candidate) {
			continue
		}
		if messageTooOldForUnansweredReplay(candidate) {
			continue
		}
		if mentionTargetAlreadyAnswered(history, i, candidate) {
			continue
		}
		if agentRespondedToUser(history, i, a.Info.ID, a.Info.Name, candidate.ID) {
			continue
		}
		pending = append(pending, candidate)
	}

	for _, candidate := range pending {
		log.Printf("[%s] Found unanswered message in %s history, processing...", a.Info.Name, channel)
		a.handleMessage(ctx, candidate)
	}
}

// discoverChannels periodically checks for new channels this agent was added to.
// Runs every second so agents respond promptly when added to new DM channels.
func (a *Agent) discoverChannels(ctx context.Context) {
	a.discoverChannelsOnce(ctx)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.discoverChannelsOnce(ctx)
		}
	}
}

func (a *Agent) discoverChannelsOnce(ctx context.Context) {
	channels := a.Hub.GetAgentChannels(a.Info.ID)
	for _, ch := range channels {
		a.channelMu.Lock()
		_, exists := a.activeChannels[ch]
		a.channelMu.Unlock()
		if !exists {
			if err := a.AddChannel(ctx, ch); err != nil {
				log.Printf("[%s] Failed to add discovered channel %s: %v", a.Info.Name, ch, err)
			}
		}
	}
}

// Stop stops the agent
func (a *Agent) Stop() {
	close(a.stopCh)
	a.channelMu.Lock()
	for _, cancel := range a.activeChannels {
		cancel()
	}
	a.activeChannels = make(map[string]context.CancelFunc)
	a.channelMu.Unlock()
}

// handleMessage processes incoming messages and decides if/how to respond
