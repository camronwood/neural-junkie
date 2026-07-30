package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/intent"
)

type semanticEvalCorpus struct {
	SchemaVersion int                `json:"schema_version"`
	Cases         []semanticEvalCase `json:"cases"`
}

type semanticEvalCase struct {
	Name                 string                   `json:"name"`
	Text                 string                   `json:"text"`
	ComposerMode         string                   `json:"composer_mode"`
	HasWorkspace         bool                     `json:"has_workspace"`
	PendingActionID      string                   `json:"pending_action_id"`
	PendingAction        intent.Action            `json:"pending_action"`
	PendingDescription   string                   `json:"pending_description"`
	OpenArtifactID       string                   `json:"open_artifact_id"`
	OpenArtifactRenderer string                   `json:"open_artifact_renderer"`
	OpenArtifactTitle    string                   `json:"open_artifact_title"`
	WantAction           intent.Action            `json:"want_action"`
	WantActions          []intent.Action          `json:"want_actions"`
	WantMutation         intent.Mutation          `json:"want_mutation"`
	WantRecipient        string                   `json:"want_recipient"`
	WantRetrieval        []intent.RetrievalTarget `json:"want_retrieval"`
	WantContinuation     string                   `json:"want_continuation"`
	WantInteraction      intent.InteractionKind   `json:"want_interaction"`
	PolicyClass          string                   `json:"policy_class"`
}

func loadSemanticEvalCorpus(t *testing.T) semanticEvalCorpus {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(filename), "..", "..", "scenarios", "routing", "semantic-intents.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var corpus semanticEvalCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.SchemaVersion != intent.SchemaVersion || len(corpus.Cases) == 0 {
		t.Fatalf("invalid semantic corpus: %+v", corpus)
	}
	return corpus
}

func TestSemanticIntentCorpusIsValid(t *testing.T) {
	corpus := loadSemanticEvalCorpus(t)
	if len(corpus.Cases) < 50 {
		t.Fatalf("corpus too small: %d", len(corpus.Cases))
	}
	for _, testCase := range corpus.Cases {
		if testCase.Name == "" || testCase.Text == "" || testCase.WantAction == "" || testCase.WantMutation == "" {
			t.Fatalf("incomplete case: %+v", testCase)
		}
	}
}

