package slack

import (
	"fmt"
	"strings"

	slackapi "github.com/slack-go/slack"
)

// resolveOwnerBotDMChannel returns the IM channel between the inbox owner and this bot.
func (b *Bridge) resolveOwnerBotDMChannel(ownerUserID string) (string, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return "", fmt.Errorf("owner_slack_user_id required")
	}
	if b.botUserID == "" {
		return "", fmt.Errorf("bot user id not available")
	}
	ch, _, _, err := b.api.OpenConversation(&slackapi.OpenConversationParameters{
		Users: []string{ownerUserID, b.botUserID},
	})
	if err != nil {
		return "", err
	}
	if ch == nil || strings.TrimSpace(ch.ID) == "" {
		return "", fmt.Errorf("open conversation returned empty channel")
	}
	return ch.ID, nil
}

// InboxDMDebug is returned by GET /api/slack/inbox/dm-debug.
type InboxDMDebug struct {
	StoredChannelID  string              `json:"stored_channel_id,omitempty"`
	BotDMChannelID   string              `json:"bot_dm_channel_id,omitempty"`
	ChannelCorrect   bool                `json:"channel_correct"`
	BotUserID        string              `json:"bot_user_id,omitempty"`
	OwnerUserID      string              `json:"owner_user_id,omitempty"`
	Members          []string            `json:"members,omitempty"`
	RecentMessages   []InboxDMMessage    `json:"recent_messages,omitempty"`
	Hint             string              `json:"hint,omitempty"`
}

type InboxDMMessage struct {
	User string `json:"user,omitempty"`
	Text string `json:"text,omitempty"`
	TS   string `json:"ts,omitempty"`
}

// InboxDMDebugInfo inspects the owner↔bot DM used by the personal inbox.
func (b *Bridge) InboxDMDebugInfo() (InboxDMDebug, error) {
	out := InboxDMDebug{BotUserID: b.botUserID}
	inbox := b.inbox.Get()
	out.OwnerUserID = inbox.OwnerSlackUserID
	out.StoredChannelID = inbox.SlackDMChannelID

	resolved, err := b.resolveOwnerBotDMChannel(inbox.OwnerSlackUserID)
	if err != nil {
		return out, err
	}
	out.BotDMChannelID = resolved
	out.ChannelCorrect = out.StoredChannelID == "" || out.StoredChannelID == resolved

	if mem, _, err := b.api.GetUsersInConversation(&slackapi.GetUsersInConversationParameters{
		ChannelID: resolved,
		Limit:     10,
	}); err == nil {
		out.Members = mem
	}

	hist, err := b.api.GetConversationHistory(&slackapi.GetConversationHistoryParameters{
		ChannelID: resolved,
		Limit:     8,
	})
	if err != nil {
		return out, err
	}
	for _, m := range hist.Messages {
		out.RecentMessages = append(out.RecentMessages, InboxDMMessage{
			User: m.User,
			Text: strings.TrimSpace(m.Text),
			TS:   m.Timestamp,
		})
	}

	out.Hint = "Open Slack → Direct Messages → find the app/bot (e.g. @neural_junkie), not “Jot Something Down” / note-to-self. " +
		"Your messages must appear in recent_messages below after you DM the bot."
	if !out.ChannelCorrect {
		out.Hint = "Stored DM channel id was wrong and will be corrected on the next poll. " + out.Hint
	}
	return out, nil
}
