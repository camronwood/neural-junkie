package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// registerFileChangeBatchProposal registers a coordinated multi-file proposal.
func (h *Hub) registerFileChangeBatchProposal(msg *protocol.Message, batchRaw interface{}) error {
	if msg != nil && (msg.IdeEditorModeIsAsk() || msg.IdeEditorModeIsPlan() ||
		msg.IdeEditorMode() == "ask" || msg.IdeEditorMode() == "plan") {
		log.Printf("[FileChange] Rejected batch proposal in read-only editor mode (%s) from %s",
			msg.IdeEditorMode(), msg.From.Name)
		return fmt.Errorf("file changes not allowed in ask/plan mode")
	}
	if err := h.rejectFileChangeOnClosedCollab(msg); err != nil {
		log.Printf("[FileChange] Rejected batch on closed collaboration: %v", err)
		return err
	}

	batchBytes, err := json.Marshal(batchRaw)
	if err != nil {
		return fmt.Errorf("marshal batch proposal: %w", err)
	}
	var batch struct {
		Proposals []*protocol.FileChangeProposal `json:"proposals"`
		Paths     []string                       `json:"paths"`
	}
	if err := json.Unmarshal(batchBytes, &batch); err != nil {
		return fmt.Errorf("unmarshal batch proposal: %w", err)
	}
	if len(batch.Proposals) == 0 {
		return fmt.Errorf("batch proposal requires at least one member")
	}
	if len(batch.Proposals) > 25 {
		return fmt.Errorf("batch proposal supports at most 25 files")
	}

	wsRoot := h.resolveWorkspaceRoot(msg)
	if wsRoot == "" && len(batch.Proposals) > 0 && batch.Proposals[0] != nil && batch.Proposals[0].Metadata != nil {
		if p, ok := batch.Proposals[0].Metadata["target_workspace_path"].(string); ok && strings.TrimSpace(p) != "" {
			wsRoot = strings.TrimSpace(p)
		}
	}
	if wsRoot == "" {
		return fmt.Errorf("missing workspace context for file change batch")
	}

	var workspaceIO filechange.WorkspaceIO
	if h.fileChangeBackendFn != nil {
		workspaceIO = h.fileChangeBackendFn(wsRoot)
	}

	prepared := make([]*filechange.FileChange, 0, len(batch.Proposals))
	paths := make([]string, 0, len(batch.Proposals))
	pathStatus := make([]protocol.PathChangeStatus, 0, len(batch.Proposals))

	for i, proposal := range batch.Proposals {
		if proposal == nil {
			return fmt.Errorf("batch member %d is nil", i)
		}
		change, err := h.materializeProposalChange(msg, proposal, wsRoot, workspaceIO)
		if err != nil {
			return fmt.Errorf("batch member %d (%s): %w", i, proposal.FilePath, err)
		}
		prepared = append(prepared, change)
		paths = append(paths, change.FilePath)
		pathStatus = append(pathStatus, protocol.PathChangeStatus{
			Path:   change.FilePath,
			Status: protocol.ChangeProposalStatusPending,
		})
	}

	request, err := h.fileChangeManager.ProposeFileChangeRequest(prepared, msg.From, msg.Channel)
	if err != nil {
		return fmt.Errorf("register batch request: %w", err)
	}
	for _, change := range request.Changes {
		if err := h.fileChangeManager.BindExecutionContext(change.ID, wsRoot, workspaceIO); err != nil {
			return fmt.Errorf("bind batch member %s: %w", change.ID, err)
		}
		if change.Metadata == nil {
			change.Metadata = make(map[string]interface{})
		}
		change.Metadata["workspace_root"] = wsRoot
		change.Metadata["request_id"] = request.ID
		h.persistFileChange(change)
	}

	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata[protocol.MetaFileChangeRequestID] = request.ID
	msg.Metadata["registered_change_ids"] = func() []string {
		ids := make([]string, 0, len(request.Changes))
		for _, c := range request.Changes {
			ids = append(ids, c.ID)
		}
		return ids
	}()

	// Progressive resolved-edit stream per member before auto-approve / card.
	for _, change := range request.Changes {
		h.emitEditApplyStreamForChange(msg, change)
	}

	heldReason := h.maybeAutoApproveIDEFileChangeRequest(msg, request, wsRoot)
	status := protocol.ChangeProposalStatusPending
	reason := ""
	if heldReason != "" {
		reason = heldReason
		msg.Metadata[protocol.MetaFileChangeHeldForApproval] = true
		msg.Metadata["file_change_hold_reason"] = heldReason
		for i := range pathStatus {
			pathStatus[i].Status = protocol.ChangeProposalStatusPending
			pathStatus[i].Reason = heldReason
		}
	} else {
		refreshed, _ := h.fileChangeManager.GetFileChangeRequest(request.ID)
		if refreshed != nil && refreshed.Status == filechange.FileChangeStatusApproved {
			status = protocol.ChangeProposalStatusApproved
			msg.Metadata[protocol.MetaFileChangeAutoApproved] = true
			for i := range pathStatus {
				pathStatus[i].Status = protocol.ChangeProposalStatusApproved
			}
		}
	}

	displayPaths := paths
	if len(batch.Paths) > 0 {
		displayPaths = batch.Paths
	}
	msg.Metadata[protocol.MetaChangeProposal] = protocol.ChangeProposalCard{
		Version:     1,
		Kind:        protocol.ChangeProposalKindFile,
		ID:          request.ID,
		Status:      status,
		Operation:   "batch",
		Paths:       displayPaths,
		RequestID:   request.ID,
		PathStatus:  pathStatus,
		Message:     fmt.Sprintf("Batch edit of %d files", len(displayPaths)),
		RequestedAt: request.RequestedAt,
		ExpiresAt:   request.ExpiresAt,
		Reason:      reason,
	}
	log.Printf("[FileChange] Registered batch request %s with %d files from %s",
		request.ID, len(request.Changes), msg.From.Name)
	return nil
}

