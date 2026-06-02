package slack

import (
	"context"
	"log"
	"strings"
	"time"

	slackapi "github.com/slack-go/slack"
)

func (b *Bridge) runHumanDMPoll(ctx context.Context) {
	var lastPoll time.Time
	var lastMonitoring bool
	channelCursors := map[string]string{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		inbox := b.inbox.Get()
		userTokenSet := b.userTokens != nil && b.userTokens.HasToken()
		monitoring := ShouldMonitorHumanDMs(inbox, userTokenSet, time.Now())
		if monitoring != lastMonitoring {
			if monitoring {
				log.Printf("[slack] human_dm poll: monitoring active (away=%v schedule=%v)", inbox.HumanDMAway.AwayEnabled, inbox.HumanDMAway.ScheduleEnabled)
			} else {
				log.Printf("[slack] human_dm poll: monitoring inactive")
			}
			lastMonitoring = monitoring
			if monitoring {
				channelCursors = map[string]string{}
			}
		}
		if !monitoring {
			time.Sleep(humanDMPollIdleInterval * time.Second)
			continue
		}

		if time.Since(lastPoll) < humanDMPollActiveInterval*time.Second {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		lastPoll = time.Now()

		client, err := b.userAPIClient()
		if err != nil || client == nil {
			if InboundDebugEnabled() {
				log.Printf("[slack] human_dm poll: user client: %v", err)
			}
			time.Sleep(humanDMPollIdleInterval * time.Second)
			continue
		}

		botDM, _ := b.ensureOwnerDMChannel(inbox)
		ownerID := strings.TrimSpace(inbox.OwnerSlackUserID)

		cursor := ""
		for {
			channels, next, err := client.GetConversations(&slackapi.GetConversationsParameters{
				Types:           []string{"im"},
				Limit:           50,
				Cursor:          cursor,
				ExcludeArchived: true,
			})
			if err != nil {
				log.Printf("[slack] human_dm poll list: %v", err)
				break
			}

			for _, ch := range channels {
				channelID := strings.TrimSpace(ch.ID)
				if channelID == "" || channelID == botDM {
					continue
				}
				if b.isSelfDMChannel(client, ch, ownerID) {
					continue
				}
				b.pollHumanDMChannel(ctx, client, &inbox, channelID, channelCursors)
			}

			cursor = next
			if cursor == "" {
				break
			}
		}
	}
}

func (b *Bridge) isSelfDMChannel(client *slackapi.Client, ch slackapi.Channel, ownerID string) bool {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return false
	}
	if strings.TrimSpace(ch.User) == ownerID {
		return true
	}
	if ch.User == "USLACKBOT" {
		return true
	}
	if client == nil || strings.TrimSpace(ch.ID) == "" {
		return false
	}
	members, _, err := client.GetUsersInConversation(&slackapi.GetUsersInConversationParameters{
		ChannelID: ch.ID,
		Limit:     10,
	})
	if err != nil {
		return false
	}
	return len(members) <= 1
}

func (b *Bridge) pollHumanDMChannel(ctx context.Context, client *slackapi.Client, inbox *InboxConfig, channelID string, cursors map[string]string) {
	oldest := cursors[channelID]
	params := &slackapi.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     10,
	}
	if oldest != "" {
		params.Oldest = oldest
		params.Inclusive = false
	}

	hist, err := client.GetConversationHistory(params)
	if err != nil {
		if InboundDebugEnabled() {
			log.Printf("[slack] human_dm poll history %s: %v", channelID, err)
		}
		return
	}
	if len(hist.Messages) == 0 {
		return
	}

	if oldest == "" {
		cursors[channelID] = hist.Messages[0].Timestamp
		return
	}

	for i := len(hist.Messages) - 1; i >= 0; i-- {
		m := hist.Messages[i]
		if !shouldProcessHumanDMMessage(m, inbox.OwnerSlackUserID, b.botUserID) {
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
		b.processHumanDMInbound(ctx, in, inbox)
	}

	if ts := hist.Messages[0].Timestamp; ts != "" {
		cursors[channelID] = ts
	}
}

func shouldProcessHumanDMMessage(m slackapi.Message, ownerID, botUserID string) bool {
	msg := m.Msg
	if strings.TrimSpace(msg.Text) == "" {
		return false
	}
	if msg.BotID != "" {
		return false
	}
	if ownerID != "" && strings.TrimSpace(msg.User) == ownerID {
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
