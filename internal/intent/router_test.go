package intent

import (
	"context"
	"errors"
	"testing"
)

type testClassifier struct {
	intent SemanticIntent
	err    error
	calls  int
}

func (c *testClassifier) Classify(context.Context, TurnFeatures) (SemanticIntent, error) {
	c.calls++
	return c.intent, c.err
}

func (c *testClassifier) Model() string { return "local-test" }

func TestRouterPlanDirectiveSurvivesClassifierFailure(t *testing.T) {
	classifier := &testClassifier{err: errors.New("unavailable")}
	decision := NewRouter(classifier, 0.6).Resolve(context.Background(), TurnFeatures{
		Text:         "What should we change?",
		ComposerMode: "plan",
		HasWorkspace: true,
	})
	if classifier.calls != 1 {
		t.Fatalf("classifier calls=%d, want 1", classifier.calls)
	}
	if decision.Action != ActionPlan || decision.Mutation != MutationNone || decision.Source != SourceStructural {
		t.Fatalf("decision=%+v, want structural plan", decision)
	}
}

func TestRouterUsesClassifierForNaturalLanguage(t *testing.T) {
	classifier := &testClassifier{intent: SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionDebug,
		Retrieval:         []RetrievalTarget{RetrievalCodebase},
		MutationRequested: MutationNone,
		Confidence:        0.91,
		ReasonCodes:       []string{"runtime_failure"},
	}}
	decision := NewRouter(classifier, 0.6).Resolve(context.Background(), TurnFeatures{
		Text:                 "The program fails during startup; diagnose it.",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	})
	if classifier.calls != 1 {
		t.Fatalf("classifier calls=%d, want 1", classifier.calls)
	}
	if decision.Action != ActionDebug || decision.Source != SourceLocalModel || decision.ClassifierModel != "local-test" {
		t.Fatalf("decision=%+v, want local debug", decision)
	}
}

func TestPolicyNeverLetsAskModeMutate(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		ComposerMode:         "ask",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.99,
	}, SourceLocalModel)
	if decision.Action != ActionAnswer || decision.Mutation != MutationNone {
		t.Fatalf("decision=%+v, ask mode authorized mutation", decision)
	}
}

func TestPolicyRequiresWorkspaceForEdit(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		ComposerMode:         "agent",
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionAskUser || decision.Mutation != MutationNone {
		t.Fatalf("decision=%+v, want workspace clarification", decision)
	}
}

func TestPolicyAddsCodebaseRetrievalForWorkspaceEdits(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		Domain:            "frontend",
		RecipientType:     "frontend",
		MutationRequested: MutationWorkspace,
		Confidence:        0.95,
	}, SourceLocalModel)
	if decision.Action != ActionEdit || decision.Mutation != MutationWorkspace {
		t.Fatalf("decision=%+v, want workspace edit", decision)
	}
	if !containsRetrievalTarget(decision.Retrieval, RetrievalCodebase) {
		t.Fatalf("retrieval=%v, want codebase", decision.Retrieval)
	}
}

func TestLowConfidenceAbstainsSafely(t *testing.T) {
	classifier := &testClassifier{intent: SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.2,
	}}
	decision := NewRouter(classifier, 0.7).Resolve(context.Background(), TurnFeatures{Text: "ambiguous"})
	if decision.Action != ActionAnswer || decision.Mutation != MutationNone || decision.Source != SourceSafeFallback {
		t.Fatalf("decision=%+v, want safe fallback", decision)
	}
}

func TestSemanticIntentRejectsInvalidEnumAndConfidence(t *testing.T) {
	base := SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.5,
	}
	invalid := base
	invalid.RequestedAction = Action("destroy")
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid action accepted")
	}
	invalid = base
	invalid.Confidence = 1.1
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid confidence accepted")
	}
}

func TestSemanticConsistencyUsesTypedContinuationState(t *testing.T) {
	semantic := SemanticIntent{
		SchemaVersion: SchemaVersion, Interaction: InteractionContinuation,
		RequestedAction: ActionRun, MutationRequested: MutationNone,
		Confidence: 0.95,
	}
	features := TurnFeatures{PendingActionID: "goal-1", PendingAction: ActionEdit}
	normalizeSemanticConsistency(features, &semantic)
	if semantic.RequestedAction != ActionContinue || semantic.ContinuationTarget != "goal-1" ||
		semantic.MutationRequested != MutationWorkspace {
		t.Fatalf("semantic=%+v", semantic)
	}
}

