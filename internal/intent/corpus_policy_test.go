package intent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// corpusFile mirrors scenarios/routing/semantic-intents.json for policy graduation tests.
type corpusFile struct {
	SchemaVersion int          `json:"schema_version"`
	Cases         []corpusCase `json:"cases"`
}

type corpusCase struct {
	Name               string            `json:"name"`
	Text               string            `json:"text"`
	ComposerMode       string            `json:"composer_mode"`
	HasWorkspace       bool              `json:"has_workspace"`
	PendingActionID    string            `json:"pending_action_id"`
	PendingAction      Action            `json:"pending_action"`
	PendingDescription string            `json:"pending_description"`
	OpenArtifactID     string            `json:"open_artifact_id"`
	OpenArtifactRend   string            `json:"open_artifact_renderer"`
	OpenArtifactTitle  string            `json:"open_artifact_title"`
	StampAction        Action            `json:"stamp_action"`
	StampMutation      Mutation          `json:"stamp_mutation"`
	StampInteraction   InteractionKind   `json:"stamp_interaction"`
	StampReasonCodes   []string          `json:"stamp_reason_codes"`
	StampContextTier  ContextTier      `json:"stamp_context_tier"`
	StampSubject       ContextSubject    `json:"stamp_subject"`
	StampReviewMode    ReviewMode        `json:"stamp_review_mode"`
	WantAction         Action            `json:"want_action"`
	WantActions        []Action          `json:"want_actions"`
	WantMutation       Mutation          `json:"want_mutation"`
	WantRecipient      string            `json:"want_recipient"`
	WantRetrieval      []RetrievalTarget `json:"want_retrieval"`
	WantContinuation   string            `json:"want_continuation"`
	WantInteraction    InteractionKind   `json:"want_interaction"`
	WantOverrideAny    string            `json:"want_override_any"`
	WantContextTier   ContextTier      `json:"want_context_tier"`
	WantSubject        ContextSubject    `json:"want_subject"`
	WantReviewMode     ReviewMode        `json:"want_review_mode"`
	PolicyClass        string            `json:"policy_class"`
}

func loadRoutingCorpus(t *testing.T) corpusFile {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(filename), "..", "..", "scenarios", "routing", "semantic-intents.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var corpus corpusFile
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.SchemaVersion != SchemaVersion || len(corpus.Cases) == 0 {
		t.Fatalf("invalid corpus: version=%d n=%d", corpus.SchemaVersion, len(corpus.Cases))
	}
	return corpus
}

func TestSemanticCorpusShape(t *testing.T) {
	corpus := loadRoutingCorpus(t)
	if len(corpus.Cases) < 50 {
		t.Fatalf("corpus too small: %d (want >= 50 for semantic graduation)", len(corpus.Cases))
	}
	seen := map[string]bool{}
	for _, c := range corpus.Cases {
		if c.Name == "" || c.Text == "" || c.WantAction == "" || c.WantMutation == "" {
			t.Fatalf("incomplete case: %+v", c)
		}
		if seen[c.Name] {
			t.Fatalf("duplicate case name %q", c.Name)
		}
		seen[c.Name] = true
		if c.PolicyClass == "" {
			t.Fatalf("case %s missing policy_class", c.Name)
		}
	}
}

