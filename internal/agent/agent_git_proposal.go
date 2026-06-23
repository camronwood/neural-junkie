package agent

import (
	"context"
	"log"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

var gitChangeBlockRegex = regexp.MustCompile(`(?s)\[GIT_CHANGE\](.*?)\[/GIT_CHANGE\]`)

func (a *Agent) maybeSubmitGitChangeFromResponse(ctx context.Context, response, channel string, sourceMsg *protocol.Message) (string, bool, error) {
	if isAskModeReadOnly(sourceMsg) {
		return response, false, nil
	}
	match := gitChangeBlockRegex.FindStringSubmatch(response)
	if len(match) < 2 {
		return response, false, nil
	}
	block := match[1]
	op := strings.ToLower(strings.TrimSpace(fieldValue(block, "operation")))
	if op == "" {
		op = "commit"
	}
	msg := strings.TrimSpace(fieldValue(block, "message"))
	pathsRaw := strings.TrimSpace(fieldValue(block, "paths"))
	var paths []string
	for _, p := range strings.Split(pathsRaw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
		 paths = append(paths, p)
		}
	}
	proposal := map[string]interface{}{
		"id":           uuid.NewString(),
		"operation":   op,
		"message":     msg,
		"paths":       paths,
		"workspace_id": workspaceIDFromMessage(sourceMsg),
		"agent":       a.Info,
		"channel":     channel,
	}
	outMsg := protocol.NewMessage(protocol.MessageTypeChat, channel, a.Info, stripGitChangeBlocks(response))
	outMsg.Metadata = map[string]interface{}{"git_change_proposal": proposal}
	if a.Hub == nil {
		return stripGitChangeBlocks(response), false, nil
	}
	if err := a.Hub.SendMessage(outMsg); err != nil {
		return response, false, err
	}
	log.Printf("[%s] git_change_proposed(operation=%s)", a.Info.Name, op)
	return stripGitChangeBlocks(response), true, nil
}

func stripGitChangeBlocks(response string) string {
	return strings.TrimSpace(gitChangeBlockRegex.ReplaceAllString(response, ""))
}

func fieldValue(block, key string) string {
	re := regexp.MustCompile(`(?mi)^` + regexp.QuoteMeta(key) + `\s*:\s*(.+)$`)
	if m := re.FindStringSubmatch(block); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func workspaceIDFromMessage(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	if v, ok := msg.Metadata["workspace_id"].(string); ok {
		return strings.TrimSpace(v)
	}
	if ws, ok := msg.Metadata["workspace_context"].(map[string]interface{}); ok {
		if v, ok := ws["workspace_id"].(string); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