func TestSemanticConsistencyDerivesArtifactMutationAndRecipient(t *testing.T) {
	artifact := SemanticIntent{
		SchemaVersion: SchemaVersion, Interaction: InteractionTask,
		RequestedAction: ActionArtifact, MutationRequested: MutationNone, Confidence: 0.9,
	}
	normalizeSemanticConsistency(TurnFeatures{}, &artifact)
	if artifact.MutationRequested != MutationExternal {
		t.Fatalf("artifact=%+v", artifact)
	}
	debug := SemanticIntent{
		SchemaVersion: SchemaVersion, Interaction: InteractionTask,
		RequestedAction: ActionDebug, Domain: "frontend", RecipientType: "assistant",
		MutationRequested: MutationWorkspace, Confidence: 0.9,
	}
	normalizeSemanticConsistency(TurnFeatures{}, &debug)
	if debug.RecipientType != "frontend" {
		t.Fatalf("debug=%+v", debug)
	}
}

// Note: workspace-inspect-cue, presence-check, and creative-write phrase overrides were
// removed from ResolvePolicy (stamp-first routing) — see internal/agent/semantic_stamp.go
// and the classifier's own reason codes for the replacement behavior. The corresponding
// looksLikeWorkspaceInspectRequest / looksLikeCreativeOrGeneralAnswerRequest text
// heuristics were deleted along with their tests.

func TestPolicyOpenCanvasArtifactPromotesEditToArtifact(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "lets update the canvas to be black and white",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
		OpenArtifactID:       "art-1",
		OpenArtifactRenderer: "nj.mermaid",
		OpenArtifactTitle:    "dickory-docs architecture",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.9,
		Retrieval:         []RetrievalTarget{RetrievalCodebase},
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact via open_canvas_artifact", decision.Action)
	}
	if decision.Mutation != MutationExternal {
		t.Fatalf("mutation=%s, want external", decision.Mutation)
	}
	if !containsRetrievalTarget(decision.Retrieval, RetrievalPriorReference) {
		t.Fatalf("retrieval=%v, want prior_reference", decision.Retrieval)
	}
	found := false
	for _, o := range decision.PolicyOverrides {
		if o == "open_canvas_artifact" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("overrides=%v, want open_canvas_artifact", decision.PolicyOverrides)
	}
}

func TestPolicyBlankCanvasReasonPromotesAskUserToArtifact(t *testing.T) {
	// Live failure: qwen2.5:3b tagged blank_canvas but stamped ask_user.
	decision := ResolvePolicy(TurnFeatures{
		Text:         "lets create a canvas for this",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionCasual,
		RequestedAction:   ActionAskUser,
		MutationRequested: MutationNone,
		Confidence:        1,
		ReasonCodes:       []string{"blank_canvas"},
		RecipientType:     "assistant",
		Retrieval:         []RetrievalTarget{RetrievalMemory},
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact via canvas_reason_artifact", decision.Action)
	}
	if decision.Mutation != MutationExternal {
		t.Fatalf("mutation=%s, want external", decision.Mutation)
	}
	if !containsString(decision.PolicyOverrides, "canvas_reason_artifact") {
		t.Fatalf("overrides=%v, want canvas_reason_artifact", decision.PolicyOverrides)
	}
	if decision.Interaction != InteractionTask {
		t.Fatalf("interaction=%s, want task after promote", decision.Interaction)
	}
}

func TestPolicyBlankCanvasReasonPromotesContinuePastWorkspaceRequired(t *testing.T) {
	// Live failure: continue + blank_canvas defaulted to workspace mutation, then
	// demoted to ask_user via workspace_required (no workspace attached).
	decision := ResolvePolicy(TurnFeatures{
		Text:            "ok please do that now",
		ComposerMode:    "agent",
		PendingActionID: "dc888200-25ae-4046-a045-8650f57630ce",
		PendingAction:   ActionAskUser,
	}, SemanticIntent{
		SchemaVersion:      SchemaVersion,
		Interaction:        InteractionContinuation,
		RequestedAction:    ActionContinue,
		MutationRequested:  MutationNone,
		Confidence:         0.95,
		ContinuationTarget: "dc888200-25ae-4046-a045-8650f57630ce",
		ReasonCodes:        []string{"blank_canvas"},
		RecipientType:      "frontend",
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact (not ask_user via workspace_required)", decision.Action)
	}
	if decision.Mutation != MutationExternal {
		t.Fatalf("mutation=%s, want external", decision.Mutation)
	}
	if containsString(decision.PolicyOverrides, "workspace_required") {
		t.Fatalf("overrides=%v, must not demote blank_canvas continue to workspace_required", decision.PolicyOverrides)
	}
	if decision.ContinuationTarget != "" {
		t.Fatalf("continuation_target=%q, want cleared after canvas promote", decision.ContinuationTarget)
	}
}

func TestPolicyBlankCanvasReasonDoesNotOverrideWorkspaceEdit(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "create a canvas after you fix the typo in main.go",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.9,
		ReasonCodes:       []string{"blank_canvas"},
		Retrieval:         []RetrievalTarget{RetrievalCodebase},
	}, SourceLocalModel)
	if decision.Action != ActionEdit {
		t.Fatalf("action=%s, mixed fix+canvas must stay edit", decision.Action)
	}
}

