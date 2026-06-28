package hub

import (
	"fmt"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// persistSlashCommandExchange stores the human slash line and optional system response.
func (h *Hub) persistSlashCommandExchange(commandMsg, response *protocol.Message) error {
	if commandMsg == nil {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[commandMsg.Channel]; !ok {
		return fmt.Errorf("channel %s not found", commandMsg.Channel)
	}

	if err := h.appendAndBroadcastMessageLocked(commandMsg); err != nil {
		return err
	}
	if response != nil {
		if err := h.appendAndBroadcastMessageLocked(response); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hub) appendAndBroadcastMessageLocked(msg *protocol.Message) error {
	if msg == nil {
		return nil
	}
	if msg.IsInThread() {
		threadID := msg.GetThreadID()
		if _, exists := h.threadParentAuthors[threadID]; !exists {
			for _, channelMsg := range h.messages[msg.Channel] {
				if channelMsg.ID == threadID {
					h.threadParentAuthors[threadID] = channelMsg.From.ID
					break
				}
			}
		}
		h.appendThreadMessageLocked(threadID, msg)
		h.updateThreadMetadata(threadID, msg)
		h.broadcastToThread(threadID, msg)
	}
	if h.shouldSkipHumanJoinAnnouncementLocked(msg.Channel, msg) {
		return nil
	}
	h.appendChannelMessageLocked(msg.Channel, msg)
	h.broadcast(msg.Channel, msg)
	return nil
}
