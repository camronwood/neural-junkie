package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
)

func TestApplyContextBudget_preservesUserRulesSection(t *testing.T) {
	rules := strings.Repeat("rule-line\n", 200)
	filler := strings.Repeat("workspace ", 8000)
	prompt := rulesSectionStart + "\n" + rules + "\n" + rulesSectionEnd + "\n\n" +
		"=== SESSION SUMMARY ===\n" + filler + "\n\n" +
		"=== WORKSPACE CONTEXT ===\n" + filler + "\n" +
		ai.SystemPromptSeparator + "USER MESSAGE:\nhello"

	out, stats := applyContextBudget(prompt)
	if !stats.Truncated {
		t.Fatal("expected truncation")
	}
	if !strings.Contains(out, rulesSectionStart) || !strings.Contains(out, rulesSectionEnd) {
		t.Fatalf("user rules section should survive budget trim: %q", out[:min(200, len(out))])
	}
	if !strings.Contains(out, "hello") {
		t.Fatal("user message must remain intact")
	}
}
