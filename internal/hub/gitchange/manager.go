package gitchange

import (
	"fmt"
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
	ID          string            `json:"id"`
	Operation   Operation         `json:"operation"`
	Message     string            `json:"message,omitempty"`
	Paths       []string          `json:"paths,omitempty"`
	WorkspaceID string            `json:"workspace_id"`
	Agent       protocol.AgentInfo `json:"agent"`
	Channel     string            `json:"channel"`
	RequestedAt time.Time         `json:"requested_at"`
	Status      string            `json:"status"`
}

// Manager tracks pending git proposals.
type Manager struct {
	mu       sync.RWMutex
	pending  map[string]*Proposal
	byUser   map[string][]string
}

func NewManager() *Manager {
	return &Manager{
		pending: make(map[string]*Proposal),
		byUser:  make(map[string][]string),
	}
}

func (m *Manager) Propose(p Proposal) (*Proposal, error) {
	if p.Operation == "" {
		return nil, fmt.Errorf("operation required")
	}
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p.RequestedAt = time.Now()
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
	ids := m.byUser[userID]
	out := make([]*Proposal, 0, len(ids))
	for _, id := range ids {
		if p, ok := m.pending[id]; ok && p.Status == "pending" {
			out = append(out, p)
		}
	}
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

func (m *Manager) MarkApproved(id string) (*Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pending[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	p.Status = "approved"
	return p, nil
}

func (m *Manager) Reject(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pending[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	p.Status = "rejected"
	return nil
}
