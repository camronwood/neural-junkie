package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserRequestsMapOrRoute(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"can you draw me a map from Swansea, IL to St. Louis MO", true},
		{"can you genereat a canvas map for Swansea, IL to St. Louis, MO?", true},
		{"driving directions from Midway to Navy Pier", true},
		{"walk from Millennium Park to the Art Institute", true},
		{"please map from Swansea, IL to St. Louis, MO", true},
		{"draw me a sunset over the ocean", false},
		{"generate an image of a cat", false},
		{"what time is it?", false},
	}
	for _, tc := range cases {
		if got := UserRequestsMapOrRoute(tc.in); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestDeriveTurnGoal_MapRouteIsArtifact(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "MapsExpert", Type: protocol.AgentTypeMaps}}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-maps",
		protocol.AgentInfo{ID: "u", Name: "Camron"},
		"please map from Swansea, IL to St. Louis, MO",
	)
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact {
		t.Fatalf("action=%q want artifact", goal.Action)
	}
	if len(goal.ExpectedEvidence) != 1 || goal.ExpectedEvidence[0] != EvidenceArtifactCreated {
		t.Fatalf("evidence=%v", goal.ExpectedEvidence)
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

func TestUserRequestsGeneratedImage_ExcludesMaps(t *testing.T) {
	if UserRequestsGeneratedImage("can you draw me a map from Swansea, IL to St. Louis MO") {
		t.Fatal("map request must not trigger image generation")
	}
	if !UserRequestsGeneratedImage("draw me a logo for my startup") {
		t.Fatal("logo request should still trigger image generation")
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
	if UserRequestsMapOrRoute("Swansea, Saint Clair County to Saint Louis, Missouri") != true {
		t.Fatal("bare geocode labels should count as map/route request")
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
