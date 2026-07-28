package protocol

import "time"

// TurnContextEnvelope is the typed, per-turn context selected for generation.
// It deliberately carries message IDs and provenance so stale instructions can
// be excluded before prompt or provider history assembly.
type TurnContextEnvelope struct {
	Version              int                     `json:"version"`
	Channel              string                  `json:"channel"`
	Goal                 *TurnContextGoal        `json:"goal,omitempty"`
	Decisions            []TurnContextDecision   `json:"decisions,omitempty"`
	UnresolvedActions    []TurnContextAction     `json:"unresolved_actions,omitempty"`
	Corrections          []TurnContextCorrection `json:"corrections,omitempty"`
	SupersededMessageIDs []string                `json:"superseded_message_ids,omitempty"`
	RecentExchanges      []TurnContextExchange   `json:"recent_exchanges,omitempty"`
	RetrievedMemories    []TurnContextItem       `json:"retrieved_memories,omitempty"`
	WorkspaceEvidence    []TurnContextItem       `json:"workspace_evidence,omitempty"`
	Summary              *ConversationSummary    `json:"summary,omitempty"`
	Provenance           []TurnContextProvenance `json:"provenance,omitempty"`
	SectionBudgets       map[string]int          `json:"section_budgets,omitempty"`
}

type TurnContextGoal struct {
	ID            string `json:"id"`
	MessageID     string `json:"message_id,omitempty"`
	LastMessageID string `json:"last_message_id,omitempty"`
	Text          string `json:"text,omitempty"`
	PinnedText    string `json:"pinned_text,omitempty"`
}

type TurnContextDecision struct {
	GoalID      string    `json:"goal_id,omitempty"`
	DecisionKey string    `json:"decision_key"`
	MessageID   string    `json:"message_id,omitempty"`
	Answer      string    `json:"answer"`
	AnsweredAt  time.Time `json:"answered_at,omitempty"`
}

type TurnContextAction struct {
	ID                   string    `json:"id"`
	GoalID               string    `json:"goal_id,omitempty"`
	Action               string    `json:"action,omitempty"`
	Description          string    `json:"description"`
	LastPromiseMessageID string    `json:"last_promise_message_id,omitempty"`
	PromisedAt           time.Time `json:"promised_at,omitempty"`
}

type TurnContextCorrection struct {
	GoalID               string   `json:"goal_id,omitempty"`
	MessageID            string   `json:"message_id"`
	Instruction          string   `json:"instruction"`
	SupersedesMessageIDs []string `json:"supersedes_message_ids,omitempty"`
}

type TurnContextExchange struct {
	User      *Message `json:"user,omitempty"`
	Assistant *Message `json:"assistant,omitempty"`
}

type TurnContextItem struct {
	ID         string  `json:"id,omitempty"`
	Content    string  `json:"content,omitempty"`
	Provenance string  `json:"provenance,omitempty"`
	Score      float64 `json:"score,omitempty"`
}

type TurnContextProvenance struct {
	ID        string  `json:"id"`
	Section   string  `json:"section"`
	Source    string  `json:"source"`
	Score     float64 `json:"score,omitempty"`
	Freshness string  `json:"freshness,omitempty"`
}

// ConversationSummary is a cumulative, versioned checkpoint. Digest is the
// compact model-authored state; LastCompactedMessageID bounds the next delta.
type ConversationSummary struct {
	Version                int       `json:"version"`
	Digest                 string    `json:"digest"`
	LastCompactedMessageID string    `json:"last_compacted_message_id,omitempty"`
	UpdatedAt              time.Time `json:"updated_at,omitempty"`
}