func (h *Hub) materializeProposalChange(
	msg *protocol.Message,
	proposal *protocol.FileChangeProposal,
	wsRoot string,
	workspaceIO filechange.WorkspaceIO,
) (*filechange.FileChange, error) {
	filePath, err := h.resolveWorkspacePath(proposal.FilePath, wsRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve file path %q: %w", proposal.FilePath, err)
	}

	var operation filechange.FileOperation
	switch proposal.Operation {
	case "create":
		operation = filechange.FileOperationCreate
	case "edit":
		operation = filechange.FileOperationEdit
	case "delete":
		operation = filechange.FileOperationDelete
	case "move":
		operation = filechange.FileOperationMove
	default:
		return nil, fmt.Errorf("unknown file change operation: %s", proposal.Operation)
	}

	oldPath := proposal.OldPath
	newPath := proposal.NewPath
	if operation == filechange.FileOperationMove {
		oldPath, err = h.resolveWorkspacePath(proposal.OldPath, wsRoot)
		if err != nil {
			return nil, err
		}
		newPath, err = h.resolveWorkspacePath(proposal.NewPath, wsRoot)
		if err != nil {
			return nil, err
		}
	}

	proposalExecutor := filechange.NewFileChangeExecutor(wsRoot)
	proposalExecutor.SetWorkspaceIO(workspaceIO)
	manifest := agent.DetectStackManifest(wsRoot)
	proposal.FilePath = agent.RedirectProposalPathForOp(proposal.FilePath, manifest, proposal.Operation)
	filePath, err = h.resolveWorkspacePath(proposal.FilePath, wsRoot)
	if err != nil {
		return nil, err
	}
	if operation == filechange.FileOperationCreate {
		if _, readErr := proposalExecutor.GetFileContent(filePath); readErr == nil {
			operation = filechange.FileOperationEdit
			proposal.Operation = "edit"
		}
	}
	propOp := agent.ProposalOpCreate
	if operation == filechange.FileOperationEdit {
		propOp = agent.ProposalOpEdit
	}
	if operation != filechange.FileOperationDelete && operation != filechange.FileOperationMove {
		if err := agent.ValidateProposal(wsRoot, proposal.FilePath, propOp, manifest); err != nil {
			return nil, err
		}
	}
	if looksLikePlaceholderDeliverableContent(proposal.NewContent) {
		return nil, fmt.Errorf("placeholder deliverable content for %q", proposal.FilePath)
	}
	oldContent := proposal.OldContent
	newContent := proposal.NewContent
	if operation == filechange.FileOperationEdit {
		current, readErr := proposalExecutor.GetFileContent(filePath)
		if readErr != nil {
			return nil, fmt.Errorf("read current edit target %q: %w", proposal.FilePath, readErr)
		}
		oldContent = current
		if filechange.SanitizeFileChangeContent(newContent) ==
			filechange.SanitizeFileChangeContent(oldContent) {
			return nil, fmt.Errorf("no-op edit rejected for %q: resulting content is unchanged", proposal.FilePath)
		}
	}

	meta := map[string]interface{}{}
	if proposal.Metadata != nil {
		for k, v := range proposal.Metadata {
			meta[k] = v
		}
	}
	return &filechange.FileChange{
		Operation:  operation,
		FilePath:   filePath,
		OldPath:    oldPath,
		NewPath:    newPath,
		OldContent: oldContent,
		NewContent: newContent,
		Agent:      msg.From,
		Channel:    msg.Channel,
		Metadata:   meta,
	}, nil
}

