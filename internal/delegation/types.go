package delegation

import "github.com/camronwood/neural-junkie/internal/protocol"

// Intent classifies what kind of consult to run for a delegate.
type Intent string

const (
	IntentGeneral         Intent = "general"
	IntentDomainReasoning Intent = "domain_reasoning"
	IntentDomainTools     Intent = "domain_tools"
	IntentMultiDomain     Intent = "multi_domain"
)

// Candidate is a specialist the hub may consult.
type Candidate struct {
	AgentID   string
	AgentName string
	AgentType protocol.AgentType
	Score     int
	Intent    Intent
}

// ConsultRequest is an internal hub consult (no channel broadcast).
type ConsultRequest struct {
	FromID      string
	FromName    string
	ToID        string
	SubQuestion string
	Channel     string
	Depth       int
	Intent      Intent
}

// ConsultResult is the delegate answer returned to the consulting agent.
type ConsultResult struct {
	Text      string
	AgentName string
	Model     string
	Intent    Intent
	Err       string
}

// ResolveOptions tunes registry scoring.
type ResolveOptions struct {
	MinScore       int
	MaxCandidates  int
	ExcludeAgentID string
}
