package protocol

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
)

func TestTurnDecisionMetadataRoundTrip(t *testing.T) {
	msg := &Message{}
	want := intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionTask,
		RequestedAction: intent.ActionDebug,
		Action:          intent.ActionDebug,
		Retrieval:       []intent.RetrievalTarget{intent.RetrievalCodebase},
		Mutation:        intent.MutationNone,
		Confidence:      0.9,
		Source:          intent.SourceLocalModel,
	}
	if err := StampTurnDecision(msg, want); err != nil {
		t.Fatal(err)
	}
	got, ok := ExtractTurnDecision(msg)
	if !ok {
		t.Fatal("decision not extracted")
	}
	if got.Action != want.Action || got.Interaction != want.Interaction || got.Retrieval[0] != want.Retrieval[0] {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}

func TestInvalidTurnDecisionMetadataRejected(t *testing.T) {
	msg := &Message{Metadata: map[string]interface{}{
		MetadataTurnDecision: map[string]interface{}{
			"schema_version":   intent.SchemaVersion,
			"interaction":      "task",
			"requested_action": "destroy",
			"action":           "destroy",
			"mutation":         "workspace",
			"confidence":       1,
			"source":           "local_model",
		},
	}}
	if _, ok := ExtractTurnDecision(msg); ok {
		t.Fatal("invalid decision accepted")
	}
}

func TestTurnGovernanceOverridesContradictoryLegacyMetadata(t *testing.T) {
	msg := &Message{Metadata: map[string]interface{}{
		TurnMetaComposerMode:      "agent",
		TurnMetaCanProposeFiles:   true,
		TurnMetaCanRunImplSession: true,
	}}
	StampTurnGovernance(msg, TurnGovernance{
		ComposerMode: "ask", ContextTier: "full",
		CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: true,
		Provenance: "server_canonical",
	})
	caps := ResolveTurnCapabilities(msg)
	if caps.ComposerMode != "ask" || caps.CanProposeFiles || caps.CanRunImplSession || caps.RequiresWorkspace {
		t.Fatalf("unsafe governance resolution: %+v", caps)
	}
}
