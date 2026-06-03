package slack

import (
	"context"
	"fmt"
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

		channels, err := listHumanDMConversations(client)
		if err != nil {
			log.Printf("[slack] human_dm poll list: %v", err)
			continue
		}

		for _, ch := range channels {
			channelID := strings.TrimSpace(ch.ID)
			if channelID == "" || channelID == botDM {
				continue
			}
			if b.isSelfDMChannel(client, ch, ownerID) && !b.humanDMChannelHasPeerMessages(client, channelID, ownerID) {
				continue
			}
			b.pollHumanDMChannel(ctx, client, &inbox, channelID, channelCursors)
		}
	}
}

// listHumanDMConversations lists IM channels from the user token. mpim is fetched
// separately so missing mpim:read does not break im:read polling.
func listHumanDMConversations(client *slackapi.Client) ([]slackapi.Channel, error) {
	if client == nil {
		return nil, fmt.Errorf("user client required")
	}
	im, err := listConversationsByType(client, "im")
	if err != nil {
		return nil, err
	}
	mpim, mpimErr := listConversationsByType(client, "mpim")
	if mpimErr != nil {
		if !isMissingScopeErr(mpimErr) {
			log.Printf("[slack] human_dm poll mpim list: %v", mpimErr)
		}
		return im, nil
	}
	return append(im, mpim...), nil
}

func listConversationsByType(client *slackapi.Client, convType string) ([]slackapi.Channel, error) {
	var out []slackapi.Channel
	cursor := ""
	for {
		channels, next, err := client.GetConversations(&slackapi.GetConversationsParameters{
			Types:           []string{convType},
			Limit:           50,
			Cursor:          cursor,
			ExcludeArchived: true,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, channels...)
		cursor = next
		if cursor == "" {
			break
		}
	}
	return out, nil
}

func (b *Bridge) isSelfDMChannel(client *slackapi.Client, ch slackapi.Channel, ownerID string) bool {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return false
	}
	if ch.User == "USLACKBOT" {
		return true
	}
	var members []string
	if client != nil && strings.TrimSpace(ch.ID) != "" {
		var err error
		members, _, err = client.GetUsersInConversation(&slackapi.GetUsersInConversationParameters{
			ChannelID: ch.ID,
			Limit:     10,
		})
		if err == nil {
			return humanDMIsNoteToSelf(ownerID, ch.User, members)
		}
	}
	// Without members API, only skip when Slack labels the IM as the owner alone.
	return strings.TrimSpace(ch.User) == ownerID
}

// humanDMIsNoteToSelf reports true for Jot/note-to-self style IMs (owner only).
// Some Slack IMs use the owner's user id in ch.User even when another person has
// messaged there; member count disambiguates those from real peer DMs.
func humanDMIsNoteToSelf(ownerID, peerUserID string, members []string) bool {
	ownerID = strings.TrimSpace(ownerID)
	peerUserID = strings.TrimSpace(peerUserID)
	if ownerID == "" {
		return false
	}
	if len(members) > 1 {
		return false
	}
	if len(members) == 1 && strings.TrimSpace(members[0]) != ownerID {
		return false
	}
	return peerUserID == ownerID || len(members) == 0
}

// humanDMChannelHasPeerMessages returns true when recent history has a message from
// someone other than the owner/bot (covers Slack IMs mislabeled as note-to-self).
func (b *Bridge) humanDMChannelHasPeerMessages(client *slackapi.Client, channelID, ownerID string) bool {
	if client == nil || strings.TrimSpace(channelID) == "" {
		return false
	}
	hist, err := client.GetConversationHistory(&slackapi.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     5,
	})
	if err != nil || len(hist.Messages) == 0 {
		return false
	}
	for _, m := range hist.Messages {
		if shouldProcessHumanDMMessage(m, ownerID, b.botUserID) {
			return true
		}
	}
	return false
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
		// Seed cursor but still handle the newest peer line so a message sent just
		// before/after restart is not dropped.
		for i := 0; i < len(hist.Messages); i++ {
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
			break
		}
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
