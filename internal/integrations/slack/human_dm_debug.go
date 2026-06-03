package slack

import (
	"fmt"
	"strings"
	"time"

	slackapi "github.com/slack-go/slack"
)

// HumanDMChannelDebug describes one IM channel visible to the user token.
type HumanDMChannelDebug struct {
	ChannelID   string `json:"channel_id"`
	PeerUserID  string `json:"peer_user_id,omitempty"`
	Members     []string `json:"members,omitempty"`
	Kind        string `json:"kind"` // peer, note_to_self, bot, slackbot, unknown
	Skipped     bool   `json:"skipped"`
	LatestUser  string `json:"latest_user,omitempty"`
	LatestText  string `json:"latest_text,omitempty"`
	LatestTS    string `json:"latest_ts,omitempty"`
}

// HumanDMDebug is returned by GET /api/slack/inbox/human-dm-debug.
type HumanDMDebug struct {
	MonitoringActive   bool                  `json:"monitoring_active"`
	UserTokenSet       bool                  `json:"user_token_set"`
	BridgeTokenSet     bool                  `json:"bridge_token_set"`
	BotDMChannelID     string                `json:"bot_dm_channel_id,omitempty"`
	OwnerUserID        string                `json:"owner_user_id,omitempty"`
	PeerChannelCount   int                   `json:"peer_channel_count"`
	Channels           []HumanDMChannelDebug `json:"channels,omitempty"`
	Hint               string                `json:"hint,omitempty"`
}

// HumanDMDebugInfo inspects IM channels visible to the user token and why each is skipped or monitored.
func (b *Bridge) HumanDMDebugInfo() (HumanDMDebug, error) {
	out := HumanDMDebug{
		BridgeTokenSet: b.userTokens != nil && b.userTokens.HasToken(),
	}
	inbox := b.inbox.Get()
	out.OwnerUserID = inbox.OwnerSlackUserID
	out.UserTokenSet = out.BridgeTokenSet
	out.MonitoringActive = ShouldMonitorHumanDMs(inbox, out.UserTokenSet, time.Now())

	botDM, _ := b.ensureOwnerDMChannel(inbox)
	out.BotDMChannelID = botDM

	client, err := b.userAPIClient()
	if err != nil {
		out.Hint = "User token not available on the running bridge: " + err.Error() + ". Try Settings → Restart Slack bridge or re-authorize."
		return out, nil
	}

	ownerID := strings.TrimSpace(inbox.OwnerSlackUserID)
	channels, err := listHumanDMConversations(client)
	if err != nil {
		return out, fmt.Errorf("conversations.list: %w", err)
	}
	for _, ch := range channels {
		entry := HumanDMChannelDebug{ChannelID: ch.ID, PeerUserID: ch.User}
		selfDM := b.isSelfDMChannel(client, ch, ownerID) && !b.humanDMChannelHasPeerMessages(client, ch.ID, ownerID)
		entry.Kind, entry.Skipped = classifyHumanDMChannel(ch.ID, ch.User, botDM, ownerID, selfDM)
		if !entry.Skipped {
			out.PeerChannelCount++
		}
		if hist, err := client.GetConversationHistory(&slackapi.GetConversationHistoryParameters{
			ChannelID: ch.ID,
			Limit:     1,
		}); err == nil && len(hist.Messages) > 0 {
			m := hist.Messages[0]
			entry.LatestUser = m.User
			entry.LatestText = strings.TrimSpace(m.Text)
			entry.LatestTS = m.Timestamp
		}
		out.Channels = append(out.Channels, entry)
	}

	switch {
	case !inbox.HumanDMAway.Enabled:
		out.Hint = "Enable Human DM away mode in Settings and save."
	case !out.UserTokenSet:
		out.Hint = "Click Authorize Slack DM access in Settings."
	case !out.MonitoringActive:
		out.Hint = "Turn on I'm away now, or enable Schedule while outside work hours."
	case out.PeerChannelCount == 0:
		out.Hint = "No peer 1:1 DMs found. Have someone DM you directly on Slack — not note-to-self (Jot Something Down) and not the NJ bot."
	default:
		out.Hint = fmt.Sprintf("Monitoring %d peer DM channel(s). New messages from others while away should reach your inbox agent.", out.PeerChannelCount)
	}
	return out, nil
}

func classifyHumanDMChannel(channelID, peerUserID, botDM, ownerID string, selfDM bool) (kind string, skipped bool) {
	channelID = strings.TrimSpace(channelID)
	if channelID == botDM {
		return "bot", true
	}
	if peerUserID == "USLACKBOT" {
		return "slackbot", true
	}
	if selfDM {
		return "note_to_self", true
	}
	if strings.TrimSpace(peerUserID) == ownerID {
		// ch.User is owner but members API shows a real peer conversation.
		return "peer", false
	}
	return "peer", false
}
