package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func isDiscoverTool(name string) bool {
	switch strings.TrimSpace(name) {
	case "read_file", "glob_file_search", "list_dir", "grep", "semantic_search":
		return true
	default:
		return false
	}
}

func (s *ImplementationSessionState) groundingSatisfied() bool {
	if s == nil {
		return true
	}
	if s.SeedsLoaded >= 1 {
		return true
	}
	if len(s.LastReadPaths) >= 1 {
		return true
	}
	if s.BootFixIntent {
		if s.BootFixReadsSatisfied() {
			return true
		}
		return false
	}
	return false
}

func (s *ImplementationSessionState) targetGrounded(path string) bool {
	if s == nil {
		return true
	}
	path = normalizeFileChangeRelPath(path)
	for _, readPath := range s.LastReadPaths {
		if normalizeFileChangeRelPath(readPath) == path {
			return true
		}
	}
	// Seed files are injected verbatim into the implementation prompt. The seed
	// collector chooses requested/open/manifest target files, so they count as reads.
	return s.SeedsLoaded > 0
}

func (s *ImplementationSessionState) recordDiscoverTool(name string) {
	if s == nil || !isDiscoverTool(name) {
		return
	}
	name = strings.TrimSpace(name)
	for _, existing := range s.DiscoverTools {
		if existing == name {
			return
		}
	}
	s.DiscoverTools = append(s.DiscoverTools, name)
}

func (a *Agent) manifestForProposal(ctx context.Context, sourceMsg *protocol.Message) *StackManifest {
	st := implementationSessionStateFromContext(ctx)
	if st != nil && st.StackManifest != nil {
		return st.StackManifest
	}
	if sourceMsg == nil {
		return nil
	}
	wsPath := a.resolveWorkspacePath(sourceMsg)
	if wsPath == "" {
		return nil
	}
	return DetectStackManifest(wsPath)
}

// ResolveProposalPath normalizes and redirects paths using the stack manifest.
func (a *Agent) ResolveProposalPath(ctx context.Context, sourceMsg *protocol.Message, path string) string {
	manifest := a.manifestForProposal(ctx, sourceMsg)
	return RedirectProposalPath(path, manifest)
}

