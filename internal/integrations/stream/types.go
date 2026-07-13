package stream

import "time"

// Protocol identifies the broker protocol for a subscription.
type Protocol string

const (
	ProtocolMQTT  Protocol = "mqtt"
	ProtocolKafka Protocol = "kafka"
)

// ActionType is the dispatcher target for a matched message.
type ActionType string

const (
	ActionRunbook ActionType = "runbook"
	ActionChannel ActionType = "channel"
	ActionWebhook ActionType = "webhook"
)

// MatchOp is how MatchSpec compares a JSON field.
type MatchOp string

const (
	MatchEquals   MatchOp = "equals"
	MatchContains MatchOp = "contains"
)

// MatchSpec optionally filters messages by a JSON field.
// Empty/nil Match means every message matches.
type MatchSpec struct {
	JSONPath string  `json:"json_path,omitempty"`
	Op       MatchOp `json:"op,omitempty"` // equals (default) | contains
	Value    string  `json:"value,omitempty"`
}

// ActionSpec describes what to do when a message matches.
type ActionSpec struct {
	Type ActionType `json:"type"`

	// Runbook action
	DefinitionID string            `json:"definition_id,omitempty"`
	Version      int               `json:"version,omitempty"`
	AgentIDs     []string          `json:"agent_ids,omitempty"`
	Channel      string            `json:"channel,omitempty"` // hub channel for runbook create
	InputMap     map[string]string `json:"input_map,omitempty"` // JSON path → runbook input key

	// Channel action
	HubChannel      string   `json:"hub_channel,omitempty"`
	MessageTemplate string   `json:"message_template,omitempty"` // {{payload}} {{topic}} {{key}}
	MentionAgentIDs []string `json:"mention_agent_ids,omitempty"`

	// Webhook action
	WebhookConnectorID string `json:"webhook_connector_id,omitempty"`
	URLOverride        string `json:"url_override,omitempty"`
	BodyTemplate       string `json:"body_template,omitempty"` // default: raw payload
}

// Subscription binds a broker topic to an action.
type Subscription struct {
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	Enabled     bool       `json:"enabled"`
	Protocol    Protocol   `json:"protocol"`
	ConnectorID string     `json:"connector_id"`
	Topic       string     `json:"topic"`
	Match       *MatchSpec `json:"match,omitempty"`
	DebounceMs  int        `json:"debounce_ms,omitempty"`
	Action      ActionSpec `json:"action"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty"`
}

// Event is a normalized inbound stream message.
type Event struct {
	Protocol       Protocol  `json:"protocol"`
	Topic          string    `json:"topic"`
	Key            string    `json:"key,omitempty"`
	Payload        string    `json:"payload"`
	ReceivedAt     time.Time `json:"received_at"`
	SubscriptionID string    `json:"subscription_id"`
}

// SubStatus is live status for one subscription worker.
type SubStatus struct {
	SubscriptionID string     `json:"subscription_id"`
	Label          string     `json:"label,omitempty"`
	Enabled        bool       `json:"enabled"`
	Connected      bool       `json:"connected"`
	LastMessageAt  *time.Time `json:"last_message_at,omitempty"`
	LastFireAt     *time.Time `json:"last_fire_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	FireCount      int64      `json:"fire_count"`
	SkipCount      int64      `json:"skip_count"`
}

// ManagerStatus is returned from GET /api/stream/status.
type ManagerStatus struct {
	Running       bool        `json:"running"`
	Subscriptions []SubStatus `json:"subscriptions"`
}
