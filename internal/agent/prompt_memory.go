package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func buildMemoryPromptContext(msg *protocol.Message, history []*protocol.Message, plan routing.KnowledgePlan) memory.PromptContext {
	pctx := memory.PromptContext{}
	if msg == nil {
		return pctx
	}
	pctx.Query = strings.TrimSpace(msg.Content)
	pctx.Channel = msg.Channel
	pctx.CollaborationID = memory.ResolveCollabID(msg.Channel)
	pctx.SourceTypes = MemorySourceFilter(plan)
	for _, m := range history {
		if m != nil && m.ID != "" {
			pctx.ExcludeMessageIDs = append(pctx.ExcludeMessageIDs, m.ID)
		}
	}
	if msg.ID != "" {
		pctx.ExcludeMessageIDs = append(pctx.ExcludeMessageIDs, msg.ID)
	}
	return pctx
}

// AppendMemoryForMessage injects retrieved past context into the system portion of a prompt.
func AppendMemoryForMessage(system *strings.Builder, msg *protocol.Message, history []*protocol.Message, plan routing.KnowledgePlan) memory.PromptResult {
	if system == nil || !ShouldInjectMemory(plan) {
		return memory.PromptResult{}
	}
	pctx := buildMemoryPromptContext(msg, history, plan)
	pr := memory.AppendForPrompt(system, pctx)
	if pr.Count > 0 && msg != nil {
		if msg.Metadata == nil {
			msg.Metadata = map[string]any{}
		}
		msg.Metadata["injected_memory_count"] = pr.Count
		if len(pr.IDs) > 0 {
			msg.Metadata["injected_memory_ids"] = pr.IDs
		}
	}
	return pr
}

func (a *Agent) appendMemoryForMessage(system *strings.Builder, msg *protocol.Message, history []*protocol.Message) memory.PromptResult {
	plan := a.effectiveKnowledgePlan(msg)
	pr := AppendMemoryForMessage(system, msg, history, plan)
	if pr.Count > 0 {
		a.recordKnowledgeExecuted(knowledgeExecutedPathForMemory(plan))
	}
	return pr
}
