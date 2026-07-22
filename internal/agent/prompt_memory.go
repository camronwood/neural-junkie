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
	pctx.ThreadID = msg.GetThreadID()
	pctx.GoalID = firstStringMetadata(msg.Metadata, "original_goal_id", "goal_id")
	pctx.IsCorrection = protocol.IsUserLikeSender(msg.From) && userCorrectionRE.MatchString(msg.Content)
	pctx.CollaborationID = memory.ResolveCollabID(msg.Channel)
	pctx.SourceTypes = MemorySourceFilter(plan)
	for _, m := range history {
		if m != nil && m.ID != "" {
			pctx.ExcludeMessageIDs = append(pctx.ExcludeMessageIDs, m.ID)
			pctx.SupersededMessageIDs = append(pctx.SupersededMessageIDs, metadataStringSlice(m.Metadata, "supersedes_message_ids")...)
		}
	}
	pctx.SupersededMessageIDs = append(pctx.SupersededMessageIDs, metadataStringSlice(msg.Metadata, "supersedes_message_ids")...)
	if pctx.IsCorrection {
		if priorID := relevantPriorUserInstructionID(history, msg); priorID != "" {
			pctx.SupersededMessageIDs = append(pctx.SupersededMessageIDs, priorID)
		}
	}
	if msg.ID != "" {
		pctx.ExcludeMessageIDs = append(pctx.ExcludeMessageIDs, msg.ID)
	}
	return pctx
}

func metadataStringSlice(metadata map[string]interface{}, key string) []string {
	if metadata == nil {
		return nil
	}
	switch values := metadata[key].(type) {
	case []string:
		return append([]string(nil), values...)
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return out
	default:
		return nil
	}
}

// AppendMemoryForMessage injects retrieved past context into the system portion of a prompt.
func AppendMemoryForMessage(system *strings.Builder, msg *protocol.Message, history []*protocol.Message, plan routing.KnowledgePlan) memory.PromptResult {
	if system == nil || !ShouldInjectMemory(plan) {
		return memory.PromptResult{}
	}
	pctx := buildMemoryPromptContext(msg, history, plan)
	return appendMemoryPromptContext(system, msg, pctx)
}

func appendMemoryPromptContext(system *strings.Builder, msg *protocol.Message, pctx memory.PromptContext) memory.PromptResult {
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
	plan := a.effectiveKnowledgePlanFromMessage(msg)
	if system == nil || !ShouldInjectMemory(plan) {
		return memory.PromptResult{}
	}
	pctx := buildMemoryPromptContext(msg, history, plan)
	if a != nil && a.Hub != nil && msg != nil {
		if provider, ok := a.Hub.(turnConversationContextProvider); ok {
			envelope := provider.GetTurnConversationContext(msg.Channel)
			if envelope.Goal != nil && strings.TrimSpace(envelope.Goal.ID) != "" {
				pctx.GoalID = strings.TrimSpace(envelope.Goal.ID)
			}
			pctx.SupersededMessageIDs = append(
				pctx.SupersededMessageIDs,
				envelope.SupersededMessageIDs...,
			)
		}
	}
	pr := appendMemoryPromptContext(system, msg, pctx)
	if pr.Count > 0 {
		a.recordKnowledgeExecutedFor(msg.ID, knowledgeExecutedPathForMemory(plan))
	}
	return pr
}