func TestPolicyMapsRouteReasonPromotesRunToArtifact(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "draw me a map from Swansea, IL to St. Louis MO",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionRun,
		MutationRequested: MutationExternal,
		Confidence:        1,
		ReasonCodes:       []string{"maps_route"},
		RecipientType:     "frontend",
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact via maps_route reason", decision.Action)
	}
	if !containsString(decision.PolicyOverrides, "canvas_reason_artifact") {
		t.Fatalf("overrides=%v, want canvas_reason_artifact", decision.PolicyOverrides)
	}
}

func TestPolicyBlankCanvasReasonDoesNotPromoteMeetingQuestion(t *testing.T) {
	// Live failure: after a today-notes reply, "When was the last meeting…" was
	// sprayed with blank_canvas/startup_failure and promoted into a blank canvas.
	decision := ResolvePolicy(TurnFeatures{
		Text:         "When was the last meeting that does have notes?",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAskUser,
		MutationRequested: MutationNone,
		Confidence:        0.95,
		ReasonCodes: []string{
			"startup_failure", "runtime_failure", "blank_canvas", "durable_artifact",
		},
		RecipientType: "assistant",
	}, SourceLocalModel)
	if decision.Action == ActionArtifact {
		t.Fatalf("meeting-notes question must not become artifact: %+v", decision)
	}
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer (not ask_user question prompt)", decision.Action)
	}
	if containsString(decision.PolicyOverrides, "canvas_reason_artifact") {
		t.Fatalf("overrides=%v must not include canvas_reason_artifact", decision.PolicyOverrides)
	}
}

func TestPolicySpuriousArtifactStampDemotesMeetingQuestion(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "When was the last meeting that does have notes?",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionArtifact,
		MutationRequested: MutationExternal,
		Confidence:        0.95,
		ReasonCodes:       []string{"blank_canvas", "durable_artifact"},
		RecipientType:     "assistant",
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer via spurious_artifact_demote", decision.Action)
	}
	if !containsString(decision.PolicyOverrides, "spurious_artifact_demote") {
		t.Fatalf("overrides=%v, want spurious_artifact_demote", decision.PolicyOverrides)
	}
}

func TestPolicyCanvasTextAskPromotesCreateWithThatInformation(t *testing.T) {
	// Live failure: "create a canvas with that information" stamped answer +
	// advisory_question (no blank_canvas) → ASCII fake canvas.
	decision := ResolvePolicy(TurnFeatures{
		Text:         "create a canvas with that information please",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionContinuation,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.95,
		ReasonCodes:       []string{"advisory_question"},
		Retrieval:         []RetrievalTarget{RetrievalPriorReference},
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact via canvas_text_artifact", decision.Action)
	}
	if !containsString(decision.PolicyOverrides, "canvas_text_artifact") {
		t.Fatalf("overrides=%v, want canvas_text_artifact", decision.PolicyOverrides)
	}
	if !containsRetrievalTarget(decision.Retrieval, RetrievalPriorReference) {
		t.Fatalf("retrieval=%v, want prior_reference", decision.Retrieval)
	}
}

func TestPolicyCanvasTextAskDoesNotPromoteStatusQuestion(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "what's on this canvas?",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer for canvas status/content question", decision.Action)
	}
}

