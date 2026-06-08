package slack

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const humanDMPollActiveInterval = 3
const humanDMPollIdleInterval = 30
// humanDMListCacheInterval avoids re-listing IM/mpim on every active poll (Slack rate limits).
const humanDMListCacheInterval = 60

// BuildHumanDMInboxMessage converts a polled human DM into a hub inbox message.
// When routeToAgent is false (forward mode), the message is surfaced for manual NJ reply only.
func BuildHumanDMInboxMessage(in InboundInput, inbox *InboxConfig, threads *ThreadMap, authorLabel string, routeToAgent bool, njChannel string) *protocol.Message {
	displayName := strings.TrimSpace(authorLabel)
	if displayName == "" {
		displayName = SlackUserDisplayOnly(in)
	}
	if displayName == "" {
		displayName = "someone"
	}
	content := strings.TrimSpace(StripSlackMentionMarkup(in.Text))

	forward := &ForwardMatch{
		SourceChannelID: in.ChannelID,
		SourceTS:        in.SlackTS,
		SourceThreadTS:  in.ThreadTS,
		SourceAuthor:    displayName,
	}

	msg := BuildInboxMessage(in, inbox, threads, forward, njChannel)
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["source"] = "slack_human_dm"
	msg.Metadata[protocol.SlackMetaHumanDM] = true
	msg.From.Name = displayName
	msg.Metadata[protocol.SlackMetaUserDisplayName] = displayName
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
	msg := BuildHumanDMInboxMessage(in, &active, b.threads, SlackUserDisplayOnly(in), routeToAgent, peerChannel)
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
