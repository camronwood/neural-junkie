package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Note: UserRequestsMapOrRoute is a deprecated phrase-matching stub (always false) — see // phrase-migration-shim
// maps_shortcut.go. Maps routing is now stamp-first via messageStampedMapsRoute /
// tryMapsRouteShortcut, exercised below with a stamped TurnDecision that carries the
// "maps_route" reason code.

func TestDeriveTurnGoal_MapRouteIsArtifact(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "MapsExpert", Type: protocol.AgentTypeMaps}}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-maps",
		protocol.AgentInfo{ID: "u", Name: "Camron"},
		"please map from Swansea, IL to St. Louis, MO",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 0.95, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"maps_route"},
	}); err != nil {
		t.Fatal(err)
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact {
		t.Fatalf("action=%q want artifact", goal.Action)
	}
	if len(goal.ExpectedEvidence) != 1 || goal.ExpectedEvidence[0] != EvidenceArtifactCreated {
		t.Fatalf("evidence=%v", goal.ExpectedEvidence)
	}
	if len(goal.RequiredCapabilities) != 1 || goal.RequiredCapabilities[0] != mapsCreateToolName {
		t.Fatalf("required capabilities=%v want [%s]", goal.RequiredCapabilities, mapsCreateToolName)
	}
}

func TestShouldRewriteAsSafeFailureForGoal_KeepsMapArtifact(t *testing.T) {
	goal := TurnGoal{Action: ActionRun, ExpectedEvidence: []EvidenceKind{EvidenceCommandRun}}
	ledger := &ActionEvidenceLedger{}
	ledger.Record(ActionEvidence{Kind: EvidenceArtifactCreated, Tool: mapsCreateToolName, Status: "succeeded"})
	resp := "Posted an interactive Neural Canvas map for Swansea to Saint Louis (driving)."
	issues := validateResponseAgainstEvidence(goal, ledger, &protocol.Message{Content: "please map from Swansea, IL to St. Louis, MO"}, resp, nil)
	if shouldRewriteAsSafeFailureForGoal(goal, ledger, issues, resp) {
		t.Fatalf("should keep map success reply; issues=%v", issues)
	}
}

func TestParseMapEndpoints(t *testing.T) {
	from, to, ok := ParseMapEndpoints("can you draw me a map from Swansea, IL to St. Louis MO")
	if !ok {
		t.Fatal("expected endpoints")
	}
	if from != "Swansea, IL" || !containsFold(to, "St. Louis") {
		t.Fatalf("got from=%q to=%q", from, to)
	}
	from, to, ok = ParseMapEndpoints("can you genereat a canvas map for Swansea, IL to St. Louis, MO?")
	if !ok || from != "Swansea, IL" || !containsFold(to, "St. Louis") {
		t.Fatalf("canvas map parse failed: ok=%v from=%q to=%q", ok, from, to)
	}
	from, to, ok = ParseMapEndpoints("Swansea, Saint Clair County to Saint Louis, Missouri")
	if !ok || !containsFold(from, "Swansea") || !containsFold(to, "Saint Louis") {
		t.Fatalf("bare place→place parse failed: ok=%v from=%q to=%q", ok, from, to)
	}
	if _, _, ok := ParseMapEndpoints("I need to restart the server"); ok {
		t.Fatal("need to … must not parse as places")
	}
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if equalFoldASCII(s[i:i+len(sub)], sub) {
					return true
				}
			}
			return false
		})()))
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
