package learning

import "time"

// Category classifies a user-confirmed learning.
type Category string

const (
	CategoryPreference    Category = "preference"
	CategoryFact          Category = "fact"
	CategoryWorkflow      Category = "workflow"
	CategoryCommunication Category = "communication"
)

// Entry is a user-confirmed fact scoped to one expert agent.
type Entry struct {
	ID               string    `json:"id"`
	AgentID          string    `json:"agent_id"`
	AgentType        string    `json:"agent_type,omitempty"`
	AgentName        string    `json:"agent_name,omitempty"`
	Content          string    `json:"content"`
	Category         Category  `json:"category"`
	SourceChannel    string    `json:"source_channel,omitempty"`
	SourceMessageID  string    `json:"source_message_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	ConfirmedAt      time.Time `json:"confirmed_at"`
	Active           bool      `json:"active"`
}

const (
	DefaultPromptBudget = 2 * 1024
)