func TestPolicyOpenCanvasDoesNotPromoteMeetingNotesSummary(t *testing.T) {
	// Live failure: focused Neural Canvas tab promoted "summarize today's meeting
	// notes" into an artifact update instead of a chat answer.
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "can you summarize todays meeting notes for me?",
		ComposerMode:         "agent",
		OpenArtifactID:       "541e3c624b6c00cba4eb83b9b50bf2f8",
		OpenArtifactRenderer: "nj.markdown",
		OpenArtifactTitle:    "St. Louis to Sea Side",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s overrides=%v, want answer for meeting-notes summary", decision.Action, decision.PolicyOverrides)
	}
	if containsString(decision.PolicyOverrides, "open_canvas_artifact") {
		t.Fatalf("overrides=%v must not include open_canvas_artifact", decision.PolicyOverrides)
	}
}

func TestPolicyOpenCanvasDoesNotPromoteChatSurfaceAsk(t *testing.T) {
	cases := []string{
		"please summarize my last meeting notes in the chat",
		"just show me text in the chat please",
	}
	for _, text := range cases {
		decision := ResolvePolicy(TurnFeatures{
			Text:                 text,
			ComposerMode:         "agent",
			OpenArtifactID:       "art-1",
			OpenArtifactRenderer: "nj.markdown",
			OpenArtifactTitle:    "St. Louis to Sea Side",
		}, SemanticIntent{
			SchemaVersion:     SchemaVersion,
			Interaction:       InteractionQuestion,
			RequestedAction:   ActionArtifact,
			MutationRequested: MutationExternal,
			Confidence:        0.9,
			ReasonCodes:       []string{"blank_canvas", "durable_artifact"},
		}, SourceLocalModel)
		if decision.Action != ActionAnswer {
			t.Fatalf("text=%q action=%s overrides=%v, want answer", text, decision.Action, decision.PolicyOverrides)
		}
		if !containsString(decision.PolicyOverrides, "chat_surface_demote") {
			t.Fatalf("text=%q overrides=%v, want chat_surface_demote", text, decision.PolicyOverrides)
		}
	}
}

func TestPolicyOpenCanvasStillPromotesPutMeetingNotesInThere(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "put the meeting notes in there",
		ComposerMode:         "agent",
		OpenArtifactID:       "art-1",
		OpenArtifactRenderer: "nj.markdown",
		OpenArtifactTitle:    "Plan",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s overrides=%v, want artifact for explicit canvas fill", decision.Action, decision.PolicyOverrides)
	}
	if !containsString(decision.PolicyOverrides, "open_canvas_artifact") {
		t.Fatalf("overrides=%v, want open_canvas_artifact", decision.PolicyOverrides)
	}
}

func TestPolicyOpenCanvasArtifactPromotesTaskAnswerToArtifact(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "add a list of places we are going to visit",
		ComposerMode:         "agent",
		OpenArtifactID:       "art-md-1",
		OpenArtifactRenderer: "nj.markdown",
		OpenArtifactTitle:    "Canvas",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact for open-canvas fill-in", decision.Action)
	}
	if decision.Mutation != MutationExternal {
		t.Fatalf("mutation=%s, want external", decision.Mutation)
	}
}

func TestPolicyOpenCanvasPromotesWeatherFillAsk(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "Can you get todays weather for St. Louis, MO and put it in the canvas please",
		ComposerMode:         "agent",
		OpenArtifactID:       "art-md-1",
		OpenArtifactRenderer: "nj.markdown",
		OpenArtifactTitle:    "Canvas",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0,
	}, SourceSafeFallback)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact for put-weather-in-canvas", decision.Action)
	}
}

