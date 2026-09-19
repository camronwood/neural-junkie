package hub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// EmitEditApplyStream broadcasts progressive stream_delta chunks of resolved
// new_content for a registered create/edit proposal, then stream_end.
// Old clients ignore stream_kind=edit_apply and still receive the proposal card.
func (h *Hub) EmitEditApplyStream(
	channel string,
	from protocol.AgentInfo,
	changeID string,
	path string,
	newContent string,
) {
	if h == nil {
		return
	}
	channel = strings.TrimSpace(channel)
	changeID = strings.TrimSpace(changeID)
	path = strings.TrimSpace(path)
	if channel == "" || changeID == "" || path == "" || newContent == "" {
		return
	}

	chunks := protocol.ChunkEditApplyContent(newContent, protocol.EditApplyStreamChunkSize)
	if len(chunks) == 0 {
		return
	}

	streamID := uuid.New().String()
	offset := 0
	for _, chunk := range chunks {
		delta := protocol.NewMessage(protocol.MessageTypeStreamDelta, channel, from, chunk)
		delta.ID = streamID
		if delta.Metadata == nil {
			delta.Metadata = make(map[string]interface{})
		}
		protocol.ApplyEditApplyStreamMeta(delta.Metadata, protocol.EditApplyStreamMeta{
			ChangeID: changeID,
			Path:     path,
			Offset:   offset,
		})
		h.BroadcastDirect(channel, delta)
		offset += len(chunk)
	}

	end := protocol.NewMessage(protocol.MessageTypeStreamEnd, channel, from, "")
	end.ID = streamID
	if end.Metadata == nil {
		end.Metadata = make(map[string]interface{})
	}
	protocol.ApplyEditApplyStreamMeta(end.Metadata, protocol.EditApplyStreamMeta{
		ChangeID: changeID,
		Path:     path,
		Offset:   offset,
		Done:     true,
	})
	h.BroadcastDirect(channel, end)
}

// emitEditApplyStreamForChange streams resolved content for a create/edit change.
func (h *Hub) emitEditApplyStreamForChange(msg *protocol.Message, change *filechange.FileChange) {
	if h == nil || msg == nil || change == nil {
		return
	}
	switch change.Operation {
	case filechange.FileOperationCreate, filechange.FileOperationEdit:
	default:
		return
	}
	if strings.TrimSpace(change.NewContent) == "" {
		return
	}
	channel := strings.TrimSpace(msg.Channel)
	if channel == "" {
		channel = strings.TrimSpace(change.Channel)
	}
	from := msg.From
	if from.ID == "" && from.Name == "" {
		from = change.Agent
	}
	h.EmitEditApplyStream(channel, from, change.ID, change.FilePath, change.NewContent)
}
