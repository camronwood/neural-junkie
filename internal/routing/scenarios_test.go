package routing_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/camronwood/neural-junkie/internal/routing"
)

func TestKnowledgeRouterArchetypes(t *testing.T) {
	path := filepath.Join("..", "..", "scenarios", "routing", "archetypes.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read scenarios: %v", err)
	}
	var cases []struct {
		Name          string   `json:"name"`
		Input         string   `json:"input"`
		Expect        string   `json:"expect"`
		ExpectTargets []string `json:"expect_targets"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse scenarios: %v", err)
	}
	for _, tc := range cases {
		got := routing.ClassifyKnowledgeRoute(tc.Input)
		if string(got.Target) != tc.Expect {
			t.Fatalf("%s: Classify(%q) = %s, want %s", tc.Name, tc.Input, got.Target, tc.Expect)
		}
		plan := routing.PlanKnowledgeRoute(tc.Input)
		if tc.ExpectTargets != nil {
			gotTargets := make([]string, len(plan.Targets))
			for i, target := range plan.Targets {
				gotTargets[i] = string(target)
			}
			if !reflect.DeepEqual(gotTargets, tc.ExpectTargets) {
				t.Fatalf("%s: Plan(%q) targets = %v, want %v", tc.Name, tc.Input, gotTargets, tc.ExpectTargets)
			}
		}
	}
}
