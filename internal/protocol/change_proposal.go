package protocol

import (
	"encoding/json"
	"strings"
	"time"
)

const MetaChangeProposal = "change_proposal"

type ChangeProposalKind string

const (
	ChangeProposalKindFile ChangeProposalKind = "file_change"
	ChangeProposalKindGit  ChangeProposalKind = "git_change"
)

type ChangeProposalStatus string

const (
	ChangeProposalStatusPending  ChangeProposalStatus = "pending"
	ChangeProposalStatusApplying ChangeProposalStatus = "applying"
	ChangeProposalStatusApproved ChangeProposalStatus = "approved"
	ChangeProposalStatusRejected ChangeProposalStatus = "rejected"
	ChangeProposalStatusStale    ChangeProposalStatus = "stale"
	ChangeProposalStatusExpired  ChangeProposalStatus = "expired"
	ChangeProposalStatusFailed   ChangeProposalStatus = "failed"
)

// ChangeProposalCard is the typed, versioned metadata rendered as an inline
// chat card. The proposal managers remain authoritative for mutating state.
type ChangeProposalCard struct {
	Version     int                  `json:"version"`
	Kind        ChangeProposalKind   `json:"kind"`
	ID          string               `json:"id"`
	Status      ChangeProposalStatus `json:"status"`
	Operation   string               `json:"operation"`
	FilePath    string               `json:"file_path,omitempty"`
	OldPath     string               `json:"old_path,omitempty"`
	NewPath     string               `json:"new_path,omitempty"`
	Message     string               `json:"message,omitempty"`
	Paths       []string             `json:"paths,omitempty"`
	WorkspaceID string               `json:"workspace_id,omitempty"`
	RequestedAt time.Time            `json:"requested_at,omitempty"`
	ExpiresAt   time.Time            `json:"expires_at,omitempty"`
	Reason      string               `json:"reason,omitempty"`
	Error       string               `json:"error,omitempty"`
}

func ParseChangeProposalCard(raw interface{}) (ChangeProposalCard, bool) {
	var card ChangeProposalCard
	if raw == nil {
		return card, false
	}
	data, err := json.Marshal(raw)
	if err != nil || json.Unmarshal(data, &card) != nil {
		return ChangeProposalCard{}, false
	}
	card.ID = strings.TrimSpace(card.ID)
	if card.ID == "" || card.Kind == "" {
		return ChangeProposalCard{}, false
	}
	if card.Version == 0 {
		card.Version = 1
	}
	if card.Status == "" {
		card.Status = ChangeProposalStatusPending
	}
	return card, true
}
