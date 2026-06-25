package hub

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	editorTrustAutoApply = "auto_apply_edits"
	editorTrustYolo       = "yolo"
)

func (h *Hub) maybeAutoApproveIDEFileChange(msg *protocol.Message, change *filechange.FileChange, operation filechange.FileOperation, wsRoot string) {
	if h == nil || msg == nil || change == nil || h.fileChangeManager == nil {
		return
	}
	if msg.GetCollaborationID() != "" {
		return
	}
	trust := strings.TrimSpace(msg.EditorAgentTrust())
	if trust == "" {
		if msg.Metadata != nil {
			if t, ok := msg.Metadata["editor_agent_trust"].(string); ok {
				trust = strings.TrimSpace(t)
			}
		}
	}
	if trust == "" {
		trust = agent.EffectiveEditorTrustForAutoApprove(msg)
	}
	if trust != editorTrustAutoApply && trust != editorTrustYolo {
		return
	}
	if msg.IdeEditorMode() == "ask" {
		return
	}
	if operation != filechange.FileOperationCreate &&
		operation != filechange.FileOperationEdit &&
		operation != filechange.FileOperationDelete {
		return
	}
	approvedBy := "system"
	if msg.From.ID != "" {
		approvedBy = msg.From.ID
	}
	approved, err := h.fileChangeManager.ApproveFileChange(change.ID, approvedBy)
	if err != nil {
		log.Printf("[IDE] Auto-approve file change %s: %v", change.ID, err)
		return
	}
	h.NotifyFileChangeApproved(approved, approvedBy)
	log.Printf("[IDE] Auto-approved file change %s (%s) trust=%s", change.ID, change.FilePath, trust)
}
