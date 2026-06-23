package routing

import (
	"strings"
)

// RouteTarget is a knowledge retrieval mode.
type RouteTarget string

const (
	RouteGeneral            RouteTarget = "general"
	RouteConversationMemory RouteTarget = "conversation_memory"
	RouteCodebase           RouteTarget = "codebase"
	RouteCollabArtifact     RouteTarget = "collab_artifact"
	RoutePriorReference     RouteTarget = "prior_reference"
)

// Decision captures routing output for trace/debug.
type Decision struct {
	Target RouteTarget `json:"target"`
	Reason string      `json:"reason"`
}

// ClassifyKnowledgeRoute picks retrieval strategy from user text (rules-based MVP).
func ClassifyKnowledgeRoute(text string) Decision {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return Decision{RouteGeneral, "empty"}
	}
	if isClosure(lower) {
		return Decision{RouteGeneral, "closure_phrase"}
	}
	if strings.Contains(lower, "@codebase") || strings.Contains(lower, "in the repo") || strings.Contains(lower, "in this codebase") {
		return Decision{RouteCodebase, "codebase_cue"}
	}
	if strings.Contains(lower, "collab") || strings.Contains(lower, "collaboration") || strings.Contains(lower, "collabs/") {
		return Decision{RouteCollabArtifact, "collab_cue"}
	}
	if strings.Contains(lower, "that message") || strings.Contains(lower, "you said") {
		return Decision{RoutePriorReference, "prior_reference_cue"}
	}
	if strings.Contains(lower, "earlier") || strings.Contains(lower, "remember when") || strings.Contains(lower, "last time") {
		return Decision{RouteConversationMemory, "memory_cue"}
	}
	return Decision{RouteGeneral, "default"}
}

func isClosure(lower string) bool {
	phrases := []string{"thanks", "thank you", "done", "that's all", "thats all", "bye", "goodbye"}
	for _, p := range phrases {
		if lower == p || strings.HasPrefix(lower, p+" ") {
			return true
		}
	}
	return false
}
