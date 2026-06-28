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
	wsPath := workspacePathFromMetadata(msg)
	if wsPath == "" {
		wsPath = a.resolveWorkspacePath(msg)
	}
	if wsPath == "" {
		return prompt
	}
	subQ := shapeRepoConsultQuestion(msg)
	text, agentName, err := ch.ConsultRepoForPath(ctx, wsPath, subQ, msg.Channel)
	if err != nil || strings.TrimSpace(text) == "" {
		return prompt
	}
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\n=== REPO_CONSULT ===\n")
	b.WriteString(fmt.Sprintf("Source: %s (%s)\n", agentName, wsPath))
	b.WriteString("Merge the following indexed repository context into your answer. Do not invent facts beyond it.\n")
	b.WriteString(strings.TrimSpace(text))
	b.WriteString("\n=== END REPO_CONSULT ===\n")
	a.setLastRepoConsulted(agentName)
	return b.String()
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
	return false
}

func (a *Agent) shouldRunRepoConsult(ctx context.Context, msg *protocol.Message, intent TurnIntent) bool {
	if a.shouldSkipRepoConsult(msg) {
		return false
	}
	wsPath := workspacePathFromMetadata(msg)
	if wsPath == "" {
		wsPath = a.resolveWorkspacePath(msg)
	}
	if wsPath == "" {
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
		return userRequestsCodeReview(msg.Content) || strings.Contains(strings.ToLower(msg.Content), "architecture")
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
