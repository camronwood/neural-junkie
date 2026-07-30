package intent

import (
	"context"
	"encoding/json"
	"testing"
)

// scriptedClassifier returns a fixed SemanticIntent for golden Resolve tests.
// It encodes the contract the local classifier is expected to produce so policy
// and stamp consumers can be tested without a live model.
type scriptedClassifier struct {
	byText map[string]SemanticIntent
	model  string
}

func (c *scriptedClassifier) Model() string {
	if c.model == "" {
		return "golden-script"
	}
	return c.model
}

func (c *scriptedClassifier) Classify(_ context.Context, features TurnFeatures) (SemanticIntent, error) {
	if intent, ok := c.byText[features.Text]; ok {
		return intent, nil
	}
	return SemanticIntent{}, errGoldenMissing(features.Text)
}

type goldenMissingError string

func (e goldenMissingError) Error() string { return "golden missing for: " + string(e) }

func errGoldenMissing(text string) error { return goldenMissingError(text) }

func mustIntent(t *testing.T, action Action, mutation Mutation, reasons ...string) SemanticIntent {
	t.Helper()
	intent := SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   action,
		MutationRequested: mutation,
		Confidence:        0.95,
		ReasonCodes:       reasons,
		RecipientType:     "assistant",
	}
	switch action {
	case ActionInspect, ActionDebug, ActionEdit, ActionRun:
		intent.Retrieval = []RetrievalTarget{RetrievalCodebase}
	case ActionArtifact:
		intent.Retrieval = []RetrievalTarget{RetrievalPriorReference}
		intent.ReasonCodes = append(intent.ReasonCodes, "durable_artifact")
	case ActionImage, ActionMusic:
		intent.MutationRequested = MutationExternal
	case ActionAnswer:
		intent.Interaction = InteractionQuestion
		intent.Retrieval = []RetrievalTarget{RetrievalMemory}
	case ActionContinue:
		intent.Interaction = InteractionContinuation
	}
	if err := intent.Validate(); err != nil {
		t.Fatalf("invalid golden intent: %v", err)
	}
	return intent
}