func TestLocalSemanticIntentEvaluation(t *testing.T) {
	if os.Getenv("NJ_RUN_LOCAL_SEMANTIC_EVAL") != "1" {
		t.Skip("set NJ_RUN_LOCAL_SEMANTIC_EVAL=1 (or make semantic-eval) to run live classify+policy corpus")
	}
	cfg := config.DefaultConfig()
	if model := os.Getenv("NJ_SEMANTIC_CLASSIFIER_MODEL"); model != "" {
		cfg.Routing.SemanticClassifierModel = model
	}
	router := semanticTurnRouter(cfg)
	if router == nil {
		t.Fatal("semantic router unavailable")
	}
	corpus := loadSemanticEvalCorpus(t)
	correctAction := 0
	correctFull := 0
	abstentionCount := 0
	misstampCount := 0
	structuralMissCount := 0
	sourceCounts := map[string]int{}
	type failRow struct {
		Name, PolicyClass string
		GotAction         intent.Action
		WantAction        intent.Action
		GotMutation       intent.Mutation
		WantMutation      intent.Mutation
		Source            intent.Source
		Overrides         []string
		FailKind          string
		AbstentionReason  string
	}
	var fails []failRow
	for _, testCase := range corpus.Cases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			features := intent.TurnFeatures{
				Text:                 testCase.Text,
				ComposerMode:         testCase.ComposerMode,
				HasWorkspace:         testCase.HasWorkspace,
				CanProposeFiles:      testCase.ComposerMode == "agent" || testCase.ComposerMode == "export",
				CanRunImplementation: testCase.ComposerMode == "agent" || testCase.ComposerMode == "export",
				PendingActionID:      testCase.PendingActionID,
				PendingAction:        testCase.PendingAction,
				PendingDescription:   testCase.PendingDescription,
				OpenArtifactID:       testCase.OpenArtifactID,
				OpenArtifactRenderer: testCase.OpenArtifactRenderer,
				OpenArtifactTitle:    testCase.OpenArtifactTitle,
			}
			decision := router.Resolve(context.Background(), features)
			sourceCounts[string(decision.Source)]++
			actionOK := actionAccepted(decision.Action, testCase.WantAction, testCase.WantActions)
			fullOK := actionOK && decision.Mutation == testCase.WantMutation
			if actionOK {
				correctAction++
			}
			if fullOK {
				correctFull++
			}
			if !fullOK {
				kind := semanticEvalFailKind(decision)
				switch kind {
				case "abstention":
					abstentionCount++
				case "misstamp":
					misstampCount++
				default:
					structuralMissCount++
				}
				fails = append(fails, failRow{
					Name: testCase.Name, PolicyClass: testCase.PolicyClass,
					GotAction: decision.Action, WantAction: testCase.WantAction,
					GotMutation: decision.Mutation, WantMutation: testCase.WantMutation,
					Source: decision.Source, Overrides: decision.PolicyOverrides,
					FailKind: kind, AbstentionReason: decision.AbstentionReason,
				})
				// Per-case misses are diagnostics; ship bar is the dual gate below.
				t.Logf("case miss action=%s mutation=%s source=%s fail_kind=%s overrides=%v want action=%s|%v mutation=%s",
					decision.Action, decision.Mutation, decision.Source, kind, decision.PolicyOverrides,
					testCase.WantAction, testCase.WantActions, testCase.WantMutation)
			}
			if testCase.WantRecipient != "" && decision.RecipientType != testCase.WantRecipient {
				t.Logf("recipient=%s want=%s (soft)", decision.RecipientType, testCase.WantRecipient)
			}
			if testCase.WantContinuation != "" && decision.ContinuationTarget != testCase.WantContinuation {
				t.Logf("continuation=%s want=%s (soft)", decision.ContinuationTarget, testCase.WantContinuation)
			}
			if testCase.WantInteraction != "" && decision.Interaction != testCase.WantInteraction {
				t.Logf("interaction=%s want=%s (soft)", decision.Interaction, testCase.WantInteraction)
			}
			for _, target := range testCase.WantRetrieval {
				if !containsRetrieval(decision.Retrieval, target) {
					t.Logf("retrieval=%v missing=%s (soft)", decision.Retrieval, target)
				}
			}
			if (testCase.ComposerMode == "ask" || testCase.ComposerMode == "plan") &&
				decision.Mutation != intent.MutationNone {
				t.Errorf("unsafe mutation in %s mode: %+v", testCase.ComposerMode, decision)
			}
		})
	}
	n := float64(len(corpus.Cases))
	actionAcc := float64(correctAction) / n
	fullAcc := float64(correctFull) / n
	abstentionRate := float64(abstentionCount) / n
	misstampRate := float64(misstampCount) / n
	nCommitted := len(corpus.Cases) - abstentionCount
	committedActionAcc := 0.0
	committedFullAcc := 0.0
	if nCommitted > 0 {
		// Denominator excludes abstention fails only (plan formula).
		committedActionAcc = float64(correctAction) / float64(nCommitted)
		committedFullAcc = float64(correctFull) / float64(nCommitted)
	}
	model := strings.TrimSpace(cfg.Routing.SemanticClassifierModel)
	if model == "" {
		model = config.SemanticClassifierOllamaModel
	}
	t.Logf("semantic live eval model=%s action_accuracy=%.3f full_accuracy=%.3f abstention_rate=%.3f misstamp_rate=%.3f committed_action_accuracy=%.3f n=%d fails=%d (abstention=%d misstamp=%d structural_miss=%d)",
		model, actionAcc, fullAcc, abstentionRate, misstampRate, committedActionAcc,
		len(corpus.Cases), len(fails), abstentionCount, misstampCount, structuralMissCount)
	if out := os.Getenv("NJ_SEMANTIC_EVAL_OUT"); out != "" {
		payload := map[string]any{
			"model":                      model,
			"n":                          len(corpus.Cases),
			"action_accuracy":            actionAcc,
			"full_accuracy":              fullAcc,
			"abstention_count":           abstentionCount,
			"misstamp_count":             misstampCount,
			"structural_miss_count":      structuralMissCount,
			"abstention_rate":            abstentionRate,
			"misstamp_rate":              misstampRate,
			"n_committed":                nCommitted,
			"committed_action_accuracy":  committedActionAcc,
			"committed_full_accuracy":    committedFullAcc,
			"source_counts":              sourceCounts,
			"fails":                      fails,
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		outPath := out
		if !filepath.IsAbs(outPath) {
			_, filename, _, _ := runtime.Caller(0)
			repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
			outPath = filepath.Join(repoRoot, outPath)
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			t.Errorf("mkdir eval out: %v", err)
		} else if err := os.WriteFile(outPath, data, 0o644); err != nil {
			t.Errorf("write eval out: %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "wrote %s\n", outPath)
		}
	}
	minAcc := 0.90
	if v := os.Getenv("NJ_SEMANTIC_EVAL_MIN_ACC"); v != "" {
		if parsed, err := fmt.Sscanf(v, "%f", &minAcc); err != nil || parsed != 1 {
			t.Fatalf("bad NJ_SEMANTIC_EVAL_MIN_ACC=%q", v)
		}
	}
	maxMisstamp := 0.05
	if v := os.Getenv("NJ_SEMANTIC_EVAL_MAX_MISSTAMP"); v != "" {
		if parsed, err := fmt.Sscanf(v, "%f", &maxMisstamp); err != nil || parsed != 1 {
			t.Fatalf("bad NJ_SEMANTIC_EVAL_MAX_MISSTAMP=%q", v)
		}
	}
	if actionAcc < minAcc {
		t.Fatalf("semantic action accuracy %.2f below %.2f (model=%s)", actionAcc, minAcc, model)
	}
	if misstampRate > maxMisstamp {
		t.Fatalf("semantic misstamp_rate %.3f above %.3f (model=%s) — confident wrong stamps are the dangerous class",
			misstampRate, maxMisstamp, model)
	}
}

// semanticEvalFailKind classifies an end-to-end decision failure for diagnostics.
// Abstention is safer (answer/none fallback); misstamp is a confident wrong stamp.
func semanticEvalFailKind(d intent.TurnDecision) string {
	switch d.Source {
	case intent.SourceSafeFallback:
		return "abstention"
	case intent.SourceLocalModel:
		return "misstamp"
	case intent.SourceStructural:
		if strings.TrimSpace(d.AbstentionReason) != "" {
			return "abstention"
		}
		return "structural_miss"
	default:
		return "structural_miss"
	}
}

func TestSemanticEvalFailKind(t *testing.T) {
	cases := []struct {
		name string
		d    intent.TurnDecision
		want string
	}{
		{"safe_fallback", intent.TurnDecision{Source: intent.SourceSafeFallback}, "abstention"},
		{"local_model", intent.TurnDecision{Source: intent.SourceLocalModel}, "misstamp"},
		{"structural_with_reason", intent.TurnDecision{Source: intent.SourceStructural, AbstentionReason: "classifier_error"}, "abstention"},
		{"structural_pure", intent.TurnDecision{Source: intent.SourceStructural}, "structural_miss"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := semanticEvalFailKind(tc.d); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func containsRetrieval(values []intent.RetrievalTarget, target intent.RetrievalTarget) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func actionAccepted(got, want intent.Action, alts []intent.Action) bool {
	if got == want {
		return true
	}
	for _, a := range alts {
		if got == a {
			return true
		}
	}
	return false
}
