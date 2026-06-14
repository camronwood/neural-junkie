package slack

import (
	"context"
	"log"
	"strings"
	"time"

	slackapi "github.com/slack-go/slack"
)

const inboxDMPollInterval = 5 * time.Second
const inboxDMResolveBackoff = 5 * time.Minute

// runInboxDMPoll watches the owner DM via conversations.history when Socket Mode
// does not deliver message.im (common if Event Subscriptions omit message.im).
func (b *Bridge) runInboxDMPoll(ctx context.Context) {
	ticker := time.NewTicker(inboxDMPollInterval)
	defer ticker.Stop()

	var lastSeenTS string
	seeded := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			inbox := b.inbox.Get()
			if !inbox.Enabled || strings.TrimSpace(inbox.OwnerSlackUserID) == "" {
				lastSeenTS = ""
				seeded = false
				continue
			}

			if b.botRateLimited() {
				continue
			}

			channelID, err := b.ensureOwnerDMChannel(inbox)
			if err != nil || channelID == "" {
				if InboundDebugEnabled() {
					log.Printf("[slack] inbox DM poll: open channel: %v", err)
				}
				continue
			}

			params := &slackapi.GetConversationHistoryParameters{
				ChannelID: channelID,
				Limit:     10,
			}
			if seeded && lastSeenTS != "" {
				params.Oldest = lastSeenTS
				params.Inclusive = false
			}

			hist, err := b.api.GetConversationHistory(params)
			if err != nil {
				b.noteBotRateLimit(err)
				if isSlackRateLimited(err) {
					log.Printf("[slack] inbox DM poll history: rate limited, backing off")
					continue
				}
				if isChannelNotFoundErr(err) {
					b.handleStaleInboxDMChannel(inbox)
					continue
				}
				if InboundDebugEnabled() || isMissingScopeErr(err) {
					log.Printf("[slack] inbox DM poll history: %v", err)
				}
				continue
			}
			b.clearInboxResolveBackoff()
			if len(hist.Messages) == 0 {
				if !seeded {
					seeded = true
				}
				continue
			}

			if !seeded {
				lastSeenTS = hist.Messages[0].Timestamp
				seeded = true
				log.Printf("[slack] inbox DM poll seeded at ts=%s (enable message.im for instant delivery)", lastSeenTS)
				continue
			}

			// History is newest-first; process oldest-first.
			for i := len(hist.Messages) - 1; i >= 0; i-- {
				m := hist.Messages[i]
				if !shouldProcessPolledDM(m, b.botUserID) {
					continue
				}
				in := InboundInput{
					WorkspaceID: b.teamID,
					ChannelID:   channelID,
					UserID:      m.User,
					Text:        m.Text,
					SlackTS:     m.Timestamp,
					ThreadTS:    m.ThreadTimestamp,
					BotID:       m.BotID,
					Subtype:     m.SubType,
				}
				log.Printf("[slack] inbox DM poll → hub channel=%s user=%s", channelID, m.User)
				b.routeInbound(ctx, in)
			}

			if ts := hist.Messages[0].Timestamp; ts != "" {
				lastSeenTS = ts
			}
		}
	}
}

// ensureOwnerDMChannel returns the owner↔bot DM channel id without calling Slack when already stored.
func (b *Bridge) ensureOwnerDMChannel(inbox InboxConfig) (string, error) {
	if ch := strings.TrimSpace(inbox.SlackDMChannelID); ch != "" {
		return ch, nil
	}
	if b.inboxResolveBackedOff() {
		return "", nil
	}
	return b.resolveAndPersistOwnerDMChannel(inbox)
}

func (b *Bridge) handleStaleInboxDMChannel(inbox InboxConfig) {
	stale := strings.TrimSpace(inbox.SlackDMChannelID)
	if stale != "" {
		log.Printf("[slack] inbox DM channel %s invalid — clearing cached id (DM the bot app once in Slack)", stale)
		_ = b.inbox.ClearDMChannelID()
	}
	if b.inboxResolveBackedOff() {
		return
	}
	if _, err := b.resolveAndPersistOwnerDMChannel(inbox); err != nil {
		b.noteBotRateLimit(err)
		b.setInboxResolveBackoff(inboxDMResolveBackoff)
		if isMissingScopeErr(err) {
			log.Printf("[slack] inbox DM resolve: %v — add bot scope im:write, reinstall app, DM the bot once", err)
		} else if InboundDebugEnabled() || !isSlackRateLimited(err) {
			log.Printf("[slack] inbox DM resolve: %v", err)
		}
	}
}

func (b *Bridge) resolveAndPersistOwnerDMChannel(inbox InboxConfig) (string, error) {
	resolved, err := b.resolveOwnerBotDMChannel(inbox.OwnerSlackUserID)
	if err != nil {
		return "", err
	}
	stored := strings.TrimSpace(inbox.SlackDMChannelID)
	if stored != "" && stored != resolved {
		log.Printf("[slack] inbox DM channel corrected %s → %s (DM the bot app, not note-to-self)", stored, resolved)
	}
	if stored != resolved {
		_ = b.inbox.UpdateDMChannelID(resolved)
	}
	b.clearInboxResolveBackoff()
	return resolved, nil
}

func shouldProcessPolledDM(m slackapi.Message, botUserID string) bool {
	msg := m.Msg
	if strings.TrimSpace(msg.Text) == "" {
		return false
	}
	if msg.BotID != "" {
		return false
	}
	if botUserID != "" && strings.TrimSpace(msg.User) == botUserID {
		return false
	}
	switch msg.SubType {
	case "bot_message", "message_changed", "message_deleted", "channel_join", "channel_leave":
		return false
	}
	return true
}