func TestPolicyOpenCanvasPromotesListItemReviseAsk(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		action Action
	}{
		{name: "declarative_item", text: "the 3rd item is Arrive in Flordia", action: ActionAnswer},
		{name: "add_list_item", text: "ok add a 3rd list item, arrive in flordia", action: ActionAskUser},
		{name: "continue_add_item", text: "ok add a 3rd list item, arrive in flordia", action: ActionContinue},
		{name: "edit_item", text: "the 3rd item is Arrive in Flordia", action: ActionEdit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := ResolvePolicy(TurnFeatures{
				Text:                 tc.text,
				ComposerMode:         "agent",
				OpenArtifactID:       "73fdd68522d91b3669a679c0e4a58c74",
				OpenArtifactRenderer: "nj.markdown",
				OpenArtifactTitle:    "Trip Planning",
			}, SemanticIntent{
				SchemaVersion:     SchemaVersion,
				Interaction:       InteractionQuestion,
				RequestedAction:   tc.action,
				MutationRequested: MutationNone,
				Confidence:        0.9,
			}, SourceLocalModel)
			if decision.Action != ActionArtifact {
				t.Fatalf("action=%s overrides=%v, want artifact", decision.Action, decision.PolicyOverrides)
			}
			if decision.Mutation != MutationExternal {
				t.Fatalf("mutation=%s, want external", decision.Mutation)
			}
			if !containsString(decision.PolicyOverrides, "open_canvas_artifact") {
				t.Fatalf("overrides=%v, want open_canvas_artifact", decision.PolicyOverrides)
			}
			if !containsRetrievalTarget(decision.Retrieval, RetrievalPriorReference) {
				t.Fatalf("retrieval=%v, want prior_reference", decision.Retrieval)
			}
		})
	}
}

func TestPolicyOpenCanvasPromotesWithIDOnlyNoRenderer(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:           "ok add a 3rd list item, arrive in flordia",
		ComposerMode:   "agent",
		OpenArtifactID: "art-md-1",
		// Renderer intentionally empty — promote must still fire on open id.
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionAskUser,
		MutationRequested: MutationNone,
		Confidence:        0.85,
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s overrides=%v, want artifact with open id only", decision.Action, decision.PolicyOverrides)
	}
}

func TestLooksLikeOpenCanvasReviseAsk(t *testing.T) {
	if !LooksLikeOpenCanvasReviseAsk("the 3rd item is Arrive in Flordia") {
		t.Fatal("expected declarative item revise")
	}
	if !LooksLikeOpenCanvasReviseAsk("ok add a 3rd list item, arrive in flordia") {
		t.Fatal("expected add list item revise")
	}
	if LooksLikeOpenCanvasReviseAsk("fix the typo in main.go") {
		t.Fatal("workspace file edit must not match")
	}
	if LooksLikeOpenCanvasReviseAsk("did you update the canvas with the info?") {
		t.Fatal("status question must not match")
	}
}

func TestPolicyOpenCanvasDoesNotPromoteStatusQuestion(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "did you update the canvas with the info?",
		ComposerMode:         "agent",
		OpenArtifactID:       "art-md-1",
		OpenArtifactRenderer: "nj.markdown",
		OpenArtifactTitle:    "Canvas",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
		ReasonCodes:       []string{"canvas_meta_question"},
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer for canvas status question", decision.Action)
	}
}

func TestLooksLikeCanvasRenameAsk(t *testing.T) {
	cases := []string{
		`Lets name it "St. Louis to Florida"`,
		`The title of the document should be St. Louis to Florida`,
		`set the title to Trip Plan`,
		`change the title to Road Trip`,
		`rename the canvas to Packing List`,
	}
	for _, c := range cases {
		if !LooksLikeCanvasRenameAsk(c) {
			t.Fatalf("LooksLikeCanvasRenameAsk(%q)=false, want true", c)
		}
	}
	if LooksLikeCanvasRenameAsk("why did you name it weather forcast?") {
		t.Fatal("title questions must not count as rename asks")
	}
	if LooksLikeCanvasRenameAsk("add a packing list to the canvas") {
		t.Fatal("fill asks must not count as rename asks")
	}
}

