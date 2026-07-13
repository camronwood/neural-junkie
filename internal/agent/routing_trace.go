package agent

import (
	"github.com/camronwood/neural-junkie/internal/protocol"
	unified "github.com/camronwood/neural-junkie/internal/routing"
)

func (a *Agent) recordTurnGovernance(msg *protocol.Message) {
	if a == nil || msg == nil {
		return
	}
	caps := protocol.ResolveTurnCapabilities(msg)
	a.RecordRoutingSnapshot(RoutingSnapshot{
		ComposerMode: caps.ComposerMode,
		ContextScope: caps.ContextTier,
		ImplSession:  caps.CanRunImplSession,
	})
}

func (a *Agent) recordClassifierRouting(msg *protocol.Message) {
	if a == nil || msg == nil || msg.Type == protocol.MessageTypeCollabTask {
		return
	}
	dec := unified.ClassifyRules(unified.Input{
		Text:      msg.Content,
		AgentType: string(a.Info.Type),
	})
	snap := RoutingSnapshot{
		Domain:               dec.Domain,
		CostTier:             dec.CostTier,
		ClassifierIntent:     dec.Intent,
		ClassifierToolNeed:   dec.ToolNeed,
		ClassifierConfidence: dec.Confidence,
		ClassifierLoRATag:    dec.LoRATag,
	}
	a.RecordRoutingSnapshot(snap)
}
