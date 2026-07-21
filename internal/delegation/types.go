package delegation

import (
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

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
	AgentID      string
	AgentName    string
	AgentType    protocol.AgentType
	Score        int
	Intent       Intent
	CapabilityID string
}

type CapabilityHelpRequest struct {
	FromID          string
	FromName        string
	CreatedBy       string
	CapabilityID    string
	Task            string
	SourceChannel   string
	SourceMessageID string
	Depth           int
}

type CapabilityHelpResult struct {
	HandoffID     string `json:"handoff_id"`
	Channel       string `json:"channel"`
	SourceChannel string `json:"source_channel"`
	HelperID      string `json:"helper_id,omitempty"`
	HelperName    string `json:"helper_name,omitempty"`
	Summary       string `json:"summary,omitempty"`
	Status        string `json:"status"`
	Err           string `json:"error,omitempty"`
}

type CapabilityProvider struct {
	CapabilityID string   `json:"capability_id"`
	AgentIDs     []string `json:"agent_ids"`
	AgentNames   []string `json:"agent_names"`
}

type HandoffRecord struct {
	ID              string     `json:"id"`
	SourceChannel   string     `json:"source_channel"`
	SourceMessageID string     `json:"source_message_id,omitempty"`
	Channel         string     `json:"channel"`
	RequestingID    string     `json:"requesting_agent_id"`
	RequestingName  string     `json:"requesting_agent_name"`
	HelperID        string     `json:"helper_id"`
	HelperName      string     `json:"helper_name"`
	CreatedBy       string     `json:"created_by,omitempty"`
	CapabilityID    string     `json:"capability_id"`
	Task            string     `json:"task"`
	Status          string     `json:"status"`
	Result          string     `json:"result,omitempty"`
	Error           string     `json:"error,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
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