func TestPolicyOpenCanvasDoesNotPromoteTitleQuestion(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "why did you name it weather forcast?",
		ComposerMode:         "agent",
		OpenArtifactID:       "art-md-1",
		OpenArtifactRenderer: "nj.markdown",
		OpenArtifactTitle:    "Weather Forecast",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
		ReasonCodes:       []string{"canvas_title_question"},
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer for canvas title question", decision.Action)
	}
	if !LooksLikeCanvasTitleQuestion("why did you name it weather forcast?") {
		t.Fatal("expected title question detector")
	}
}

func TestPolicyOpenCanvasDoesNotPromoteQuestionAnswer(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "what's on this canvas?",
		ComposerMode:         "agent",
		OpenArtifactID:       "art-md-1",
		OpenArtifactRenderer: "nj.markdown",
		OpenArtifactTitle:    "Canvas",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
		ReasonCodes:       []string{"canvas_meta_question"},
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer for canvas question", decision.Action)
	}
}

func TestPolicyOpenCanvasDoesNotOverridePendingEdit(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "ok sounds good",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
		PendingActionID:      "goal-1",
		PendingAction:        ActionEdit,
		OpenArtifactID:       "art-1",
		OpenArtifactRenderer: "nj.mermaid",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionContinuation,
		RequestedAction:   ActionContinue,
		MutationRequested: MutationWorkspace,
		Confidence:        1,
	}, SourceLocalModel)
	if decision.Action == ActionArtifact {
		t.Fatalf("pending edit continuation must not become artifact: %+v", decision)
	}
}

func TestLooksLikeAdvisoryImplementationQuestion_deprecatedStub(t *testing.T) {
	// Advisory meaning is owned by the classifier (reason_codes). The phrase helper is a no-op.
	if LooksLikeAdvisoryImplementationQuestion(
		"now outline the hook changes you'd make in hub.go for better errors",
	) {
		t.Fatal("deprecated LooksLikeAdvisoryImplementationQuestion must return false")
	}
	if LooksLikeAdvisoryImplementationQuestion("please implement hub.go error hooks now") {
		t.Fatal("deprecated LooksLikeAdvisoryImplementationQuestion must return false")
	}
}

func TestPolicyCanvasTextAskPromotesCreateNeuralCanvasNow(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "yes that works please create the neural canvas now",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.95,
		ReasonCodes:       []string{"advisory_question"},
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s overrides=%v, want artifact", decision.Action, decision.PolicyOverrides)
	}
}

func TestPolicyCanvasTypoCanvansStillPromotes(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "create the canvans with this information please",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.95,
		ReasonCodes:       []string{"startup_failure"},
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s overrides=%v, want artifact despite canvans typo", decision.Action, decision.PolicyOverrides)
	}
	if containsString(decision.PolicyOverrides, "workspace_required") {
		t.Fatalf("must not demote canvas create to workspace_required: %v", decision.PolicyOverrides)
	}
}

func TestPolicyCanvasTextAskPromotesDespitePendingEdit(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:            "yes that works please create the neural canvas now",
		ComposerMode:    "agent",
		PendingActionID: "goal-edit",
		PendingAction:   ActionEdit,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.95,
		ReasonCodes:       []string{"advisory_question"},
	}, SourceLocalModel)
	if decision.Action != ActionArtifact {
		t.Fatalf("action=%s, canvas create must not be blocked by pending edit", decision.Action)
	}
}

func TestPrefersChatOverOpenCanvas(t *testing.T) {
	if !PrefersChatOverOpenCanvas("can you summarize todays meeting notes for me?") {
		t.Fatal("expected meeting-notes summary to prefer chat")
	}
	if !PrefersChatOverOpenCanvas("please summarize my last meeting notes in the chat") {
		t.Fatal("expected in-the-chat meeting summary to prefer chat")
	}
	if !PrefersChatOverOpenCanvas("just show me text in the chat please") {
		t.Fatal("expected explicit chat-surface ask")
	}
	if PrefersChatOverOpenCanvas("put the meeting notes in there") {
		t.Fatal("explicit canvas fill must still revise the open page")
	}
	if PrefersChatOverOpenCanvas("add weather to the canvas") {
		t.Fatal("canvas fill must not prefer chat")
	}
}

