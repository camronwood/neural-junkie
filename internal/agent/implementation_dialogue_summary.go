package agent

import (
	"fmt"
	"strings"
)

const implDialogueCompressAfterSteps = 15

func (s *ImplementationSessionState) noteToolStep() {
	if s == nil {
		return
	}
	s.ToolStepCount++
	if s.ToolStepCount >= implDialogueCompressAfterSteps && s.DialogueSummary == "" {
		s.DialogueSummary = s.buildDialogueSummary()
	}
}

func (s *ImplementationSessionState) buildDialogueSummary() string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("Compressed session context (earlier tool output omitted):\n")
	if len(s.LastReadPaths) > 0 {
		b.WriteString("- Files read: ")
		b.WriteString(strings.Join(limitStrings(s.LastReadPaths, 12), ", "))
		b.WriteString("\n")
	}
	if len(s.FilesChanged) > 0 {
		b.WriteString("- Files changed: ")
		b.WriteString(strings.Join(limitStrings(s.FilesChanged, 12), ", "))
		b.WriteString("\n")
	}
	if len(s.PreflightErrors) > 0 {
		b.WriteString("- Preflight errors: ")
		b.WriteString(strings.Join(limitStrings(s.PreflightErrors, 5), "; "))
		b.WriteString("\n")
	}
	if cmd := strings.TrimSpace(s.LastFailedCommand); cmd != "" {
		b.WriteString("- Last failed command: ")
		b.WriteString(cmd)
		b.WriteString("\n")
	}
	if out := strings.TrimSpace(s.LastCommandOutput()); out != "" {
		b.WriteString("- Last command excerpt: ")
		b.WriteString(truncateImplLog(out, 600))
		b.WriteString("\n")
	}
	if len(s.CommandHistory) > 0 {
		b.WriteString(fmt.Sprintf("- Commands run: %d\n", len(s.CommandHistory)))
	}
	if s.PlaybookUsedName != "" {
		b.WriteString("- Playbook used: ")
		b.WriteString(s.PlaybookUsedName)
		b.WriteString("\n")
	}
	if kind := repairFailureKindLabel(s.LastRepairFailureKind); kind != "" {
		b.WriteString("- Last repair category: ")
		b.WriteString(kind)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func limitStrings(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	out := append([]string(nil), items[:max]...)
	out = append(out, fmt.Sprintf("...(+%d more)", len(items)-max))
	return out
}

func appendDialogueSummaryPrompt(prompt string, state *ImplementationSessionState) string {
	if state == nil || strings.TrimSpace(state.DialogueSummary) == "" {
		return prompt
	}
	return prompt + "\n=== SESSION SUMMARY ===\n" + state.DialogueSummary + "\n"
}
