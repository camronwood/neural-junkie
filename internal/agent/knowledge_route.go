package agent

import (
	"log"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func (a *Agent) recordKnowledgeRoute(msg *protocol.Message, intent TurnIntent) {
	if a == nil || msg == nil {
		return
	}
	if skipKnowledgeRetrievalForMessage(msg) {
		a.RecordRoutingSnapshotFor(msg.ID, RoutingSnapshot{
			KnowledgeRoute:  "collab_turn",
			KnowledgeReason: "collab_turn",
		})
		return
	}
	var plan routing.KnowledgePlan
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		plan = routing.PlanKnowledgeRouteForDecision(decision)
	} else {
		skipDefault := intent == IntentClosure || intent == IntentLowSignal
		plan = routing.PlanKnowledgeRouteForTurn(msg.Content, skipDefault)
	}
	a.RecordRoutingSnapshotFor(msg.ID, RoutingSnapshot{
		KnowledgeRoute:   string(plan.Primary()),
		KnowledgeReason:  plan.Reason,
		KnowledgeTargets: routeTargetsToStrings(plan.Targets),
	})
	log.Printf("[%s] knowledge_route=%s reason=%s targets=%v", a.Info.Name, plan.Primary(), plan.Reason, plan.Targets)
}