func TestLooksLikeProjectOverviewAsk(t *testing.T) {
	if !LooksLikeProjectOverviewAsk("review and summerize the project I have open please") {
		t.Fatal("expected typo summerize + project open")
	}
	if !LooksLikeProjectOverviewAsk("ok can you summarize the project") {
		t.Fatal("expected summarize the project")
	}
	if LooksLikeProjectOverviewAsk("create a canvas with a project summary") {
		t.Fatal("canvas create should not count as chat overview")
	}
	if LooksLikeWorkspaceReportAsk("summarize the project") {
		t.Fatal("chat-only summarize must not be workspace_report")
	}
	if !LooksLikeWorkspaceReportAsk("write a project summary report on a canvas") {
		t.Fatal("canvas report should still match workspace_report")
	}
}

func TestLooksLikeWorkspaceFixAsk(t *testing.T) {
	if !LooksLikeWorkspaceFixAsk("fix the app") {
		t.Fatal("expected fix the app")
	}
	if !LooksLikeWorkspaceFixAsk("I am asking you to fix the app, can you?") {
		t.Fatal("expected imperative fix ask")
	}
	if !LooksLikeWorkspaceFixAsk("the app is not booting can you fix it?") {
		t.Fatal("expected boot failure fix")
	}
	if LooksLikeWorkspaceFixAsk("how would you fix a blank screen?") {
		t.Fatal("advisory how-would should not match")
	}
	if LooksLikeWorkspaceFixAsk("ok that sounds like a good plan") {
		t.Fatal("plan affirmation should not match")
	}
}

func TestPolicyWorkspaceFixPromotesPlanStamp(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "fix the app",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionPlan,
		MutationRequested: MutationNone,
		Confidence:        0.9,
		ReasonCodes:       []string{"runtime_failure"},
	}, SourceLocalModel)
	if decision.Action != ActionDebug {
		t.Fatalf("action=%s overrides=%v, want debug", decision.Action, decision.PolicyOverrides)
	}
	if decision.Mutation != MutationWorkspace {
		t.Fatalf("mutation=%s, want workspace", decision.Mutation)
	}
	if !containsString(decision.PolicyOverrides, "workspace_fix_promote") {
		t.Fatalf("expected workspace_fix_promote override, got %v", decision.PolicyOverrides)
	}
}

func TestPolicyComposerPlanStillForbidsWorkspaceFixMutation(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "fix the app",
		ComposerMode:         "plan",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionDebug,
		MutationRequested: MutationWorkspace,
		Confidence:        0.95,
		ReasonCodes:       []string{"runtime_failure"},
	}, SourceLocalModel)
	if decision.Action != ActionPlan {
		t.Fatalf("action=%s, explicit plan mode must win", decision.Action)
	}
	if decision.Mutation != MutationNone {
		t.Fatalf("mutation=%s, plan mode must forbid mutation", decision.Mutation)
	}
	if !containsRetrievalTarget(decision.Retrieval, RetrievalCodebase) {
		t.Fatalf("plan mode with workspace should retrieve codebase, got %v", decision.Retrieval)
	}
}

func TestPolicyProjectOverviewDemotesRunToInspect(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "review and summerize the project I have open please",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionContinuation,
		RequestedAction:   ActionRun,
		MutationRequested: MutationExternal,
		Confidence:        0.92,
		ReasonCodes:       []string{"project_overview", "explicit_continuation"},
	}, SourceLocalModel)
	if decision.Action != ActionInspect || decision.Mutation != MutationNone {
		t.Fatalf("decision=%+v, want inspect/none", decision)
	}
	if !containsString(decision.PolicyOverrides, "project_overview_inspect") {
		t.Fatalf("overrides=%v, want project_overview_inspect", decision.PolicyOverrides)
	}
	if !containsRetrievalTarget(decision.Retrieval, RetrievalCodebase) {
		t.Fatalf("retrieval=%v, want codebase", decision.Retrieval)
	}
}

func TestPolicyProjectOverviewDoesNotStealShellRun(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "run the tests then summarize the project",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionRun,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionRun {
		t.Fatalf("action=%s, explicit test run must stay run", decision.Action)
	}
}

