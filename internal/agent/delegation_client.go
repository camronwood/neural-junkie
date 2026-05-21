package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// DelegationClient is implemented by the hub command handler for cross-agent consult.
type DelegationClient interface {
	DelegationEnabled() bool
	ResolveConsultants(from protocol.AgentInfo, question string) []delegation.Candidate
	Consult(ctx context.Context, req delegation.ConsultRequest) (delegation.ConsultResult, error)
}

// delegationHub exposes optional delegation on HubClient implementations.
type delegationHub interface {
	GetDelegation() DelegationClient
}

func (a *Agent) getDelegationClient() DelegationClient {
	if a.Hub == nil {
		return nil
	}
	if dh, ok := a.Hub.(delegationHub); ok {
		return dh.GetDelegation()
	}
	return nil
}