// maybeAutoApproveIDEFileChangeRequest auto-approves a whole batch, or holds all
// members when any member is a destructive rewrite under non-yolo trust.
// Returns a non-empty hold reason when the batch is held for manual approval.
func (h *Hub) maybeAutoApproveIDEFileChangeRequest(
	msg *protocol.Message,
	request *filechange.FileChangeRequest,
	wsRoot string,
) string {
	if h == nil || msg == nil || request == nil || h.fileChangeManager == nil {
		return ""
	}
	if msg.GetCollaborationID() != "" {
		return ""
	}
	trust := strings.TrimSpace(msg.EditorAgentTrust())
	if trust == "" && msg.Metadata != nil {
		if t, ok := msg.Metadata["editor_agent_trust"].(string); ok {
			trust = strings.TrimSpace(t)
		}
	}
	if trust == "" {
		trust = agent.EffectiveEditorTrustForAutoApprove(msg)
	}
	if trust != editorTrustAutoApply && trust != editorTrustYolo {
		return ""
	}
	if msg.IdeEditorMode() == "ask" {
		return ""
	}
	if !hubChannelAllowsIDEAutoApprove(msg.Channel, msg) {
		return ""
	}

	routedProvider := msg.From.AIProvider
	if provider, _ := msg.Metadata[protocol.MetadataRoutingProviderID].(string); strings.TrimSpace(provider) != "" {
		routedProvider = provider
	}

	for _, change := range request.Changes {
		if change == nil {
			continue
		}
		if change.Operation != filechange.FileOperationCreate &&
			change.Operation != filechange.FileOperationEdit &&
			change.Operation != filechange.FileOperationDelete {
			return ""
		}
		isCreate := change.Operation == filechange.FileOperationCreate
		if !agent.ShouldAutoApproveFileChangeOp(change.FilePath, isCreate, wsRoot) {
			displayPath := agent.RelativizeFileChangePath(change.FilePath, wsRoot)
			if displayPath == "" {
				displayPath = change.FilePath
			}
			reason := fmt.Sprintf("Held for approval: path policy blocked auto-apply for %s", displayPath)
			log.Printf("[IDE] Holding batch %s: %s", request.ID, reason)
			for _, c := range request.Changes {
				if c == nil {
					continue
				}
				if c.Reason == "" {
					c.Reason = reason
				}
			}
			agent.RecordEditOutcome("auto_approve_batch", "path_policy", "", "pending_approval")
			return reason
		}
		destructive, ratio := agent.IsDestructiveFileRewrite(change.OldContent, change.NewContent)
		gitDestructive, _ := change.Metadata["git_baseline_destructive"].(bool)
		if gitDestructive && !destructive {
			if gitRatio, ok := change.Metadata["git_baseline_rewrite_ratio"].(float64); ok {
				ratio = gitRatio
			}
		}
		if (destructive || gitDestructive) && !agent.CanAutoApproveDestructiveRewrite(routedProvider, destructiveApproveMetadata(msg)) {
			reason := fmt.Sprintf("Held for approval: large rewrite on %s (%.0f%% replacement)",
				change.FilePath, ratio*100)
			log.Printf("[IDE] Holding batch %s: %s", request.ID, reason)
			for _, c := range request.Changes {
				if c == nil {
					continue
				}
				if c.Reason == "" {
					c.Reason = reason
				}
			}
			agent.RecordEditOutcome("auto_approve_batch", "destructive", "", "pending_approval")
			return reason
		}
	}

	approvedBy := "system"
	if msg.From.ID != "" {
		approvedBy = msg.From.ID
	}
	approved, err := h.fileChangeManager.ApproveFileChangeRequest(request.ID, approvedBy)
	if err != nil {
		log.Printf("[IDE] Auto-approve batch %s: %v", request.ID, err)
		if approved != nil {
			h.NotifyFileChangeRequestFailed(approved, err.Error())
		}
		return ""
	}
	h.NotifyFileChangeRequestApproved(approved, approvedBy)
	agent.RecordEditOutcome("auto_approve_batch", "ok", "", "auto_approved")
	log.Printf("[IDE] Auto-approved batch %s (%d files) trust=%s", request.ID, len(approved.Changes), trust)
	return ""
}

