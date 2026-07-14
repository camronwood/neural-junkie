package main

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// readWorkspaceRelFile reads a workspace-relative file via WorkspaceBackend (local or remote).
func readWorkspaceRelFile(ctx context.Context, workspaceID, relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if workspaceID == "" || relPath == "" {
		return ""
	}
	data, code, _ := backendReadFile(ctx, workspaceID, strings.TrimPrefix(relPath, "/"))
	if code != 0 || data == nil {
		return ""
	}
	return string(data)
}

func latestPendingChangeID(pending []*filechange.FileChange, channel string) string {
	channel = strings.TrimSpace(channel)
	var best *filechange.FileChange
	for _, c := range pending {
		if c == nil {
			continue
		}
		if channel != "" && c.Channel != channel {
			continue
		}
		if best == nil || c.RequestedAt.After(best.RequestedAt) {
			best = c
		}
	}
	if best == nil {
		return ""
	}
	return best.ID
}

// latestPendingChangeIDForChannel returns the newest pending file-change id for channel.
func latestPendingChangeIDForChannel(channel string) string {
	if chatHub == nil {
		return ""
	}
	return latestPendingChangeID(chatHub.GetFileChangeManager().ListPendingFileChanges(""), channel)
}

// editorHistoryPrompt builds a prompt prefix from stored editor session turns.
func editorHistoryPrompt(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	msgs := editorSessions.HistoryMessages(sessionID)
	if len(msgs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nPrevious editor session turns:\n")
	for _, m := range msgs {
		if m == nil {
			continue
		}
		role := "assistant"
		if protocol.IsUserLikeSender(m.From) {
			role = "user"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(strings.TrimSpace(m.Content))
		b.WriteString("\n")
	}
	return b.String()
}
