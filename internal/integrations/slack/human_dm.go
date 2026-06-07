package slack

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const humanDMPollActiveInterval = 3
const humanDMPollIdleInterval = 30

// BuildHumanDMInboxMessage converts a polled human DM into a hub inbox message.
// When routeToAgent is false (forward mode), the message is surfaced for manual NJ reply only.
func BuildHumanDMInboxMessage(in InboundInput, inbox *InboxConfig, threads *ThreadMap, authorLabel string, routeToAgent bool, njChannel string) *protocol.Message {
	content := StripSlackMentionMarkup(in.Text)
	if authorLabel == "" {
		authorLabel = in.UserName
	}
	if authorLabel == "" {
		authorLabel = "someone"
	}
	header := fmt.Sprintf("[DM from %s]", authorLabel)
	content = header + "\n" + strings.TrimSpace(content)

	forward := &ForwardMatch{
		SourceChannelID: in.ChannelID,
		SourceTS:        in.SlackTS,
		SourceThreadTS:  in.ThreadTS,
		SourceAuthor:    authorLabel,
	}

	msg := BuildInboxMessage(in, inbox, threads, forward, njChannel)
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["source"] = "slack_human_dm"
	msg.Metadata[protocol.SlackMetaHumanDM] = true
	msg.Content = content
	msg.Metadata[protocol.SlackMetaReplyChannelID] = in.ChannelID
	// Human DMs reply on the main timeline — never as a Slack thread.
	delete(msg.Metadata, protocol.SlackMetaReplyThreadTS)
	if !routeToAgent {
		ApplyInboxManualReply(msg)
	}
	return msg
}

func (b *Bridge) processHumanDMInbound(ctx context.Context, in InboundInput, inbox *InboxConfig) {
	if inbox == nil || !inbox.Enabled {
		return
	}
	dedupeKey := "human:" + in.ChannelID + ":" + in.SlackTS
	if in.SlackTS != "" {
		if _, loaded := b.seenInbound.LoadOrStore(dedupeKey, struct{}{}); loaded {
			return
		}
	}

	active := *inbox
	if resolved, err := b.hub.ResolveAgentID(active.AgentID, active.AgentName); err == nil {
		active.AgentID = resolved
	}

	b.resolveInboundUserIdentity(&in)
	userTokenSet := b.userTokens != nil && b.userTokens.HasToken()
	routeToAgent := ShouldAutoReplyHumanDMs(active, userTokenSet, time.Now())
	peerChannel, created, err := EnsureInboxPeerChannel(ctx, b.hub, nil, active, in.UserID, SlackUserDisplayOnly(in))
	if err != nil {
		log.Printf("[slack] ensure peer inbox channel: %v", err)
		peerChannel = active.NJChannel
	}
	msg := BuildHumanDMInboxMessage(in, &active, b.threads, in.UserName, routeToAgent, peerChannel)
	if err := b.hub.SendMessage(msg); err != nil {
		log.Printf("[slack] human_dm SendMessage: %v", err)
		return
	}
	if created {
		b.refreshInboxOutboundSubscription()
	}

	replyThread := ""
	_ = b.threads.RegisterHumanDMReplyRoute(peerChannel, in.ChannelID, "")
	if ch := msg.SlackReplyChannelID(); ch != "" {
		routeKey := msg.ID
		if msg.ThreadID != "" {
			routeKey = msg.ThreadID
		}
		_ = b.threads.RegisterHumanDMReplyRoute(routeKey, ch, replyThread)
		if msg.ID != "" && routeKey != msg.ID {
			_ = b.threads.RegisterHumanDMReplyRoute(msg.ID, ch, replyThread)
		}
	}
	b.registerBindingThreadState(in, msg)
	log.Printf("[slack] human_dm → hub %s from %s", peerChannel, in.UserName)
}
