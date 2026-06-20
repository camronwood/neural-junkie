package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/contextcompress"
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

func TestApplyContextBudget_sectionCCR(t *testing.T) {
	contextcompress.SetDefaultStore(contextcompress.NewStore(50, 60, ""))
	t.Cleanup(func() { contextcompress.SetDefaultStore(nil) })
	enabled := true
	ai.SetHubRuntimeOptions(
		config.PerformanceConfig{ContextCompressEnabled: &enabled},
		config.OllamaConfig{},
	)
	long := strings.Repeat("workspace line\n", 800)
	prompt := "=== PERSONA ===\nfixed\n\n=== WORKSPACE CONTEXT ===\n" + long +
		ai.SystemPromptSeparator + "USER MESSAGE:\nhello"
	out, stats := applyContextBudgetForMessage(nil, prompt)
	if !stats.Truncated && len(stats.CompressedSections) == 0 {
		t.Fatalf("expected compression stats, got %+v", stats)
	}
	if !strings.Contains(out, "nj_retrieve_context") {
		t.Fatalf("expected CCR marker in output: %q", out[:min(300, len(out))])
	}
}

func TestCacheStableSystemOrder_sharedPrefix(t *testing.T) {
	summary := "=== SESSION SUMMARY ===\nUser prefers tabs.\n"
	wsA := "=== WORKSPACE CONTEXT ===\n" + strings.Repeat("alpha ", 200) + "\n"
	wsB := "=== WORKSPACE CONTEXT ===\n" + strings.Repeat("beta ", 200) + "\n"
	prefix := "=== PERSONA ===\nYou are BackendEngineer.\n\n"
	p1 := prefix + wsA + summary + ai.SystemPromptSeparator + "user"
	p2 := prefix + wsB + summary + ai.SystemPromptSeparator + "user"
	o1 := cacheStableSystemOrder(p1)
	o2 := cacheStableSystemOrder(p2)
	sys1, _, _ := strings.Cut(o1, ai.SystemPromptSeparator)
	sys2, _, _ := strings.Cut(o2, ai.SystemPromptSeparator)
	common := 0
	for common < len(sys1) && common < len(sys2) && sys1[common] == sys2[common] {
		common++
	}
	if common < len(prefix)+len(summary)-10 {
		t.Fatalf("expected stable prefix including summary, common=%d prefix+summary~%d", common, len(prefix)+len(summary))
	}
}
