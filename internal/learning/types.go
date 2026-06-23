package learning

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// Category classifies a user-confirmed learning.
type Category string

const (
	CategoryPreference    Category = "preference"
	CategoryFact          Category = "fact"
	CategoryWorkflow      Category = "workflow"
	CategoryCommunication Category = "communication"
)

// Scope controls where a learning is injected.
type Scope string

const (
	ScopeAgent         Scope = "agent"
	ScopeGlobal        Scope = "global"
	ScopeCollaboration Scope = "collaboration"
	ScopeWorkspace     Scope = "workspace"
)

const (
	DefaultPromptBudget   = 2 * 1024
	DefaultGlobalBudget   = 512
	DefaultCollabBudget   = 512
	DefaultGlobalTopK     = 3
	DefaultAgentTopK      = 5
	DefaultCollabTopK     = 3
	DefaultEmbedModel     = "nomic-embed-text"
	StoreVersion          = 2
)

// Entry is a user-confirmed fact with v2 scope and lifecycle fields.
type Entry struct {
	ID              string    `json:"id"`
	Scope           Scope     `json:"scope,omitempty"`
	UserID          string    `json:"user_id,omitempty"`
	AgentID         string    `json:"agent_id"`
	AgentType       string    `json:"agent_type,omitempty"`
	AgentName       string    `json:"agent_name,omitempty"`
	CollaborationID string    `json:"collaboration_id,omitempty"`
	WorkspaceID     string    `json:"workspace_id,omitempty"`
	Content         string    `json:"content"`
	Category        Category  `json:"category"`
	ContentHash     string    `json:"content_hash,omitempty"`
	SourceChannel   string    `json:"source_channel,omitempty"`
	SourceMessageID string    `json:"source_message_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	ConfirmedAt     time.Time `json:"confirmed_at"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
	UseCount        int       `json:"use_count,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	Active          bool      `json:"active"`
}

// Filter selects active learnings for list/query.
type Filter struct {
	AgentID         string
	AgentType       string
	AgentName       string
	UserID          string
	Scope           Scope
	CollaborationID string
	WorkspaceID     string
	IncludeLegacy   bool // empty user_id rows when UserID set
}

// UpdatePatch fields for PUT /api/learnings/{id}.
type UpdatePatch struct {
	Content         *string
	Category        *Category
	Scope           *Scope
	CollaborationID *string
}

// PromptContext carries retrieval inputs at agent prompt build time.
type PromptContext struct {
	Query           string
	UserID          string
	AgentType       string
	AgentName       string
	Channel         string
	CollaborationID string
	WorkspaceID     string
}

// PromptResult is injection output for debug metadata.
type PromptResult struct {
	Count int
	IDs   []string
}

func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.ToLower(content))))
	return hex.EncodeToString(sum[:8])
}

func NormalizeScope(s Scope) Scope {
	switch s {
	case ScopeGlobal, ScopeCollaboration, ScopeWorkspace:
		return s
	default:
		return ScopeAgent
	}
}

func SlugUserID(username string) string {
	s := strings.TrimSpace(strings.ToLower(username))
	s = strings.ReplaceAll(s, " ", "-")
	if s == "" {
		return ""
	}
	return s
}