// NotifyFileChangeRequestApproved broadcasts batch approval to the UI and agents.
func (h *Hub) NotifyFileChangeRequestApproved(request *filechange.FileChangeRequest, approvedBy string) {
	if h == nil || request == nil {
		return
	}
	channel := strings.TrimSpace(request.Channel)
	if channel == "" {
		channel = "general"
	}
	h.UpdateChangeProposalStatusWithPaths(
		channel,
		request.ID,
		protocol.ChangeProposalStatusApproved,
		"",
		"",
		pathStatusFromRequest(request),
	)
	for _, change := range request.Changes {
		if change == nil {
			continue
		}
		h.resolveDurableInput(change.ID, approvedBy, map[string]any{"status": "approved"})
	}
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
		fmt.Sprintf("Applied batch change `%s` (%d files).", request.ID, len(request.Changes)),
	)
	_ = h.SendMessage(confirm)

	userFrom := humanApproverFrom(approvedBy)
	content := fmt.Sprintf(
		"Approved and applied your batch edit (`%s`, %d files). Continue with the implementation — do not ask me to approve again.",
		request.ID,
		len(request.Changes),
	)
	approvalMsg := protocol.NewMessage(protocol.MessageTypeChat, channel, userFrom, content)
	if approvalMsg.Metadata == nil {
		approvalMsg.Metadata = make(map[string]interface{})
	}
	approvalMsg.Metadata[protocol.MetaFileChangeApproved] = true
	approvalMsg.Metadata[protocol.MetaFileChangeRequestID] = request.ID
	if request.Agent.ID != "" {
		approvalMsg.Metadata[protocol.MetaFileChangeAgentID] = request.Agent.ID
		approvalMsg.Mention(request.Agent.ID)
	}
	_ = h.SendMessage(approvalMsg)
	h.scheduleImmediateSummaryRefresh(channel)
}