func TestPolicyContinueWithoutTargetBecomesAnswer(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "and what about the second option?",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:      SchemaVersion,
		Interaction:        InteractionContinuation,
		RequestedAction:    ActionContinue,
		MutationRequested:  MutationNone,
		ContinuationTarget: "invented-target",
		Confidence:         0.9,
	}, SourceLocalModel)
	if decision.Action != ActionAnswer || decision.Mutation != MutationNone {
		t.Fatalf("decision=%+v, want answer/none for soft follow-up", decision)
	}
	if !containsString(decision.PolicyOverrides, "continuation_target_missing") {
		t.Fatalf("overrides=%v, want continuation_target_missing", decision.PolicyOverrides)
	}
}

func TestPolicyInspectWithoutWorkspaceBecomesAnswer(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "When was the last meeting that does have notes?",
		ComposerMode: "agent",
		HasWorkspace: false,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionInspect,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer without workspace", decision.Action)
	}
	if !containsString(decision.PolicyOverrides, "inspect_requires_workspace") {
		t.Fatalf("overrides=%v, want inspect_requires_workspace", decision.PolicyOverrides)
	}
}

func TestPolicySpuriousAskUserDemoteOnClosure(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "thanks, that's all for now",
		ComposerMode: "agent",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionClosure,
		RequestedAction:   ActionAskUser,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer", decision.Action)
	}
	if !containsString(decision.PolicyOverrides, "spurious_ask_user_demote") {
		t.Fatalf("overrides=%v, want spurious_ask_user_demote", decision.PolicyOverrides)
	}
}

func TestPolicyRepoFactPromotesInspect(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "Which package declares the Express dependency in this repo?",
		ComposerMode: "agent",
		HasWorkspace: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
		ReasonCodes:       []string{"repo_fact"},
	}, SourceLocalModel)
	if decision.Action != ActionInspect || decision.Mutation != MutationNone {
		t.Fatalf("decision=%+v, want inspect/none", decision)
	}
	if !containsString(decision.PolicyOverrides, "repo_fact_inspect") {
		t.Fatalf("overrides=%v, want repo_fact_inspect", decision.PolicyOverrides)
	}
	if !containsRetrievalTarget(decision.Retrieval, RetrievalCodebase) {
		t.Fatalf("retrieval=%v, want codebase", decision.Retrieval)
	}
}

func TestPolicyAskUserWithAmbiguityPreserved(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "which file should I edit?",
		ComposerMode: "agent",
		HasWorkspace: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAskUser,
		MutationRequested: MutationNone,
		Confidence:        0.9,
		Ambiguities:       []string{"target_file"},
	}, SourceLocalModel)
	if decision.Action != ActionAskUser {
		t.Fatalf("action=%s, named ambiguity must keep ask_user", decision.Action)
	}
}

func TestLooksLikeRepoFactAsk(t *testing.T) {
	if !LooksLikeRepoFactAsk("Which package declares the Express dependency in this repo?") {
		t.Fatal("expected repo fact ask")
	}
	if LooksLikeRepoFactAsk("how are you today?") {
		t.Fatal("ordinary chat is not a repo fact ask")
	}
}

func TestPolicyOpenCanvasRenameNotMetaDemoted(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "rename the title to Trip Plan",
		ComposerMode:         "agent",
		OpenArtifactID:       "art-1",
		OpenArtifactRenderer: "markdown",
		OpenArtifactTitle:    "Plan",
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
		ReasonCodes:       []string{"canvas_title_question"},
	}, SourceLocalModel)
	if decision.Action != ActionArtifact || decision.Mutation != MutationExternal {
		t.Fatalf("decision=%+v, rename must stay artifact", decision)
	}
	if containsString(decision.PolicyOverrides, "open_canvas_meta_demote") {
		t.Fatalf("overrides=%v, rename must not meta-demote", decision.PolicyOverrides)
	}
	if !containsString(decision.PolicyOverrides, "open_canvas_artifact") {
		t.Fatalf("overrides=%v, want open_canvas_artifact", decision.PolicyOverrides)
	}
}

func TestPolicyImplementationPlanPromotesEdit(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "Add a ThemeToggle component to the React app.",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionPlan,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionEdit || decision.Mutation != MutationWorkspace {
		t.Fatalf("decision=%+v, want edit/workspace", decision)
	}
	if !containsString(decision.PolicyOverrides, "implementation_plan_promote") {
		t.Fatalf("overrides=%v, want implementation_plan_promote", decision.PolicyOverrides)
	}
}
