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
	decision := routing.ClassifyKnowledgeRoute(msg.Content)
	a.RecordRoutingSnapshot(RoutingSnapshot{
		KnowledgeRoute:  string(decision.Target),
		KnowledgeReason: decision.Reason,
	})
	log.Printf("[%s] knowledge_route=%s reason=%s", a.Info.Name, decision.Target, decision.Reason)
}
