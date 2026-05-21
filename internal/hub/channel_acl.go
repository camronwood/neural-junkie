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
	default:
		return true
	}
}

func userMayAccessDMChannel(userSlug, channelName, createdBy string) bool {
	if createdBy != "" && strings.EqualFold(slugUsername(createdBy), userSlug) {
		return true
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
