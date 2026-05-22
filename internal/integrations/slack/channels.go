package slack

import (
	"fmt"
	"strings"

	slackapi "github.com/slack-go/slack"
)

// ChannelInfo is a Slack channel the bot can access.
type ChannelInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsPrivate bool   `json:"is_private"`
	IsMember  bool   `json:"is_member"`
}

// ListChannels returns channels the bot is a member of (or can see).
func ListChannels(api *slackapi.Client) ([]ChannelInfo, error) {
	if api == nil {
		return nil, fmt.Errorf("slack api client required")
	}
	var out []ChannelInfo
	cursor := ""
	for {
		channels, next, err := api.GetConversations(&slackapi.GetConversationsParameters{
			Types:           []string{"public_channel", "private_channel"},
			ExcludeArchived: true,
			Limit:           200,
			Cursor:          cursor,
		})
		if err != nil {
			return nil, err
		}
		for _, ch := range channels {
			out = append(out, ChannelInfo{
				ID:        ch.ID,
				Name:      ch.Name,
				IsPrivate: ch.IsPrivate,
				IsMember:  ch.IsMember,
			})
		}
		if next == "" {
			break
		}
		cursor = next
	}
	return out, nil
}

// ValidateChannel checks that channelID exists and the bot is in the channel.
func ValidateChannel(api *slackapi.Client, channelID string) error {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return fmt.Errorf("slack_channel_id required")
	}
	if api == nil {
		return fmt.Errorf("slack bridge not running")
	}
	// Use history API so validation works with channels:history / groups:history (no channels:read required).
	_, err := api.GetConversationHistory(&slackapi.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     1,
	})
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "channel_not_found") {
		return fmt.Errorf("slack channel %s not found — copy the Channel ID from Slack → channel → About", channelID)
	}
	if strings.Contains(msg, "not_in_channel") {
		return fmt.Errorf("bot is not in that channel — run /invite @your-bot there first")
	}
	if strings.Contains(msg, "missing_scope") {
		// Fall through: post path will surface a clearer error.
		return nil
	}
	return err
}

// ResolveChannelName returns the Slack channel name (without #) for a channel ID, or "" on failure.
func ResolveChannelName(api *slackapi.Client, channelID string) string {
	channelID = strings.TrimSpace(channelID)
	if api == nil || channelID == "" {
		return ""
	}
	ch, err := api.GetConversationInfo(&slackapi.GetConversationInfoInput{ChannelID: channelID})
	if err != nil || ch == nil {
		return ""
	}
	return strings.TrimSpace(ch.Name)
}
