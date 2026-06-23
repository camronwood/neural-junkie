package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func buildLearningPromptContext(msg *protocol.Message) learning.PromptContext {
	pctx := learning.PromptContext{}
	if msg == nil {
		return pctx
	}
	pctx.Query = strings.TrimSpace(msg.Content)
	pctx.Channel = msg.Channel
	for _, name := range UsernamesForRulesLookup(msg) {
		if slug := learning.SlugUserID(name); slug != "" {
			pctx.UserID = slug
			break
		}
	}
	pctx.WorkspaceID = workspaceIDFromMetadata(msg.Metadata)
	return pctx
}

func workspaceIDFromMetadata(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	if ws, ok := meta["workspace_id"].(string); ok {
		return strings.TrimSpace(ws)
	}
	wctx, ok := meta["workspace_context"].(map[string]interface{})
	if !ok || wctx == nil {
		return ""
	}
	ws, ok := wctx["workspace_id"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(ws)
}

// AppendLearningsForMessage injects user-confirmed learnings into the system portion of a prompt.
func AppendLearningsForMessage(system *strings.Builder, msg *protocol.Message, self *protocol.AgentInfo) learning.PromptResult {
	if system == nil || self == nil {
		return learning.PromptResult{}
	}
	pctx := buildLearningPromptContext(msg)
	pctx.AgentType = string(self.Type)
	pctx.AgentName = self.Name
	pr := learning.AppendForAgent(system, self, pctx)
	if pr.Count > 0 && msg != nil {
		if msg.Metadata == nil {
			msg.Metadata = map[string]any{}
		}
		msg.Metadata["injected_learnings_count"] = pr.Count
		if len(pr.IDs) > 0 {
			msg.Metadata["injected_learning_ids"] = pr.IDs
		}
	}
	return pr
}
