package routing

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/intent"
)

// RouteTarget is a knowledge retrieval mode.
type RouteTarget string

const (
	RouteGeneral            RouteTarget = "general"
	RouteConversationMemory RouteTarget = "conversation_memory"
	RouteCodebase           RouteTarget = "codebase"
	RouteCodeGraph          RouteTarget = "code_graph"
	RouteCollabArtifact     RouteTarget = "collab_artifact"
	RoutePriorReference     RouteTarget = "prior_reference"
	RouteLearning           RouteTarget = "learning"
)

// Decision captures routing output for trace/debug (primary target only).
type Decision struct {
	Target RouteTarget `json:"target"`
	Reason string      `json:"reason"`
}

// KnowledgePlan is a composite retrieval plan for a user turn.
type KnowledgePlan struct {
	Targets []RouteTarget          `json:"targets"`
	Reason  string                 `json:"reason"`
	Cues    map[RouteTarget]string `json:"cues,omitempty"`
}

var codebaseSymbolRE = regexp.MustCompile(`[A-Z][a-zA-Z0-9]{3,}`)

// PlanKnowledgeRoute picks one or more retrieval strategies from user text.
func PlanKnowledgeRoute(text string) KnowledgePlan {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return KnowledgePlan{Reason: "empty"}
	}
	if isClosure(lower) {
		return KnowledgePlan{Reason: "closure_phrase"}
	}

	plan := KnowledgePlan{Cues: map[RouteTarget]string{}}
	add := func(t RouteTarget, reason string) {
		for _, existing := range plan.Targets {
			if existing == t {
				return
			}
		}
		plan.Targets = append(plan.Targets, t)
		plan.Cues[t] = reason
	}

	if matchesConversationMemoryCue(lower) {
		add(RouteConversationMemory, "memory_cue")
	}
	if matchesCodebaseCue(lower, text) {
		add(RouteCodebase, "codebase_cue")
	}
	if matchesCodeGraphCue(lower) {
		add(RouteCodeGraph, "code_graph_cue")
		// Graph questions usually also benefit from codebase chunks.
		add(RouteCodebase, "code_graph_implies_codebase")
	}
	if matchesCollabArtifactCue(lower) {
		add(RouteCollabArtifact, "collab_cue")
	}
	if matchesPriorReferenceCue(lower) {
		add(RoutePriorReference, "prior_reference_cue")
	}

	if len(plan.Targets) == 0 {
		// Substantive default: conversation memory for ordinary chat recall.
		plan.Targets = []RouteTarget{RouteConversationMemory}
		plan.Cues[RouteConversationMemory] = "default"
		plan.Reason = "default"
		return plan
	}
	if len(plan.Targets) == 1 {
		plan.Reason = plan.Cues[plan.Targets[0]]
		return plan
	}
	plan.Reason = "mixed"
	return plan
}

// PlanKnowledgeRouteForTurn plans retrieval; skipDefaultMemory avoids vector search on casual turns.
func PlanKnowledgeRouteForTurn(text string, skipDefaultMemory bool) KnowledgePlan {
	plan := PlanKnowledgeRoute(text)
	if skipDefaultMemory && plan.Reason == "default" && len(plan.Targets) == 1 && plan.Targets[0] == RouteConversationMemory {
		return KnowledgePlan{Reason: "low_signal_skip"}
	}
	return plan
}

// PlanKnowledgeRouteForDecision compiles the canonical semantic retrieval needs
// into the deterministic execution plan. It does not reinterpret user text.
func PlanKnowledgeRouteForDecision(decision intent.TurnDecision) KnowledgePlan {
	plan := KnowledgePlan{Cues: map[RouteTarget]string{}}
	for _, requested := range decision.Retrieval {
		var target RouteTarget
		switch requested {
		case intent.RetrievalMemory:
			target = RouteConversationMemory
		case intent.RetrievalCodebase:
			target = RouteCodebase
		case intent.RetrievalCodeGraph:
			target = RouteCodeGraph
		case intent.RetrievalPriorReference:
			target = RoutePriorReference
		case intent.RetrievalCollaboration:
			target = RouteCollabArtifact
		default:
			continue
		}
		if !plan.Has(target) {
			plan.Targets = append(plan.Targets, target)
			plan.Cues[target] = "semantic_decision"
		}
	}
	if len(plan.Targets) == 0 {
		switch decision.Interaction {
		case intent.InteractionClosure, intent.InteractionCasual:
			plan.Reason = "semantic_no_retrieval"
			return plan
		default:
			plan.Targets = []RouteTarget{RouteConversationMemory}
			plan.Cues[RouteConversationMemory] = "semantic_default"
		}
	}
	plan.Reason = "semantic_decision"
	return plan
}

