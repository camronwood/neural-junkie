package agent

import (
	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func routeTargetsToStrings(targets []routing.RouteTarget) []string {
	if len(targets) == 0 {
		return nil
	}
	out := make([]string, len(targets))
	for i, t := range targets {
		out[i] = string(t)
	}
	return out
}

func knowledgePlanFromSnapshot(snap RoutingSnapshot) routing.KnowledgePlan {
	plan := routing.KnowledgePlan{Reason: snap.KnowledgeReason, Cues: map[routing.RouteTarget]string{}}
	for _, raw := range snap.KnowledgeTargets {
		t := routing.RouteTarget(raw)
		plan.Targets = append(plan.Targets, t)
		plan.Cues[t] = snap.KnowledgeReason
	}
	return plan
}

func (a *Agent) effectiveKnowledgePlan(msg *protocol.Message) routing.KnowledgePlan {
	if a == nil {
		return routing.KnowledgePlan{}
	}
	snap := a.LastRoutingSnapshot()
	if len(snap.KnowledgeTargets) > 0 || snap.KnowledgeReason != "" {
		return knowledgePlanFromSnapshot(snap)
	}
	if msg != nil {
		return routing.PlanKnowledgeRoute(msg.Content)
	}
	return routing.KnowledgePlan{}
}

// ShouldInjectMemory reports whether conversation memory retrieval should run.
func ShouldInjectMemory(plan routing.KnowledgePlan) bool {
	if len(plan.Targets) == 0 {
		return false
	}
	return plan.Has(routing.RouteConversationMemory) || plan.Has(routing.RouteCollabArtifact)
}

// ShouldRunPriorReference reports whether prior-reference history scan should run.
func ShouldRunPriorReference(plan routing.KnowledgePlan) bool {
	return plan.Has(routing.RoutePriorReference)
}

// ShouldRunCodebaseSearch reports whether codeindex retrieval should run.
func ShouldRunCodebaseSearch(plan routing.KnowledgePlan) bool {
	return plan.Has(routing.RouteCodebase)
}

// MemorySourceFilter returns source-type scope for memory search.
func MemorySourceFilter(plan routing.KnowledgePlan) []memory.SourceType {
	hasMem := plan.Has(routing.RouteConversationMemory)
	hasCollab := plan.Has(routing.RouteCollabArtifact)
	if hasMem && hasCollab {
		return nil
	}
	if hasCollab && !hasMem {
		return []memory.SourceType{memory.SourceCollabArtifact}
	}
	if hasMem {
		return []memory.SourceType{memory.SourceMessage}
	}
	return nil
}

func (a *Agent) recordKnowledgeExecuted(path string) {
	if a == nil || path == "" {
		return
	}
	a.routingSnap.mu.Lock()
	defer a.routingSnap.mu.Unlock()
	for _, p := range a.routingSnap.snap.KnowledgeExecuted {
		if p == path {
			return
		}
	}
	a.routingSnap.snap.KnowledgeExecuted = append(a.routingSnap.snap.KnowledgeExecuted, path)
}

func knowledgeExecutedPathForMemory(plan routing.KnowledgePlan) string {
	switch {
	case plan.Has(routing.RouteConversationMemory) && plan.Has(routing.RouteCollabArtifact):
		return "memory"
	case plan.Has(routing.RouteCollabArtifact):
		return "collab_artifact"
	case plan.Has(routing.RouteConversationMemory):
		return "conversation_memory"
	default:
		return "memory"
	}
}

func (a *Agent) applyKnowledgePlanEarly(msg *protocol.Message) {
	if a == nil || msg == nil {
		return
	}
	plan := a.effectiveKnowledgePlan(msg)
	if ShouldRunCodebaseSearch(plan) {
		if MergeCodebaseForRoute(msg, plan) {
			a.recordKnowledgeExecuted("codebase")
		}
	}
}
