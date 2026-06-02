package agent

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// tryWorkspaceVisibilityResponse answers "can you see my workspace?" from message metadata (no LLM).
func (a *Agent) tryWorkspaceVisibilityResponse(msg *protocol.Message) (string, bool) {
	if msg == nil || !userAsksAboutWorkspaceVisibility(msg.Content) {
		return "", false
	}
	scope := ResolveContextScope(msg)
	if scope == ContextScopeNone {
		return "I do **not** have workspace context on this message. Turn on workspace sharing (**Auto** or **Always** in the composer) and send again.", true
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return "Workspace sharing looks enabled, but no `workspace_context` arrived on this message — try sending again.", true
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return "", false
	}
	return formatWorkspaceVisibilityReply(ctxMap, scope), true
}

func formatWorkspaceVisibilityReply(ctxMap map[string]interface{}, scope string) string {
	var b strings.Builder
	b.WriteString("Yes — I have workspace context on this message.\n\n")
	b.WriteString(fmt.Sprintf("- **Context scope:** `%s`\n", scope))

	name, _ := ctxMap["workspace_name"].(string)
	path, _ := ctxMap["workspace_path"].(string)
	if strings.TrimSpace(name) != "" {
		b.WriteString(fmt.Sprintf("- **Project:** %s\n", strings.TrimSpace(name)))
	}
	if strings.TrimSpace(path) != "" {
		b.WriteString(fmt.Sprintf("- **Path:** `%s`\n", strings.TrimSpace(path)))
	}

	if tree, ok := ctxMap["file_tree"].(string); ok && strings.TrimSpace(tree) != "" {
		lines := strings.Split(strings.TrimSpace(tree), "\n")
		b.WriteString(fmt.Sprintf("- **File tree:** %d lines", len(lines)))
		if len(lines) > 0 {
			b.WriteString(" (first entries below)\n```\n")
			max := len(lines)
			if max > 24 {
				max = 24
			}
			b.WriteString(strings.Join(lines[:max], "\n"))
			if len(lines) > max {
				b.WriteString(fmt.Sprintf("\n… (%d more lines)", len(lines)-max))
			}
			b.WriteString("\n```\n")
		} else {
			b.WriteString("\n")
		}
	}

	files, _ := ctxMap["open_files"].([]interface{})
	if len(files) > 0 {
		b.WriteString(fmt.Sprintf("- **Open files:** %d\n", len(files)))
		for _, f := range files {
			fm, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			fp, _ := fm["path"].(string)
			lang, _ := fm["language"].(string)
			active, _ := fm["is_active"].(bool)
			content, _ := fm["content"].(string)
			line := fmt.Sprintf("  - `%s` (%s)", fp, lang)
			if active {
				line += " **[active]**"
			}
			if scope == ContextScopeFocus || scope == ContextScopeFull {
				if strings.TrimSpace(content) != "" {
					line += fmt.Sprintf(" — %d chars of content shared", len(content))
				} else {
					line += " — metadata only (no body)"
				}
			} else {
				line += " — path/metadata only (no file body at this scope)"
			}
			b.WriteString(line + "\n")
		}
	} else {
		b.WriteString("- **Open files:** none listed in this payload\n")
	}

	switch scope {
	case ContextScopeHint:
		b.WriteString("\nI only have a **hint** that a project is open — not the file tree or file bodies. Switch workspace to **Auto/Always** or ask about a specific path.")
	case ContextScopeOutline:
		b.WriteString("\nI have the **project tree** but not full file bodies. Name a path or open a file in the editor for deeper review.")
	case ContextScopeFocus, ContextScopeFull:
		b.WriteString("\nI can use the paths and any shared file bodies above for code-specific help.")
	}

	b.WriteString("\nAsk a concrete next step (e.g. where to add theme tokens) and I will base it on this project.")
	return b.String()
}

// finalizeWorkspaceVisibilityReply replaces generic LLM answers when the user asked what workspace we can see.
func (a *Agent) finalizeWorkspaceVisibilityReply(msg *protocol.Message, response string) string {
	if !looksLikeIgnoresWorkspaceVisibility(msg, response) {
		return response
	}
	if vis, ok := a.tryWorkspaceVisibilityResponse(msg); ok {
		log.Printf("[%s] Model ignored workspace visibility question; used deterministic reply", a.Info.Name)
		return vis
	}
	return response
}
