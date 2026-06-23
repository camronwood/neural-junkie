package routing

import "testing"

func TestClassifyKnowledgeRoute(t *testing.T) {
	cases := []struct {
		in     string
		target RouteTarget
	}{
		{"thanks!", RouteGeneral},
		{"search @codebase for main", RouteCodebase},
		{"what did we decide in the collab?", RouteCollabArtifact},
		{"remember when we talked about auth?", RouteConversationMemory},
		{"what did you say earlier about that message?", RoutePriorReference},
		{"hello team", RouteGeneral},
	}
	for _, tc := range cases {
		got := ClassifyKnowledgeRoute(tc.in)
		if got.Target != tc.target {
			t.Fatalf("Classify(%q) = %s, want %s", tc.in, got.Target, tc.target)
		}
	}
}
