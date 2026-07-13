package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

// skipKnowledgeRetrievalForMessage avoids running memory/codebase retrieval against
// collaboration orchestration prompts (they contain boilerplate like "collabs/" and
// "collaboration" that would otherwise trigger slow, irrelevant knowledge routes).
func skipKnowledgeRetrievalForMessage(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	switch msg.Type {
	case protocol.MessageTypeCollabDiscussion, protocol.MessageTypeCollabTask, protocol.MessageTypeCollabRecap:
		return true
	}
	if msg.GetCollaborationID() != "" && msg.IsFromSystem() {
		return true
	}
	if msg.Metadata != nil {
		if internal, ok := msg.Metadata["collab_internal_event"].(bool); ok && internal {
			return true
		}
	}
	return false
}

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

func (a *Agent) effectiveKnowledgePlan(msg *protocol.Message, intent TurnIntent) routing.KnowledgePlan {
	if a == nil {
		return routing.KnowledgePlan{}
	}
	if skipKnowledgeRetrievalForMessage(msg) {
		return routing.KnowledgePlan{Reason: "collab_turn"}
	}
	snap := a.LastRoutingSnapshot()
	if len(snap.KnowledgeTargets) > 0 || snap.KnowledgeReason != "" {
		return knowledgePlanFromSnapshot(snap)
	}
	if msg != nil {
		skipDefault := intent == IntentClosure || intent == IntentLowSignal
		return routing.PlanKnowledgeRouteForTurn(msg.Content, skipDefault)
	}
	return routing.KnowledgePlan{}
}

// effectiveKnowledgePlanFromMessage classifies intent when no turn pipeline state exists.
func (a *Agent) effectiveKnowledgePlanFromMessage(msg *protocol.Message) routing.KnowledgePlan {
	intent := IntentSubstantive
	if a != nil && msg != nil && a.Hub != nil {
		intent = a.classifyTurnIntentForMessage(msg)
	}
	return a.effectiveKnowledgePlan(msg, intent)
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

func (a *Agent) applyKnowledgePlanEarly(msg *protocol.Message, intent TurnIntent) {
	if a == nil || msg == nil || skipKnowledgeRetrievalForMessage(msg) {
		return
	}
	plan := a.effectiveKnowledgePlan(msg, intent)
	a.ExecuteKnowledgePlan(context.Background(), msg, plan, intent, KnowledgePhaseEarly)
}
