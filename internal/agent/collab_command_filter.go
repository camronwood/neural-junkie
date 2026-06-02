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
	task := collaboration.CollaborationTask{Title: taskText, Description: taskText}
	if !collaboration.TaskRequiresFileDeliverable(task) {
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
		if collaboration.TaskLooksLikeMarkdownDeliverable(task) && !protocol.LooksLikeReadOnlyInspectionCommand(cmd) {
			continue
		}
		out = append(out, s)
	}
	return out
}
