package gitchange

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// Operation is a git action requiring approval.
type Operation string

const (
	OpStage  Operation = "stage"
	OpCommit Operation = "commit"
	OpPush   Operation = "push"
)

// Proposal is a pending git operation from an agent.
type Proposal struct {
	ID          string             `json:"id"`
	Operation   Operation          `json:"operation"`
	Message     string             `json:"message,omitempty"`
	Paths       []string           `json:"paths,omitempty"`
	WorkspaceID string             `json:"workspace_id"`
	Agent       protocol.AgentInfo `json:"agent"`
	Channel     string             `json:"channel"`
	RequestedAt time.Time          `json:"requested_at"`
	ExpiresAt   time.Time          `json:"expires_at"`
	Status      string             `json:"status"`
	Reason      string             `json:"reason,omitempty"`
	Error       string             `json:"error,omitempty"`
}

// Manager tracks pending git proposals.
type Manager struct {
	mu      sync.RWMutex
	pending map[string]*Proposal
	byUser  map[string][]string
}

func NewManager() *Manager {
	return &Manager{
		pending: make(map[string]*Proposal),
		byUser:  make(map[string][]string),
	}
}

// RestorePending rehydrates a durable git proposal after restart.
func (m *Manager) RestorePending(proposal *Proposal) {
	if m == nil || proposal == nil || proposal.ID == "" || proposal.Status != "pending" {
		return
	}
	copyProposal := *proposal
	copyProposal.Paths = append([]string(nil), proposal.Paths...)
	m.mu.Lock()
	if _, exists := m.pending[proposal.ID]; !exists {
		m.pending[proposal.ID] = &copyProposal
		m.byUser[proposal.Agent.ID] = append(m.byUser[proposal.Agent.ID], proposal.ID)
	}
	m.mu.Unlock()
}

func (m *Manager) Propose(p Proposal) (*Proposal, error) {
	if p.Operation == "" {
		return nil, fmt.Errorf("operation required")
	}
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p.RequestedAt = time.Now()
	p.ExpiresAt = p.RequestedAt.Add(30 * time.Minute)
	p.Status = "pending"
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := p
	m.pending[p.ID] = &cp
	user := p.Agent.ID
	m.byUser[user] = append(m.byUser[user], p.ID)
	return &cp, nil
}

func (m *Manager) ListPending(userID string) []*Proposal {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Proposal, 0)
	now := time.Now()
	for _, p := range m.pending {
		if p.Status == "pending" && now.Before(p.ExpiresAt) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].RequestedAt.Before(out[j].RequestedAt)
	})
	return out
}

func (m *Manager) Get(id string) (*Proposal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.pending[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return p, nil
}

func (m *Manager) MarkApplying(id string) (*Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pending[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	if p.Status != "pending" {
		return nil, fmt.Errorf("git change already processed")
	}
	if time.Now().After(p.ExpiresAt) {
		p.Status = "expired"
		return nil, fmt.Errorf("git change expired")
	}
	p.Status = "applying"
	return p, nil
}

func (m *Manager) MarkApproved(id string) (*Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pending[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	if p.Status != "applying" {
		return nil, fmt.Errorf("git change is not applying")
	}
	p.Status = "approved"
	p.Error = ""
	return p, nil
}

func (m *Manager) MarkFailed(id string, errText string) (*Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pending[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	p.Status = "failed"
	p.Error = errText
	return p, nil
}

func (m *Manager) Reject(id, reason string) (*Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pending[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	if p.Status != "pending" {
		return nil, fmt.Errorf("git change already processed")
	}
	p.Status = "rejected"
	p.Reason = reason
	return p, nil
}
