package hub

import "github.com/camronwood/neural-junkie/internal/agent"

// ResolveRuntimeAgentForFastEdit returns an in-process *agent.Agent for fast-edit, or nil.
func (ch *CommandHandler) ResolveRuntimeAgentForFastEdit(agentID string) *agent.Agent {
	ra := ch.resolveRuntimeAgent(agentID)
	if ra == nil {
		return nil
	}
	if ag, ok := ra.(*agent.Agent); ok {
		return ag
	}
	return nil
}
