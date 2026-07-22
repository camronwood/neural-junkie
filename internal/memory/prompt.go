package memory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	sectionStart = "=== RELEVANT PAST CONTEXT ==="
	sectionEnd   = "=== END RELEVANT PAST CONTEXT ==="
	sectionHint  = "Retrieved from earlier in this channel/collaboration. Use only if relevant to the latest user message; do not repeat tail history."
)

// SelectForPrompt retrieves and formats chunks for prompt injection.
func SelectForPrompt(ctx context.Context, pctx PromptContext) (entries []SearchResult, ids []string) {
	if !memoryEnabled() {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	results, err := Search(cctx, pctx, DefaultTopK)
	if err != nil || len(results) == 0 {
		return nil, nil
	}
	for _, r := range results {
		ids = append(ids, r.Chunk.ID)
	}
	return results, ids
}

// AppendForPrompt writes retrieved memory into a system prompt builder.
func AppendForPrompt(system *strings.Builder, pctx PromptContext) PromptResult {
	var res PromptResult
	if system == nil || !memoryEnabled() {
		return res
	}
	query := strings.TrimSpace(pctx.Query)
	if query == "" {
		return res
	}
	entries, _ := SelectForPrompt(context.Background(), pctx)
	if len(entries) == 0 {
		return res
	}
	system.WriteString("\n" + sectionStart + "\n")
	system.WriteString(sectionHint + "\n\n")
	budget := DefaultPromptBudget
	for _, e := range entries {
		line := formatChunkLine(e.Chunk)
		if len(line) > budget {
			line = truncateEntryToBudget(line, budget)
		}
		if line == "" {
			continue
		}
		system.WriteString(line)
		budget -= len(line)
		res.Count++
		res.IDs = append(res.IDs, e.Chunk.ID)
	}
	system.WriteString("\n" + sectionEnd + "\n\n")
	return res
}

func truncateEntryToBudget(line string, budget int) string {
	const suffix = "…\n"
	if budget <= len(suffix)+16 {
		return ""
	}
	maxContent := budget - len(suffix)
	if maxContent >= len(line) {
		return line
	}
	for maxContent > 0 && maxContent < len(line) && line[maxContent]&0xc0 == 0x80 {
		maxContent--
	}
	if maxContent <= 16 {
		return ""
	}
	return strings.TrimSpace(line[:maxContent]) + suffix
}

func formatChunkLine(ch Chunk) string {
	prefix := ""
	switch ch.SourceType {
	case SourceCollabArtifact:
		if ch.RelPath != "" {
			prefix = fmt.Sprintf("[%s] ", ch.RelPath)
		} else {
			prefix = "[collab artifact] "
		}
	default:
		if ch.SenderName != "" {
			prefix = fmt.Sprintf("[%s] ", ch.SenderName)
		}
	}
	content := strings.TrimSpace(ch.Content)
	if len(content) > 400 {
		content = content[:400] + "…"
	}
	return prefix + content + "\n"
}
