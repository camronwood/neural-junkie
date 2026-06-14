package delegation

import (
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

// ClassifyForAgent picks consult intent for a target specialist type.
func ClassifyForAgent(agentType protocol.AgentType, question string) Intent {
	dec := routing.ClassifyRules(routing.Input{
		Text:      question,
		AgentType: string(agentType),
	})
	if dec.ToolNeed {
		return IntentDomainTools
	}
	return IntentDomainReasoning
}

// ClassifyMessage inspects the user message for multi-domain signals.
func ClassifyMessage(_ protocol.AgentType, _ string, candidates []Candidate) Intent {
	if len(candidates) > 1 {
		return IntentMultiDomain
	}
	if len(candidates) == 1 {
		return candidates[0].Intent
	}
	return IntentGeneral
}
