package agent

import (
	"log"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func (a *Agent) recordKnowledgeRoute(msg *protocol.Message) {
	if a == nil || msg == nil {
		return
	}
	if skipKnowledgeRetrievalForMessage(msg) {
		a.RecordRoutingSnapshot(RoutingSnapshot{
			KnowledgeRoute:  "collab_turn",
			KnowledgeReason: "collab_turn",
		})
		return
	}
	plan := routing.PlanKnowledgeRoute(msg.Content)
	a.RecordRoutingSnapshot(RoutingSnapshot{
		KnowledgeRoute:   string(plan.Primary()),
		KnowledgeReason:  plan.Reason,
		KnowledgeTargets: routeTargetsToStrings(plan.Targets),
	})
	log.Printf("[%s] knowledge_route=%s reason=%s targets=%v", a.Info.Name, plan.Primary(), plan.Reason, plan.Targets)
}
