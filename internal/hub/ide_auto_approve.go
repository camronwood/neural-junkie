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
	editorTrustYolo      = "yolo"
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
	if !hubChannelAllowsIDEAutoApprove(msg.Channel, msg) {
		return
	}
	if operation != filechange.FileOperationCreate &&
		operation != filechange.FileOperationEdit &&
		operation != filechange.FileOperationDelete {
		return
	}
	isCreate := operation == filechange.FileOperationCreate
	if !agent.ShouldAutoApproveFileChangeOp(change.FilePath, isCreate, wsRoot) {
		log.Printf("[IDE] Skipping auto-approve for path: %s", change.FilePath)
		return
	}
	routedProvider := msg.From.AIProvider
	if provider, _ := msg.Metadata[protocol.MetadataRoutingProviderID].(string); strings.TrimSpace(provider) != "" {
		routedProvider = provider
	}
	destructive, ratio := agent.IsDestructiveFileRewrite(change.OldContent, change.NewContent)
	gitDestructive, _ := msg.Metadata["git_baseline_destructive"].(bool)
	if gitDestructive && !destructive {
		if gitRatio, ok := msg.Metadata["git_baseline_rewrite_ratio"].(float64); ok {
			ratio = gitRatio
		}
	}
	if (destructive || gitDestructive) && !agent.CanAutoApproveDestructiveRewrite(routedProvider, msg.Metadata) {
		log.Printf("[IDE] Holding destructive rewrite for approval: %s (replacement=%.0f%% provider=%s)",
			change.FilePath, ratio*100, routedProvider)
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