// NotifyFileChangeRequestRejected updates the batch card and transcript.
func (h *Hub) NotifyFileChangeRequestRejected(request *filechange.FileChangeRequest) {
	if h == nil || request == nil {
		return
	}
	channel := strings.TrimSpace(request.Channel)
	if channel == "" {
		channel = "general"
	}
	reason := ""
	for _, change := range request.Changes {
		if change != nil && strings.TrimSpace(change.Reason) != "" {
			reason = change.Reason
			break
		}
	}
	h.UpdateChangeProposalStatusWithPaths(
		channel,
		request.ID,
		protocol.ChangeProposalStatusRejected,
		reason,
		"",
		pathStatusFromRequest(request),
	)
	for _, change := range request.Changes {
		if change == nil {
			continue
		}
		h.resolveDurableInput(change.ID, "user", map[string]any{
			"status": "rejected", "reason": change.Reason,
		})
	}
	systemFrom := protocol.AgentInfo{
		ID:     "system",
		Name:   "System",
		Type:   protocol.AgentTypeGeneral,
		Status: "active",
	}
	content := fmt.Sprintf("Rejected batch change `%s` (%d files).", request.ID, len(request.Changes))
	if strings.TrimSpace(reason) != "" && reason != "No reason provided" {
		content += " Reason: " + strings.TrimSpace(reason)
	}
	_ = h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		systemFrom,
		content,
	))
}

// NotifyFileChangeRequestFailed updates the batch card after a mid-batch apply
// failure (including members that were applied then rolled back).
func (h *Hub) NotifyFileChangeRequestFailed(request *filechange.FileChangeRequest, errText string) {
	if h == nil || request == nil {
		return
	}
	channel := strings.TrimSpace(request.Channel)
	if channel == "" {
		channel = "general"
	}
	reason := strings.TrimSpace(errText)
	for _, change := range request.Changes {
		if change != nil && change.Status == filechange.FileChangeStatusRolledBack {
			if reason == "" {
				reason = change.Reason
			}
			break
		}
	}
	pathStatus := pathStatusFromRequest(request)
	h.UpdateChangeProposalStatusWithPaths(
		channel,
		request.ID,
		protocol.ChangeProposalStatusFailed,
		reason,
		errText,
		pathStatus,
	)
	for _, change := range request.Changes {
		if change == nil {
			continue
		}
		h.resolveDurableInput(change.ID, "system", map[string]any{
			"status": string(change.Status),
			"reason": change.Reason,
		})
	}
	systemFrom := protocol.AgentInfo{
		ID:     "system",
		Name:   "System",
		Type:   protocol.AgentTypeGeneral,
		Status: "active",
	}
	rolledBack := 0
	for _, change := range request.Changes {
		if change != nil && change.Status == filechange.FileChangeStatusRolledBack {
			rolledBack++
		}
	}
	content := fmt.Sprintf("Batch change `%s` failed (%d files).", request.ID, len(request.Changes))
	if rolledBack > 0 {
		content += fmt.Sprintf(" Rolled back %d applied file(s).", rolledBack)
	}
	if strings.TrimSpace(errText) != "" {
		content += " Error: " + strings.TrimSpace(errText)
	}
	_ = h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		systemFrom,
		content,
	))
}

func pathStatusFromRequest(request *filechange.FileChangeRequest) []protocol.PathChangeStatus {
	if request == nil {
		return nil
	}
	out := make([]protocol.PathChangeStatus, 0, len(request.Changes))
	for _, change := range request.Changes {
		if change == nil {
			continue
		}
		path := change.GetDisplayPath()
		if path == "" {
			path = change.FilePath
		}
		out = append(out, protocol.PathChangeStatus{
			Path:   path,
			Status: protocol.ChangeProposalStatus(change.Status),
			Reason: change.Reason,
		})
	}
	return out
}
