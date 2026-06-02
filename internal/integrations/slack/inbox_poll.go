package slack

import (
	"context"
	"log"
	"strings"
	"time"

	slackapi "github.com/slack-go/slack"
)

const inboxDMPollInterval = 2 * time.Second

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
				if InboundDebugEnabled() || isMissingScopeErr(err) {
					log.Printf("[slack] inbox DM poll history: %v", err)
				}
				continue
			}
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

func (b *Bridge) ensureOwnerDMChannel(inbox InboxConfig) (string, error) {
	resolved, err := b.resolveOwnerBotDMChannel(inbox.OwnerSlackUserID)
	if err != nil {
		if ch := strings.TrimSpace(inbox.SlackDMChannelID); ch != "" {
			return ch, nil
		}
		return "", err
	}
	stored := strings.TrimSpace(inbox.SlackDMChannelID)
	if stored != "" && stored != resolved {
		log.Printf("[slack] inbox DM channel corrected %s → %s (DM the bot app, not note-to-self)", stored, resolved)
		_ = b.inbox.UpdateDMChannelID(resolved)
	} else if stored == "" {
		_ = b.inbox.UpdateDMChannelID(resolved)
	}
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

func isMissingScopeErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "missing_scope")
}
