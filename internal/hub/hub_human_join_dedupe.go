package hub

import (
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const humanJoinDedupeWindow = 5 * time.Minute

func isHumanJoinAnnouncement(msg *protocol.Message) bool {
	if msg == nil || msg.Type != protocol.MessageTypeSystemInfo {
		return false
	}
	return strings.Contains(msg.Content, "has joined the chat")
}

func humanJoinAnnouncementName(content string) string {
	content = strings.TrimSpace(content)
	const suffix = " has joined the chat"
	if !strings.HasSuffix(content, suffix) {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(strings.TrimSuffix(content, suffix)))
}

// shouldSkipHumanJoinAnnouncementLocked drops duplicate human join lines within a short window.
// Caller must hold h.mu (write lock).
func (h *Hub) shouldSkipHumanJoinAnnouncementLocked(channelName string, msg *protocol.Message) bool {
	if !isHumanJoinAnnouncement(msg) {
		return false
	}
	name := humanJoinAnnouncementName(msg.Content)
	if name == "" && strings.TrimSpace(msg.From.ID) == "" {
		return false
	}
	cutoff := time.Now().Add(-humanJoinDedupeWindow)
	if !msg.Timestamp.IsZero() {
		cutoff = msg.Timestamp.Add(-humanJoinDedupeWindow)
	}
	msgs := h.messages[channelName]
	for i := len(msgs) - 1; i >= 0 && i >= len(msgs)-40; i-- {
		m := msgs[i]
		if m == nil || !isHumanJoinAnnouncement(m) {
			continue
		}
		if !m.Timestamp.IsZero() && m.Timestamp.Before(cutoff) {
			break
		}
		if strings.TrimSpace(msg.From.ID) != "" && m.From.ID == msg.From.ID {
			return true
		}
		if name != "" && humanJoinAnnouncementName(m.Content) == name {
			return true
		}
	}
	return false
}
