package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/intent"
)

type semanticEvalCorpus struct {
	SchemaVersion int                `json:"schema_version"`
	Cases         []semanticEvalCase `json:"cases"`
}

type semanticEvalCase struct {
	Name               string                   `json:"name"`
	Text               string                   `json:"text"`
	ComposerMode       string                   `json:"composer_mode"`
	HasWorkspace       bool                     `json:"has_workspace"`
	PendingActionID    string                   `json:"pending_action_id"`
	PendingAction      intent.Action            `json:"pending_action"`
	PendingDescription string                   `json:"pending_description"`
	WantAction         intent.Action            `json:"want_action"`
	WantMutation       intent.Mutation          `json:"want_mutation"`
	WantRecipient      string                   `json:"want_recipient"`
	WantRetrieval      []intent.RetrievalTarget `json:"want_retrieval"`
	WantContinuation   string                   `json:"want_continuation"`
	WantInteraction    intent.InteractionKind   `json:"want_interaction"`
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
	for _, testCase := range corpus.Cases {
		if testCase.Name == "" || testCase.Text == "" || testCase.WantAction == "" || testCase.WantMutation == "" {
			t.Fatalf("incomplete case: %+v", testCase)
		}
	}
}

func TestLocalSemanticIntentEvaluation(t *testing.T) {
	if os.Getenv("NJ_RUN_LOCAL_SEMANTIC_EVAL") != "1" {
		t.Skip("set NJ_RUN_LOCAL_SEMANTIC_EVAL=1 to run the local utility-model corpus")
	}
	router := semanticTurnRouter(config.DefaultConfig())
	if router == nil {
		t.Fatal("semantic router unavailable")
	}
	corpus := loadSemanticEvalCorpus(t)
	correct := 0
	for _, testCase := range corpus.Cases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			features := intent.TurnFeatures{
				Text: testCase.Text, ComposerMode: testCase.ComposerMode,
				HasWorkspace:         testCase.HasWorkspace,
				CanProposeFiles:      testCase.ComposerMode == "agent" || testCase.ComposerMode == "export",
				CanRunImplementation: testCase.ComposerMode == "agent" || testCase.ComposerMode == "export",
				PendingActionID:      testCase.PendingActionID,
				PendingAction:        testCase.PendingAction,
				PendingDescription:   testCase.PendingDescription,
			}
			decision := router.Resolve(context.Background(), features)
			if decision.Action == testCase.WantAction {
				correct++
			}
			if decision.Action != testCase.WantAction || decision.Mutation != testCase.WantMutation {
				t.Errorf("decision=%+v want action=%s mutation=%s", decision, testCase.WantAction, testCase.WantMutation)
			}
			if testCase.WantRecipient != "" && decision.RecipientType != testCase.WantRecipient {
				t.Errorf("recipient=%s want=%s", decision.RecipientType, testCase.WantRecipient)
			}
			if testCase.WantContinuation != "" && decision.ContinuationTarget != testCase.WantContinuation {
				t.Errorf("continuation=%s want=%s", decision.ContinuationTarget, testCase.WantContinuation)
			}
			if testCase.WantInteraction != "" && decision.Interaction != testCase.WantInteraction {
				t.Errorf("interaction=%s want=%s", decision.Interaction, testCase.WantInteraction)
			}
			for _, target := range testCase.WantRetrieval {
				if !containsRetrieval(decision.Retrieval, target) {
					t.Errorf("retrieval=%v missing=%s", decision.Retrieval, target)
				}
			}
			if (testCase.ComposerMode == "ask" || testCase.ComposerMode == "plan") &&
				decision.Mutation != intent.MutationNone {
				t.Errorf("unsafe mutation in %s mode: %+v", testCase.ComposerMode, decision)
			}
		})
	}
	if accuracy := float64(correct) / float64(len(corpus.Cases)); accuracy < 0.9 {
		t.Fatalf("semantic action accuracy %.2f below 0.90", accuracy)
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