// TestResolvePolicyAgainstCorpus feeds stamp_* (or want_* when stamp omitted) through
// ResolvePolicy. This is the CI gate that lets us delete LooksLike* branches once
// gold stamps + structural features alone produce want_*.
func TestResolvePolicyAgainstCorpus(t *testing.T) {
	corpus := loadRoutingCorpus(t)
	for _, c := range corpus.Cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			features := TurnFeatures{
				Text:                 c.Text,
				ComposerMode:         c.ComposerMode,
				HasWorkspace:         c.HasWorkspace,
				CanProposeFiles:      c.ComposerMode == "agent" || c.ComposerMode == "export",
				CanRunImplementation: c.ComposerMode == "agent" || c.ComposerMode == "export",
				PendingActionID:      c.PendingActionID,
				PendingAction:        c.PendingAction,
				PendingDescription:   c.PendingDescription,
				OpenArtifactID:       c.OpenArtifactID,
				OpenArtifactRenderer: c.OpenArtifactRend,
				OpenArtifactTitle:    c.OpenArtifactTitle,
			}
			stampAction := c.StampAction
			if stampAction == "" {
				stampAction = c.WantAction
			}
			stampMutation := c.StampMutation
			if stampMutation == "" {
				stampMutation = c.WantMutation
			}
			stampInteraction := c.StampInteraction
			if stampInteraction == "" {
				if c.WantInteraction != "" {
					stampInteraction = c.WantInteraction
				} else if stampAction == ActionAnswer {
					stampInteraction = InteractionQuestion
				} else {
					stampInteraction = InteractionTask
				}
			}
			semantic := SemanticIntent{
				SchemaVersion:      SchemaVersion,
				Interaction:        stampInteraction,
				RequestedAction:    stampAction,
				MutationRequested:  stampMutation,
				Confidence:         0.9,
				ReasonCodes:        append([]string(nil), c.StampReasonCodes...),
				RecipientType:      "assistant",
				ContinuationTarget: c.WantContinuation,
				ContextTier:       c.StampContextTier,
				Subject:            c.StampSubject,
				ReviewMode:         c.StampReviewMode,
			}
			if c.PendingActionID != "" && stampAction == ActionContinue {
				semantic.ContinuationTarget = c.PendingActionID
			}
			decision := ResolvePolicy(features, semantic, SourceLocalModel)
			if !actionAccepted(decision.Action, c.WantAction, c.WantActions) {
				t.Errorf("action=%s want=%s|%v overrides=%v reasons=%v", decision.Action, c.WantAction, c.WantActions, decision.PolicyOverrides, decision.ReasonCodes)
			}
			if decision.Mutation != c.WantMutation {
				t.Errorf("mutation=%s want=%s overrides=%v", decision.Mutation, c.WantMutation, decision.PolicyOverrides)
			}
			if c.WantOverrideAny != "" && !containsString(decision.PolicyOverrides, c.WantOverrideAny) {
				t.Errorf("overrides=%v missing %q", decision.PolicyOverrides, c.WantOverrideAny)
			}
			if c.WantContinuation != "" && decision.ContinuationTarget != c.WantContinuation {
				t.Errorf("continuation=%s want=%s", decision.ContinuationTarget, c.WantContinuation)
			}
			if c.WantInteraction != "" && decision.Interaction != c.WantInteraction {
				t.Errorf("interaction=%s want=%s", decision.Interaction, c.WantInteraction)
			}
			for _, target := range c.WantRetrieval {
				if !containsRetrievalTarget(decision.Retrieval, target) {
					t.Errorf("retrieval=%v missing %s", decision.Retrieval, target)
				}
			}
			if c.WantContextTier != "" && decision.ContextPlan.Tier != c.WantContextTier {
				t.Errorf("context_tier=%s want=%s plan=%+v", decision.ContextPlan.Tier, c.WantContextTier, decision.ContextPlan)
			}
			if c.WantSubject != "" && decision.ContextPlan.Subject != c.WantSubject {
				t.Errorf("subject=%s want=%s plan=%+v", decision.ContextPlan.Subject, c.WantSubject, decision.ContextPlan)
			}
			if c.WantReviewMode != "" && decision.ContextPlan.ReviewMode != c.WantReviewMode {
				t.Errorf("review_mode=%s want=%s plan=%+v", decision.ContextPlan.ReviewMode, c.WantReviewMode, decision.ContextPlan)
			}
		})
	}
}

// TestResolvePolicyDualGateTextGatesOff compares corpus outcomes with LooksLike*
// policy overrides disabled. Failures here are graduation blockers before helper deletion.
func TestResolvePolicyDualGateTextGatesOff(t *testing.T) {
	prevDisabled := !TextGatesEnabled()
	SetTextGatesDisabled(true)
	t.Cleanup(func() { SetTextGatesDisabled(prevDisabled) })

	corpus := loadRoutingCorpus(t)
	var disagreements int
	for _, c := range corpus.Cases {
		features := TurnFeatures{
			Text:                 c.Text,
			ComposerMode:         c.ComposerMode,
			HasWorkspace:         c.HasWorkspace,
			CanProposeFiles:      c.ComposerMode == "agent" || c.ComposerMode == "export",
			CanRunImplementation: c.ComposerMode == "agent" || c.ComposerMode == "export",
			PendingActionID:      c.PendingActionID,
			PendingAction:        c.PendingAction,
			PendingDescription:   c.PendingDescription,
			OpenArtifactID:       c.OpenArtifactID,
			OpenArtifactRenderer: c.OpenArtifactRend,
			OpenArtifactTitle:    c.OpenArtifactTitle,
		}
		stampAction := c.StampAction
		if stampAction == "" {
			stampAction = c.WantAction
		}
		stampMutation := c.StampMutation
		if stampMutation == "" {
			stampMutation = c.WantMutation
		}
		stampInteraction := c.StampInteraction
		if stampInteraction == "" {
			stampInteraction = InteractionTask
		}
		semantic := SemanticIntent{
			SchemaVersion:     SchemaVersion,
			Interaction:       stampInteraction,
			RequestedAction:   stampAction,
			MutationRequested: stampMutation,
			Confidence:        0.9,
			ReasonCodes:       append([]string(nil), c.StampReasonCodes...),
			RecipientType:     "assistant",
			ContextTier:      c.StampContextTier,
			Subject:           c.StampSubject,
			ReviewMode:        c.StampReviewMode,
		}
		decision := ResolvePolicy(features, semantic, SourceLocalModel)
		if !actionAccepted(decision.Action, c.WantAction, c.WantActions) || decision.Mutation != c.WantMutation {
			disagreements++
			t.Logf("dual-gate disagreement %s: got action=%s mutation=%s want=%s/%s overrides=%v",
				c.Name, decision.Action, decision.Mutation, c.WantAction, c.WantMutation, decision.PolicyOverrides)
		}
	}
	// Soft gate while graduating: keep disagreement rate under 20% of corpus.
	max := len(corpus.Cases) / 5
	if disagreements > max {
		t.Fatalf("text-gates-off disagreements=%d exceed %d (%.0f%% of corpus); keep graduating reason_codes",
			disagreements, max, 100*float64(disagreements)/float64(len(corpus.Cases)))
	}
}

func actionAccepted(got, want Action, alts []Action) bool {
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
