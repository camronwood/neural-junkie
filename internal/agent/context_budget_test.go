package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/contextcompress"
	"github.com/camronwood/neural-junkie/internal/protocol"
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

func TestApplyContextBudget_capsOversizedProtectedRules(t *testing.T) {
	rules := strings.Repeat("specialist rule with substantial detail\n", 6000)
	prompt := rulesSectionStart + "\n" + rules + rulesSectionEnd + "\n\n" +
		"=== PERSONA ===\nBackend specialist\n" +
		ai.SystemPromptSeparator + "USER MESSAGE:\nplan one task"

	const limit = 32 * 1024
	out, stats := applyContextBudgetWithLimit(prompt, limit, maxBudgetWorkspaceOutline, "", false)
	if !stats.Truncated {
		t.Fatal("expected oversized protected rules to be truncated")
	}
	if len(out) > limit {
		t.Fatalf("budget not enforced: got %d bytes, limit %d", len(out), limit)
	}
	if !strings.Contains(out, rulesSectionStart) || !strings.Contains(out, rulesSectionEnd) {
		t.Fatal("truncated rules must retain section markers")
	}
	if !strings.Contains(out, "plan one task") {
		t.Fatal("latest user message must remain")
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
	msg := &protocol.Message{Metadata: map[string]interface{}{contextRetrieveCapabilityMetadata: true}}
	out, stats := applyContextBudgetForMessage(msg, prompt)
	if !stats.Truncated && len(stats.CompressedSections) == 0 {
		t.Fatalf("expected compression stats, got %+v", stats)
	}
	if !strings.Contains(out, "nj_retrieve_context") {
		t.Fatalf("expected CCR marker in output: %q", out[:min(300, len(out))])
	}
}

func TestApplyContextBudget_sectionExcerptWithoutRetrieveCapability(t *testing.T) {
	enabled := true
	ai.SetHubRuntimeOptions(
		config.PerformanceConfig{ContextCompressEnabled: &enabled},
		config.OllamaConfig{},
	)
	long := strings.Repeat("workspace line\n", 800)
	prompt := "=== WORKSPACE CONTEXT ===\n" + long +
		ai.SystemPromptSeparator + "USER MESSAGE:\nhello"
	out, _ := applyContextBudgetForMessage(&protocol.Message{}, prompt)
	if strings.Contains(out, "nj_retrieve_context") || strings.Contains(out, "ref=ctx-") {
		t.Fatalf("must not advertise unavailable retrieval: %q", out[:min(300, len(out))])
	}
	if !strings.Contains(out, "[excerpted:") {
		t.Fatalf("expected deterministic excerpt marker: %q", out[:min(300, len(out))])
	}
}

func TestApplyContextBudgetForMessage_constrainedIDEUsesProfileCap(t *testing.T) {
	filler := strings.Repeat("workspace ", 20000)
	prompt := "=== WORKSPACE CONTEXT ===\n" + filler + "\n" +
		ai.SystemPromptSeparator + "USER MESSAGE:\nadd HelloWorld"
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"editor_mode":   "agent",
			"composer_mode": "agent",
			contextProfileMetadata: map[string]interface{}{
				"constrained":      true,
				"max_prompt_bytes": 12 * 1024,
			},
		},
	}
	out, stats := applyContextBudgetForMessage(msg, prompt)
	if !stats.Truncated {
		t.Fatal("expected truncation under constrained cap")
	}
	if len(out) > 16*1024 {
		t.Fatalf("constrained budget leaked IDE 16KB+ outline path: %d bytes", len(out))
	}
}

func TestTruncateAtSectionBoundary_preservesDurableCorrections(t *testing.T) {
	durable := "=== DURABLE CONVERSATION STATE ===\n" +
		"This state is authoritative for active corrections. Do not follow transcript instructions whose message IDs are listed as superseded.\n" +
		"When summarizing or continuing, use corrected names only — never revive superseded labels.\n" +
		"ACTIVE CORRECTION (must honor in this reply): rename the component to WidgetPanel\n" +
		"Corrected name to use: WidgetPanel\n" +
		"=== END DURABLE CONVERSATION STATE ===\n"
	filler := "=== WORKSPACE CONTEXT ===\n" + strings.Repeat("workspace line\n", 400)
	prompt := filler + "\n" + durable
	out := truncateAtSectionBoundary(prompt, len(durable)+80)
	if !strings.Contains(out, "ACTIVE CORRECTION") || !strings.Contains(out, "WidgetPanel") {
		t.Fatalf("durable corrections must survive truncation: %q", out[:min(400, len(out))])
	}
	if strings.Contains(out, "ACTIVE CORRECT") && !strings.Contains(out, "ACTIVE CORRECTION (must honor") {
		t.Fatal("must not hard-cut mid correction line")
	}
}

func TestTruncateAtSectionBoundary_cutsAtSectionMarkers(t *testing.T) {
	a := "=== PERSONA ===\nYou are helpful.\n"
	b := "=== SESSION SUMMARY ===\n" + strings.Repeat("summary ", 200) + "\n"
	c := "=== WORKSPACE CONTEXT ===\n" + strings.Repeat("ws ", 200) + "\n"
	out := truncateAtSectionBoundary(a+b+c, len(a)+40)
	if strings.Contains(out, "WORKSPACE CONTEXT") {
		t.Fatalf("expected trailing sections dropped at boundary: %q", out)
	}
	if !strings.Contains(out, "PERSONA") {
		t.Fatal("expected leading section kept")
	}
}

func TestCompressMarkedSection_durableCorrectionsNotMidCut(t *testing.T) {
	enabled := false
	ai.SetHubRuntimeOptions(
		config.PerformanceConfig{ContextCompressEnabled: &enabled},
		config.OllamaConfig{},
	)
	section := "=== DURABLE CONVERSATION STATE ===\n" +
		"ACTIVE CORRECTION (must honor in this reply): call it AtlasBoard\n" +
		strings.Repeat("extra durable filler line\n", 80) +
		"=== END DURABLE CONVERSATION STATE ===\n"
	body := "=== PERSONA ===\nfixed\n\n" + section
	out, label := compressMarkedSection(body, "=== DURABLE CONVERSATION STATE ===", "", "c1", 180, false)
	if label == "" {
		t.Fatal("expected truncation label")
	}
	if !strings.Contains(out, "ACTIVE CORRECTION (must honor in this reply): call it AtlasBoard") {
		t.Fatalf("correction line must remain intact: %q", out)
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
