package hub

import (
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CanUserAccessChannel reports whether username may read/post in the channel.
func (h *Hub) CanUserAccessChannel(username, channelName string) bool {
	if h == nil {
		return false
	}
	user := slugUsername(username)
	if user == "" || user == "local" {
		return true
	}
	ch, err := h.GetChannel(channelName)
	if err != nil || ch == nil {
		return false
	}
	typ := inferChannelTypeForName(ch.Name, ch.Type)
	switch typ {
	case protocol.ChannelTypePublic:
		return true
	case protocol.ChannelTypeDM:
		return userMayAccessDMChannel(user, ch.Name, ch.CreatedBy)
	case protocol.ChannelTypeCollaboration:
		// Collaboration rooms: creator or any authenticated user on single-user installs
		if ch.CreatedBy != "" && strings.EqualFold(slugUsername(ch.CreatedBy), user) {
			return true
		}
		return true
	case protocol.ChannelTypeCustom:
		// Slack mirror channels (slack:C…) are open to all local users; bridge owns creation.
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(ch.Name)), "slack:") {
			return true
		}
		if ch.CreatedBy == "" {
			return true
		}
		if strings.EqualFold(slugUsername(ch.CreatedBy), user) {
			return true
		}
		for _, m := range ch.HumanMembers {
			if strings.EqualFold(slugUsername(m), user) {
				return true
			}
		}
		return false
	case protocol.ChannelTypeRoom:
		if ch.RoomID == "" {
			return false
		}
		room, ok := h.GetRoom(ch.RoomID)
		if !ok || room == nil {
			return false
		}
		if strings.EqualFold(slugUsername(room.HostUser), user) {
			return true
		}
		for _, m := range room.Members {
			if strings.EqualFold(slugUsername(m.Username), user) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func isAutomationPrincipal(userSlug string) bool {
	switch userSlug {
	case "apikey", "automation", "automation-admin", "local":
		return true
	default:
		return false
	}
}

func channelDMUserSlug(channelName string) string {
	name := strings.ToLower(strings.TrimSpace(channelName))
	if !strings.HasPrefix(name, "dm-") {
		return ""
	}
	parts := strings.Split(name, "-")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

func userMayAccessDMChannel(userSlug, channelName, createdBy string) bool {
	if createdBy != "" && strings.EqualFold(slugUsername(createdBy), userSlug) {
		return true
	}
	// Release-prep / scenario harness uses nj_ API keys (username "apikey") against
	// DM channels owned by ChatScenario, DeliverableJudge, etc.
	if isAutomationPrincipal(userSlug) && createdBy != "" {
		owner := slugUsername(createdBy)
		if owner != "" && owner == channelDMUserSlug(channelName) {
			return true
		}
	}
	name := strings.ToLower(strings.TrimSpace(channelName))
	if !strings.HasPrefix(name, "dm-") {
		return false
	}
	parts := strings.Split(name, "-")
	if len(parts) >= 3 && parts[1] == userSlug {
		return true
	}
	return false
}

// RequireChannelAccess writes 403 when the user cannot access the channel.
func (h *Hub) RequireChannelAccess(w http.ResponseWriter, username, channelName string) bool {
	if strings.TrimSpace(channelName) == "" {
		return true
	}
	if h.CanUserAccessChannel(username, channelName) {
		return true
	}
	http.Error(w, "Forbidden: no access to this channel", http.StatusForbidden)
	return false
}
