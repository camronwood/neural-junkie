package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// appendRepoConsultContext merges indexed repo knowledge when a workspace path is open.
func (a *Agent) appendRepoConsultContext(ctx context.Context, msg *protocol.Message, prompt string, intent TurnIntent) string {
	if a == nil || msg == nil || a.Info.Type == protocol.AgentTypeRepo {
		return prompt
	}
	if !a.shouldRunRepoConsult(ctx, msg, intent) {
		return prompt
	}
	ch := a.Hub.GetCommandHandler()
	if ch == nil {
		return prompt
	}
	subQ := shapeRepoConsultQuestion(msg)
	refs := selectReposForConsult(msg, intent)
	if len(refs) == 0 {
		wsPath := workspacePathFromMetadata(msg)
		if wsPath == "" {
			wsPath = a.resolveWorkspacePath(msg)
		}
		if wsPath == "" {
			return prompt
		}
		refs = []WorkspaceRef{{Path: wsPath, Name: filepathBaseName(wsPath)}}
	}
	if multi, ok := ch.(CommandHandlerInterface); ok && len(refs) > 1 {
		blocks, err := multi.ConsultReposForPaths(ctx, refs, subQ, msg.Channel)
		if err == nil && len(blocks) > 0 {
			a.recordKnowledgeExecutedFor(msg.ID, "repo_consult")
			return appendRepoConsultBlocks(prompt, blocks, a, msg)
		}
	}
	text, agentName, err := ch.ConsultRepoForPath(ctx, refs[0].Path, subQ, msg.Channel)
	if err != nil || strings.TrimSpace(text) == "" {
		return prompt
	}
	a.recordKnowledgeExecutedFor(msg.ID, "repo_consult")
	label := refs[0].Name
	if label == "" {
		label = filepathBaseName(refs[0].Path)
	}
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\n=== REPO_CONSULT")
	if label != "" {
		b.WriteString(fmt.Sprintf(" (%s)", label))
	}
	b.WriteString(" ===\n")
	b.WriteString(fmt.Sprintf("Source: %s (%s)\n", agentName, refs[0].Path))
	b.WriteString("Merge the following indexed repository context into your answer. Do not invent facts beyond it.\n")
	b.WriteString(strings.TrimSpace(text))
	b.WriteString("\n=== END REPO_CONSULT ===\n")
	a.setLastRepoConsulted(agentName)
	out := b.String()
	return appendCrossRepoHints(out, msg)
}

func appendRepoConsultBlocks(prompt string, blocks []RepoConsultBlock, a *Agent, msg *protocol.Message) string {
	var b strings.Builder
	b.WriteString(prompt)
	for _, block := range blocks {
		label := filepathBaseName(block.Path)
		b.WriteString("\n\n=== REPO_CONSULT")
		if label != "" {
			b.WriteString(fmt.Sprintf(" (%s)", label))
		}
		b.WriteString(" ===\n")
		b.WriteString(fmt.Sprintf("Source: %s (%s)\n", block.AgentName, block.Path))
		b.WriteString("Merge the following indexed repository context into your answer. Do not invent facts beyond it.\n")
		b.WriteString(strings.TrimSpace(block.Text))
		b.WriteString("\n=== END REPO_CONSULT ===\n")
	}
	if len(blocks) > 0 {
		a.setLastRepoConsulted(blocks[0].AgentName)
	}
	return appendCrossRepoHints(b.String(), msg)
}

func filepathBaseName(path string) string {
	path = strings.TrimSpace(strings.TrimRight(path, "/"))
	if path == "" {
		return ""
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func (a *Agent) shouldSkipRepoConsult(msg *protocol.Message) bool {
	if msg == nil {
		return true
	}
	if msg.GetCollaborationID() != "" {
		return true
	}
	switch msg.Type {
	case protocol.MessageTypeCollabTask, protocol.MessageTypeCollabRecap:
		return true
	}
	if msg.From.Type != "human" && msg.From.Type != protocol.AgentTypeGeneral {
		return true
	}
	if len(msg.Content) > 0 && msg.Content[0] == '/' {
		return true
	}
	if msg.IdeEditorModeIsPlan() {
		return true
	}
	if a.constrainedIDETurn(msg) {
		return true
	}
	return false
}

func (a *Agent) shouldRunRepoConsult(ctx context.Context, msg *protocol.Message, intent TurnIntent) bool {
	if a.shouldSkipRepoConsult(msg) {
		return false
	}
	if !ShouldRunCodebaseSearch(a.effectiveKnowledgePlanFromMessage(msg)) {
		return false
	}
	wsPath := workspacePathFromMetadata(msg)
	if wsPath == "" {
		wsPath = a.resolveWorkspacePath(msg)
	}
	if wsPath == "" && len(linkedWorkspacesFromMetadata(msg)) == 0 {
		return false
	}
	if st := implementationSessionStateFromContext(ctx); st != nil {
		if st.BootFixIntent && strings.TrimSpace(st.LastCommandOutput()) != "" {
			return false
		}
		if !st.groundingSatisfied() {
			return true
		}
	}
	switch intent {
	case IntentTask, IntentSubstantive:
		return true
	default:
		return userRequestsCodeReviewForMessage(msg) || strings.Contains(strings.ToLower(msg.Content), "architecture")
	}
}

func shapeRepoConsultQuestion(msg *protocol.Message) string {
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return "Summarize relevant architecture, entry points, and file locations for this task."
	}
	lower := strings.ToLower(content)
	if strings.Contains(lower, "where") || strings.Contains(lower, "how") || strings.Contains(lower, "architecture") {
		return content
	}
	return "For this task: " + content + "\nFocus on architecture, entry points, and relevant file paths."
}

func (a *Agent) setLastRepoConsulted(name string) {
	a.delegationMu.Lock()
	a.lastRepoConsulted = name
	a.delegationMu.Unlock()
}

// TakeRepoConsulted returns the repo agent consulted on the last turn and clears it.
func (a *Agent) TakeRepoConsulted() string {
	a.delegationMu.Lock()
	defer a.delegationMu.Unlock()
	out := a.lastRepoConsulted
	a.lastRepoConsulted = ""
	return out
}
