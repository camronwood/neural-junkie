package agent

import (
	"context"
	"fmt"
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
	if s.SeedsLoaded >= 2 {
		return true
	}
	if len(s.DiscoverTools) >= 1 {
		return true
	}
	if s.StackManifest != nil && s.StackManifest.HasEntryPoint() {
		return true
	}
	return false
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

func (a *Agent) validateProposalForSession(ctx context.Context, sourceMsg *protocol.Message, path string, op ProposalOperation) error {
	path = a.ResolveProposalPath(ctx, sourceMsg, path)
	wsPath := ""
	if sourceMsg != nil {
		wsPath = a.resolveWorkspacePath(sourceMsg)
	}

	st := implementationSessionStateFromContext(ctx)
	if st != nil && !st.groundingSatisfied() {
		return fmt.Errorf("grounding required: read the stack manifest and use read_file or glob_file_search before proposing edits")
	}

	manifest := a.manifestForProposal(ctx, sourceMsg)
	op = a.inferProposalOp(wsPath, path, op)

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
	if t := sourceMsg.EditorAgentTrust(); t != "" {
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