func TestClassifierGoldenContract(t *testing.T) {
	type gold struct {
		text           string
		features       TurnFeatures
		intent         SemanticIntent
		wantAction     Action
		wantReasonAny  string
		wantOverride   string
		openCanvasEdit bool // classifier wrongly says edit; policy open_canvas restamps
	}

	cases := []gold{
		{
			text:       "Create a Neural Canvas Mermaid diagram of this architecture",
			intent:     mustIntent(t, ActionArtifact, MutationExternal, "durable_artifact"),
			wantAction: ActionArtifact,
		},
		{
			// Live misroute: classifier emits blank_canvas but stamps ask_user.
			text: "lets create a canvas for this",
			intent: func() SemanticIntent {
				i := mustIntent(t, ActionAskUser, MutationNone, "blank_canvas")
				i.Interaction = InteractionCasual
				i.Retrieval = []RetrievalTarget{RetrievalMemory}
				return i
			}(),
			wantAction:   ActionArtifact,
			wantOverride: "canvas_reason_artifact",
			wantReasonAny: "blank_canvas",
		},
		{
			// Live misroute: blank_canvas sprayed on a meeting-notes question.
			text: "When was the last meeting that does have notes?",
			intent: func() SemanticIntent {
				i := mustIntent(t, ActionAskUser, MutationNone,
					"startup_failure", "runtime_failure", "blank_canvas", "durable_artifact")
				i.Interaction = InteractionQuestion
				return i
			}(),
			wantAction: ActionAnswer,
		},
		{
			// Follow-up continue of a broken ask_user canvas goal must still become artifact.
			text: "can we create a new canvas for this?",
			features: TurnFeatures{
				PendingActionID: "dc888200-25ae-4046-a045-8650f57630ce",
				PendingAction:   ActionAskUser,
			},
			intent: func() SemanticIntent {
				i := mustIntent(t, ActionContinue, MutationNone, "blank_canvas")
				i.ContinuationTarget = "dc888200-25ae-4046-a045-8650f57630ce"
				return i
			}(),
			wantAction:   ActionArtifact,
			wantOverride: "canvas_reason_artifact",
		},
		{
			text: "lets update the canvas to be black and white",
			features: TurnFeatures{
				OpenArtifactRenderer: "nj.mermaid",
				OpenArtifactID:       "art-1",
				HasWorkspace:         true,
				CanProposeFiles:      true,
				CanRunImplementation: true,
			},
			intent:         mustIntent(t, ActionEdit, MutationWorkspace),
			wantAction:     ActionArtifact,
			wantOverride:   "open_canvas_artifact",
			openCanvasEdit: true,
		},
		{
			text:       "draw me a map from Swansea, IL to St. Louis MO",
			intent:     mustIntent(t, ActionArtifact, MutationExternal, "maps_route"),
			wantAction: ActionArtifact,
			wantReasonAny: "maps_route",
		},
		{
			// Mis-stamped maps_route as run — policy promotes via reason code.
			text: "please map from Swansea, IL to St. Louis, MO",
			intent: func() SemanticIntent {
				i := mustIntent(t, ActionRun, MutationExternal, "maps_route")
				i.Domain = "frontend"
				i.RecipientType = "frontend"
				return i
			}(),
			wantAction:    ActionArtifact,
			wantOverride:  "canvas_reason_artifact",
			wantReasonAny: "maps_route",
		},
		{
			// Live misroute: project summarize stamped as run + external.
			text: "review and summerize the project I have open please",
			features: TurnFeatures{
				HasWorkspace:         true,
				CanProposeFiles:      true,
				CanRunImplementation: true,
			},
			intent: func() SemanticIntent {
				i := mustIntent(t, ActionRun, MutationExternal, "project_overview", "explicit_continuation")
				i.Interaction = InteractionContinuation
				return i
			}(),
			wantAction:   ActionInspect,
			wantOverride: "project_overview_inspect",
		},
		{
			text:       "Generate an image of a mountain sunset",
			intent:     mustIntent(t, ActionImage, MutationExternal),
			wantAction: ActionImage,
		},
		{
			text:       "compose an upbeat jazz track",
			intent:     mustIntent(t, ActionMusic, MutationExternal),
			wantAction: ActionMusic,
		},
		{
			text:       "write me a short story about robots",
			intent:     mustIntent(t, ActionAnswer, MutationNone),
			wantAction: ActionAnswer,
		},
		{
			text: "are you there?",
			intent: func() SemanticIntent {
				i := mustIntent(t, ActionAnswer, MutationNone)
				i.Interaction = InteractionCasual
				return i
			}(),
			wantAction: ActionAnswer,
		},
		{
			text: "ok go ahead and have a look at the workspace",
			features: TurnFeatures{
				HasWorkspace:         true,
				CanProposeFiles:      true,
				CanRunImplementation: true,
			},
			intent:     mustIntent(t, ActionInspect, MutationNone),
			wantAction: ActionInspect,
		},
		{
			text: "please continue",
			features: TurnFeatures{
				PendingActionID:      "goal-1",
				PendingAction:        ActionEdit,
				ReplyTarget:          "msg-1",
				HasWorkspace:         true,
				CanProposeFiles:      true,
				CanRunImplementation: true,
			},
			intent: func() SemanticIntent {
				i := mustIntent(t, ActionContinue, MutationWorkspace, "explicit_continuation")
				i.ContinuationTarget = "goal-1"
				return i
			}(),
			wantAction: ActionContinue,
		},
		{
			text: "the app is not booting can you fix it?",
			features: TurnFeatures{
				HasWorkspace:         true,
				CanProposeFiles:      true,
				CanRunImplementation: true,
			},
			intent:     mustIntent(t, ActionDebug, MutationWorkspace, "startup_failure"),
			wantAction: ActionDebug,
			wantReasonAny: "startup_failure",
		},
		{
			text: "create a canvas after you fix the typo in main.go",
			features: TurnFeatures{
				HasWorkspace:         true,
				CanProposeFiles:      true,
				CanRunImplementation: true,
			},
			intent:     mustIntent(t, ActionEdit, MutationWorkspace),
			wantAction: ActionEdit,
		},
	}

	byText := map[string]SemanticIntent{}
	for _, tc := range cases {
		byText[tc.text] = tc.intent
	}
	router := NewRouter(&scriptedClassifier{byText: byText}, 0.65)

	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			features := tc.features
			features.Text = tc.text
			if features.ComposerMode == "" {
				features.ComposerMode = "agent"
			}
			decision := router.Resolve(context.Background(), features)
			if decision.Action != tc.wantAction {
				raw, _ := json.Marshal(decision)
				t.Fatalf("action=%s want=%s decision=%s", decision.Action, tc.wantAction, raw)
			}
			if tc.wantReasonAny != "" && !containsString(decision.ReasonCodes, tc.wantReasonAny) {
				t.Fatalf("reason_codes=%v want containing %q", decision.ReasonCodes, tc.wantReasonAny)
			}
			if tc.wantOverride != "" && !containsString(decision.PolicyOverrides, tc.wantOverride) {
				t.Fatalf("policy_overrides=%v want containing %q", decision.PolicyOverrides, tc.wantOverride)
			}
		})
	}
}
