package hub

import (
	"fmt"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// GetChannelMessageByID returns a stored message with full metadata (not API-redacted).
func (h *Hub) GetChannelMessageByID(channelName, messageID string) (*protocol.Message, error) {
	if h == nil || channelName == "" || messageID == "" {
		return nil, fmt.Errorf("channel and message id required")
	}

	h.mu.RLock()
	msgs := h.messages[channelName]
	h.mu.RUnlock()

	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i] != nil && msgs[i].ID == messageID {
			return protocol.CloneMessage(msgs[i])
		}
	}

	if h.persistentStore != nil {
		batch, err := h.persistentStore.ListChannelMessages(channelName, 500, "")
		if err != nil {
			return nil, err
		}
		for i := len(batch) - 1; i >= 0; i-- {
			if batch[i] != nil && batch[i].ID == messageID {
				return protocol.CloneMessage(batch[i])
			}
		}
	}

	return nil, fmt.Errorf("message %s not found in channel %s", messageID, channelName)
}

// GeneratedImageBytesFromMessage returns image bytes and MIME from stored generated_image metadata.
func GeneratedImageBytesFromMessage(msg *protocol.Message) (mime string, data []byte, ok bool) {
	if msg == nil || msg.Metadata == nil {
		return "", nil, false
	}
	raw, ok := msg.Metadata["generated_image"].(map[string]interface{})
	if !ok {
		return "", nil, false
	}
	mime, _ = raw["mime"].(string)
	if mime == "" {
		mime = "image/png"
	}
	if path, _ := raw["path"].(string); path != "" {
		if b, m, err := readGeneratedImageFile(path); err == nil {
			if m != "" {
				mime = m
			}
			return mime, b, true
		}
	}
	if dataStr, _ := raw["data"].(string); dataStr != "" {
		if b, err := decodeGeneratedImageBase64(dataStr); err == nil {
			return mime, b, true
		}
	}
	if path, ok := generatedImageFileForMessageID(msg.ID); ok {
		if b, m, err := readGeneratedImageFile(path); err == nil {
			if m != "" {
				mime = m
			}
			return mime, b, true
		}
	}
	return "", nil, false
}
