package agent

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/plans"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func buildStrictPlanRetryPrompt(msg *protocol.Message) string {
	var b strings.Builder
	b.WriteString("Reply with ONLY valid plan markdown. Do not add prose before the opening ---.\n")
	b.WriteString("Use exactly this shape:\n\n")
	b.WriteString("---\n")
	b.WriteString("name: Short title\n")
	b.WriteString("overview: One or two sentences.\n")
	b.WriteString("todos:\n")
	b.WriteString("  - id: step-one\n")
	b.WriteString("    content: First step with a file path\n")
	b.WriteString("    status: pending\n")
	b.WriteString("isProject: false\n")
	b.WriteString("---\n\n")
	b.WriteString("# Title\n\n")
	b.WriteString("## Out of scope\n\n")
	b.WriteString("- Follow-ups not listed in todos.\n")
	if msg != nil && strings.TrimSpace(msg.Content) != "" {
		b.WriteString("\nUser request:\n")
		b.WriteString(strings.TrimSpace(msg.Content))
		b.WriteString("\n")
	}
	return b.String()
}

func preparePlanMarkdown(response string) (string, bool) {
	response = ensurePlanModeStructure(response)
	if prepared, ok := plans.NormalizeMarkdown(response); ok && plans.Parseable(prepared) {
		return prepared, true
	}
	if plans.Parseable(response) {
		return response, true
	}
	prepared := plans.PrepareMarkdown(response)
	return prepared, plans.Parseable(prepared)
}

func (a *Agent) recoverMalformedPlanReply(ctx context.Context, msg *protocol.Message, eff ai.AIProvider, text string) (string, bool) {
	if msg == nil || eff == nil || !msg.IdeEditorModeIsPlan() {
		return "", false
	}
	if plans.Parseable(text) {
		return "", false
	}
	retry, err := eff.GenerateResponse(ctx, buildStrictPlanRetryPrompt(msg), nil)
	if err != nil || strings.TrimSpace(retry) == "" {
		return "", false
	}
	retry = ensurePlanModeStructure(retry)
	if prepared, ok := plans.NormalizeMarkdown(retry); ok && plans.Parseable(prepared) {
		log.Printf("[%s] Plan malformed reply; strict retry + normalize succeeded", a.Info.Name)
		return prepared, true
	}
	if plans.Parseable(retry) {
		log.Printf("[%s] Plan malformed reply; strict retry succeeded", a.Info.Name)
		return retry, true
	}
	return "", false
}

// finalizePlanResponse normalizes plan markdown and optionally retries once with a strict template.
func (a *Agent) finalizePlanResponse(ctx context.Context, msg *protocol.Message, eff ai.AIProvider, response string) (string, bool) {
	prepared, ok := preparePlanMarkdown(response)
	if ok {
		return prepared, true
	}
	if retry, recovered := a.recoverMalformedPlanReply(ctx, msg, eff, response); recovered {
		if prepared, ok := preparePlanMarkdown(retry); ok {
			return prepared, true
		}
	}
	return response, false
}

func stampPlanFormatInvalid(responseMsg *protocol.Message) {
	if responseMsg == nil {
		return
	}
	if responseMsg.Metadata == nil {
		responseMsg.Metadata = make(map[string]interface{})
	}
	responseMsg.Metadata[protocol.MetaPlanFormatInvalid] = true
}