func protectedWorkspaceFilesFromMessage(msg *protocol.Message) []string {
	if msg == nil || msg.Metadata == nil {
		return nil
	}
	raw, ok := msg.Metadata["workspace_context"].(map[string]interface{})
	if !ok {
		return nil
	}
	files, ok := raw["unchanged_files"].([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, item := range files {
		if s, ok := item.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func isProtectedWorkspaceFile(msg *protocol.Message, path string) bool {
	if msg == nil {
		return false
	}
	rel := normalizeFileChangeRelPath(path)
	for _, p := range protectedWorkspaceFilesFromMessage(msg) {
		if normalizeFileChangeRelPath(p) == rel {
			return true
		}
	}
	lower := strings.ToLower(msg.Content)
	base := strings.ToLower(filepath.Base(rel))
	if base == "" {
		return false
	}
	for _, phrase := range []string{
		"do not modify " + base,
		"do not change " + base,
		"don't modify " + base,
		"don't change " + base,
		"do not edit " + base,
		"don't edit " + base,
		"never modify " + base,
		"never change " + base,
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}


func greenfieldCreatePath(wsPath, path string) bool {
	if wsPath == "" {
		return false
	}
	path = normalizeFileChangeRelPath(path)
	if path == "" {
		return false
	}
	abs := filepath.Join(wsPath, filepath.FromSlash(path))
	_, err := os.Stat(abs)
	return os.IsNotExist(err)
}

func (a *Agent) validateProposalForSession(ctx context.Context, sourceMsg *protocol.Message, path string, op ProposalOperation) error {
	path = a.ResolveProposalPath(ctx, sourceMsg, path)
	if isProtectedWorkspaceFile(sourceMsg, path) {
		return fmt.Errorf("protected file: edits to %s are not allowed for this request", path)
	}
	wsPath := ""
	if sourceMsg != nil {
		wsPath = a.resolveWorkspacePath(sourceMsg)
	}

	if op == ProposalOpCreate && sourceMsg != nil {
		want := normalizeFileChangeRelPath(path)
		for _, p := range DetectFilePaths(sourceMsg.Content) {
			if normalizeFileChangeRelPath(p) == want {
				op = a.inferProposalOp(wsPath, path, op)
				return ValidateProposal(wsPath, path, op, a.manifestForProposal(ctx, sourceMsg))
			}
		}
	}

	st := implementationSessionStateFromContext(ctx)
	if st != nil && !st.groundingSatisfied() {
		if sourceMsg != nil && collaborationRestrictsDiscoveryTools(sourceMsg) &&
			len(collaborationFocusAllowedReadPaths(sourceMsg)) > 0 {
			// Hub already merged focus sources into open_files; discovery tools are unavailable.
		} else if sourceMsg != nil && a != nil && userAffirmsPendingImplementation(sourceMsg.Content) &&
			(channelHasRecentImplementationAsk(a.channelHistory(sourceMsg.Channel), sourceMsg.ID) ||
				channelHasRecentImplementationActivity(a.channelHistory(sourceMsg.Channel), sourceMsg.ID, a.Info.ID)) {
			// continuation turn: prior user ask already grounded the session
		} else {
			return fmt.Errorf("grounding required: read the stack manifest and exact target file before proposing edits")
		}
	}
	manifest := a.manifestForProposal(ctx, sourceMsg)
	op = a.inferProposalOp(wsPath, path, op)

	if st != nil && !st.targetGrounded(path) {
		if op != ProposalOpCreate || !greenfieldCreatePath(wsPath, path) {
			return fmt.Errorf("grounding required: read target file %s before proposing edits", path)
		}
	}

	if err := ValidateProposal(wsPath, path, op, manifest); err != nil {
		if st != nil {
			st.PreflightErrors = appendUnique(st.PreflightErrors, []string{err.Error()})
		}
		return err
	}
	return nil
}

func (a *Agent) inferProposalOp(wsPath, path string, op ProposalOperation) ProposalOperation {
	if op != ProposalOpEdit {
		return op
	}
	if InferProposalOperation(wsPath, path) == ProposalOpCreate {
		return ProposalOpCreate
	}
	return op
}

// attachIdeSessionMetadataToProposal copies IDE session fields from the triggering user message
// so hub auto-approve (maybeAutoApproveIDEFileChange) sees editor_agent_trust on the proposal message.
func attachIdeSessionMetadataToProposal(msg *protocol.Message, sourceMsg *protocol.Message) {
	if msg == nil || sourceMsg == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	if t := resolveImplementationTrustMode(sourceMsg); t != "" {
		msg.Metadata["editor_agent_trust"] = t
	} else if t := sourceMsg.EditorAgentTrust(); t != "" {
		msg.Metadata["editor_agent_trust"] = t
	}
	if m := sourceMsg.IdeEditorMode(); m != "" {
		msg.Metadata["editor_mode"] = m
	}
	if sourceMsg.ImplementationSession() {
		msg.Metadata[protocol.IdeMetaImplementationSession] = true
	}
}

func formatPreflightRepairNote(errors []string, manifest *StackManifest) string {
	if len(errors) == 0 {
		note := "Proposal preflight failed. Use paths that match the stack manifest and existing files."
		if manifest != nil {
			note += manifest.FormatRepairHints()
		}
		note += "\n\nFix paths and emit corrected [FILE_CHANGE] blocks or call propose_file_edit with valid paths."
		return note
	}
	var b strings.Builder
	b.WriteString("Proposal preflight failed:\n")
	seen := make(map[string]bool, len(errors))
	for _, e := range errors {
		if seen[e] {
			continue
		}
		seen[e] = true
		b.WriteString("- ")
		b.WriteString(e)
		b.WriteString("\n")
	}
	if manifest != nil {
		b.WriteString(manifest.FormatRepairHints())
	}
	b.WriteString("\nFix paths and emit corrected [FILE_CHANGE] blocks or call propose_file_edit with valid paths.")
	b.WriteString(" Do NOT paste JSON tool calls in chat — use propose_file_edit or [FILE_CHANGE].")
	return b.String()
}
