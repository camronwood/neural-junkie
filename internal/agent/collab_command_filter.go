package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// filterCollabCommandSuggestions drops build/deploy commands during collab
// execution when the assigned task is a file deliverable (common doc/schema work).
func filterCollabCommandSuggestions(msg *protocol.Message, suggestions []protocol.CommandSuggestion) []protocol.CommandSuggestion {
	if msg == nil || len(suggestions) == 0 {
		return suggestions
	}
	if msg.Type != protocol.MessageTypeCollabTask {
		return suggestions
	}
	if strings.TrimSpace(msg.GetCollaborationPhase()) != "executing" {
		return suggestions
	}

	taskText := strings.TrimSpace(msg.Content)
	title, _ := msg.Metadata["task_title"].(string)
	desc, _ := msg.Metadata["task_description"].(string)
	if title == "" && desc == "" {
		title = taskText
		desc = taskText
	}
	policy := collaboration.NewDeliverablePolicy(
		collaboration.CollaborationTask{Title: title, Description: desc},
		"",
		nil,
	)
	if !policy.RequiresFile() {
		return suggestions
	}

	out := make([]protocol.CommandSuggestion, 0, len(suggestions))
	for _, s := range suggestions {
		cmd := strings.TrimSpace(s.Command)
		if cmd == "" {
			continue
		}
		if protocol.LooksLikeStackToolCommand(cmd) {
			continue
		}
		if policy.MarkdownOnly() && !protocol.LooksLikeReadOnlyInspectionCommand(cmd) {
			continue
		}
		out = append(out, s)
	}
	return out
}