func ClassifyKnowledgeRoute(text string) Decision {
	plan := PlanKnowledgeRoute(text)
	return Decision{Target: plan.Primary(), Reason: plan.Reason}
}

// Primary returns the first planned target or general when none.
func (p KnowledgePlan) Primary() RouteTarget {
	if len(p.Targets) == 0 {
		return RouteGeneral
	}
	return p.Targets[0]
}

// Has reports whether the plan includes a target.
func (p KnowledgePlan) Has(t RouteTarget) bool {
	for _, target := range p.Targets {
		if target == t {
			return true
		}
	}
	return false
}

func matchesConversationMemoryCue(lower string) bool {
	cues := []string{
		"remember when", "last time",
		"what did we decide", "we decided", "decided about", "talked about",
		"earlier in", "earlier when", "earlier we", "earlier today",
	}
	for _, c := range cues {
		if strings.Contains(lower, c) {
			return true
		}
	}
	return false
}

func matchesCodebaseCue(lower, raw string) bool {
	if strings.Contains(lower, "@codebase") ||
		strings.Contains(lower, "in the repo") ||
		strings.Contains(lower, "in this codebase") ||
		strings.Contains(lower, "in the code") ||
		strings.Contains(lower, "this code") ||
		strings.Contains(lower, "my code") ||
		strings.Contains(lower, "where is it in the repo") ||
		strings.Contains(lower, "where is that in the repo") {
		return true
	}
	codeFailureCues := []string{
		"app will not boot", "app won't boot", "app is not booting",
		"cannot boot", "can't boot", "fails to boot",
		"build is failing", "build fails", "compile error", "runtime error",
		"blank screen", "white screen",
	}
	for _, cue := range codeFailureCues {
		if strings.Contains(lower, cue) {
			return true
		}
	}
	locationCue := strings.Contains(lower, "where is it") ||
		strings.Contains(lower, "where is that") ||
		strings.Contains(lower, "find in the repo") ||
		strings.Contains(lower, "in the repo")
	if locationCue && codebaseSymbolRE.MatchString(raw) {
		return true
	}
	return false
}

func matchesCodeGraphCue(lower string) bool {
	cues := []string{
		"how does", "how do", "relate to", "related to",
		"path between", "who calls", "who imports", "what imports",
		"depends on", "dependency on", "call graph", "knowledge graph",
		"connected to", "imports from", "used by", "where is it used",
	}
	for _, c := range cues {
		if strings.Contains(lower, c) {
			return true
		}
	}
	return false
}

func matchesCollabArtifactCue(lower string) bool {
	return strings.Contains(lower, "collab") ||
		strings.Contains(lower, "collaboration") ||
		strings.Contains(lower, "collabs/")
}

func matchesPriorReferenceCue(lower string) bool {
	cues := []string{
		"that message", "you said", "what you wrote", "you wrote",
		"what you said", "few messages back", "earlier you",
		"previous reply", "last reply", "messages back",
	}
	for _, c := range cues {
		if strings.Contains(lower, c) {
			return true
		}
	}
	return false
}

func isClosure(lower string) bool {
	// Strip trailing punctuation so "thanks!" matches closure.
	trimmed := strings.TrimRight(lower, "!.?")
	phrases := []string{"thanks", "thank you", "done", "that's all", "thats all", "bye", "goodbye"}
	for _, p := range phrases {
		if trimmed == p || strings.HasPrefix(trimmed, p+" ") {
			return true
		}
	}
	return false
}
