package hub

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func humanApproverFrom(userID string) protocol.AgentInfo {
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "default" {
		return protocol.AgentInfo{ID: "human-user", Name: "User", Type: "human"}
	}
	return protocol.AgentInfo{ID: userID, Name: userID, Type: "human"}
}

// NotifyFileChangeApproved broadcasts UI confirmation and an agent-visible approval
// message that is kept in LLM chat history.
func (h *Hub) NotifyFileChangeApproved(change *filechange.FileChange, approvedBy string) {
	if h == nil || change == nil {
		return
	}
	h.resolveDurableInput(change.ID, approvedBy, map[string]any{"status": "approved"})
	channel := strings.TrimSpace(change.Channel)
	if channel == "" {
		channel = "general"
	}
	displayPath := change.GetDisplayPath()
	h.UpdateChangeProposalStatus(
		channel,
		change.ID,
		protocol.ChangeProposalStatusApproved,
		"",
		"",
	)

	systemFrom := protocol.AgentInfo{
		ID:     "system",
		Name:   "System",
		Type:   protocol.AgentTypeGeneral,
		Status: "active",
	}
	confirm := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		systemFrom,
		fmt.Sprintf("Applied change `%s` to `%s`.", change.ID, displayPath),
	)
	if err := h.SendMessage(confirm); err != nil {
		log.Printf("[FileChange] Failed to send UI confirmation: %v", err)
	}

	userFrom := humanApproverFrom(approvedBy)
	content := fmt.Sprintf(
		"Approved and applied your %s change to `%s`. Continue with the implementation — do not ask me to approve again.",
		change.Operation,
		displayPath,
	)
	approvalMsg := protocol.NewMessage(protocol.MessageTypeChat, channel, userFrom, content)
	if approvalMsg.Metadata == nil {
		approvalMsg.Metadata = make(map[string]interface{})
	}
	approvalMsg.Metadata[protocol.MetaFileChangeApproved] = true
	approvalMsg.Metadata[protocol.MetaFileChangeID] = change.ID
	approvalMsg.Metadata[protocol.MetaFileChangePath] = displayPath
	if change.Agent.ID != "" {
		approvalMsg.Metadata[protocol.MetaFileChangeAgentID] = change.Agent.ID
		approvalMsg.Mention(change.Agent.ID)
	}
	if err := h.SendMessage(approvalMsg); err != nil {
		log.Printf("[FileChange] Failed to send agent-visible approval: %v", err)
	}

	h.scheduleImmediateSummaryRefresh(channel)

	rel := filepath.ToSlash(strings.TrimSpace(change.FilePath))
	if strings.HasPrefix(rel, "collabs/") && strings.HasSuffix(strings.ToLower(rel), ".md") {
		wsRoot := ""
		if h.fileChangeManager != nil && h.fileChangeManager.GetExecutor() != nil {
			wsRoot = h.fileChangeManager.GetExecutor().GetWorkspaceRoot()
		}
		memory.IndexCollabMarkdownRel(wsRoot, rel, channel)
	}
}

// NotifyFileChangeRejected updates the durable proposal card and records the
// decision in the visible transcript.
func (h *Hub) NotifyFileChangeRejected(change *filechange.FileChange) {
	if h == nil || change == nil {
		return
	}
	h.resolveDurableInput(change.ID, "user", map[string]any{
		"status": "rejected", "reason": change.Reason,
	})
	channel := strings.TrimSpace(change.Channel)
	if channel == "" {
		channel = "general"
	}
	h.UpdateChangeProposalStatus(
		channel,
		change.ID,
		protocol.ChangeProposalStatusRejected,
		change.Reason,
		"",
	)
	systemFrom := protocol.AgentInfo{
		ID:     "system",
		Name:   "System",
		Type:   protocol.AgentTypeGeneral,
		Status: "active",
	}
	content := fmt.Sprintf("Rejected change `%s` for `%s`.", change.ID, change.GetDisplayPath())
	if strings.TrimSpace(change.Reason) != "" && change.Reason != "No reason provided" {
		content += " Reason: " + strings.TrimSpace(change.Reason)
	}
	if err := h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		systemFrom,
		content,
	)); err != nil {
		log.Printf("[FileChange] Failed to send rejection confirmation: %v", err)
	}
}
