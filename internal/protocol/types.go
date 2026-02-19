package protocol

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// MessageType defines the type of message being sent
type MessageType string

const (
	MessageTypeChat              MessageType = "chat"
	MessageTypeQuestion          MessageType = "question"
	MessageTypeAnswer            MessageType = "answer"
	MessageTypeSystemInfo        MessageType = "system_info"
	MessageTypeAgentJoin         MessageType = "agent_join"
	MessageTypeAgentLeave        MessageType = "agent_leave"
	MessageTypeAgentStatus       MessageType = "agent_status"
	MessageTypeContextShare      MessageType = "context_share"
	MessageTypeRequestHelp       MessageType = "request_help"
	MessageTypeCommandOutput     MessageType = "command_output"
	MessageTypeCommandSuggestion MessageType = "command_suggestion"
	MessageTypeDesignOutput      MessageType = "design_output"
	MessageTypeFileChange        MessageType = "file_change"
)

// AgentType defines the specialty of an agent
type AgentType string

const (
	AgentTypeFrontend   AgentType = "frontend"
	AgentTypeBackend    AgentType = "backend"
	AgentTypeDevOps     AgentType = "devops"
	AgentTypeDatabase   AgentType = "database"
	AgentTypeSecurity   AgentType = "security"
	AgentTypeRust       AgentType = "rust"
	AgentTypeGeneral    AgentType = "general"
	AgentTypeRepo       AgentType = "repo"
	AgentTypeHelper     AgentType = "helper"     // Custom helper/expert agents
	AgentTypeModerator  AgentType = "moderator"  // System moderator agent
	AgentTypeAssistant  AgentType = "assistant"  // Personal assistant agent
	AgentTypeConfluence AgentType = "confluence" // Confluence documentation agents
	AgentTypeCLI        AgentType = "cli"        // CLI-backed agents (Cursor, Claude CLI, etc.)
)

// AIProviderType defines the AI provider being used
type AIProviderType string

const (
	ProviderClaude    AIProviderType = "claude"
	ProviderOllama    AIProviderType = "ollama"
	ProviderCursorCLI AIProviderType = "cursor-cli"
)

// IndexingStatus defines the status of repository indexing for repo agents
type IndexingStatus string

const (
	IndexingStatusIndexing   IndexingStatus = "indexing"
	IndexingStatusReady      IndexingStatus = "ready"
	IndexingStatusReindexing IndexingStatus = "reindexing"
	IndexingStatusError      IndexingStatus = "error"
)

// ThinkingStatus defines the status of agent thinking/response generation
type ThinkingStatus string

const (
	ThinkingStatusStarted   ThinkingStatus = "started"
	ThinkingStatusCompleted ThinkingStatus = "completed"
	ThinkingStatusError     ThinkingStatus = "error"
)

// Message represents a message in the chat room
type Message struct {
	ID                string                 `json:"id"`
	Type              MessageType            `json:"type"`
	Channel           string                 `json:"channel"`
	From              AgentInfo              `json:"from"`
	Content           string                 `json:"content"`
	Timestamp         time.Time              `json:"timestamp"`
	ReplyTo           string                 `json:"reply_to,omitempty"`        // ID of message being replied to
	ThreadID          string                 `json:"thread_id,omitempty"`       // ID of the thread this message belongs to
	IsThreadReply     bool                   `json:"is_thread_reply,omitempty"` // Whether this is a reply in a thread
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Tags              []string               `json:"tags,omitempty"`               // e.g., ["urgent", "security", "bug"]
	Mentions          []string               `json:"mentions,omitempty"`           // Agent IDs mentioned
}

// IsFromSystem checks if the message is from a system agent
func (m *Message) IsFromSystem() bool {
	return m.From.Type == AgentTypeGeneral && m.From.Name == "System"
}

// AgentInfo contains information about an agent
type AgentInfo struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Type               AgentType `json:"type"`
	Expertise          []string  `json:"expertise"`            // Specific skills/technologies
	Status             string    `json:"status"`               // "active", "busy", "idle", "paused", "removed"
	Model              string    `json:"model"`                // AI model being used
	AIProvider         string    `json:"ai_provider"`          // AI provider being used ("claude", "ollama")
	AIModel            string    `json:"ai_model"`             // Specific model name (e.g., "claude-sonnet", "llama3.1")
	IsPaused           bool      `json:"is_paused"`            // Whether the agent is paused
	SupportsVision     bool      `json:"supports_vision"`      // Whether the agent can process images
	IndexingStatus     string    `json:"indexing_status"`      // "indexing", "ready", "reindexing", "error" (for repo/confluence agents)
	IndexProgress      int       `json:"index_progress"`       // 0-100 percentage (for repo/confluence agents)
	RepositoryPath     string    `json:"repository_path"`      // Path to repository (for repo agents)
	KnowledgePath      string    `json:"knowledge_path"`       // Path to knowledge base (for helper agents)
	ConfluenceSpaceKey string    `json:"confluence_space_key"` // Confluence space key (for confluence agents)
	LastActiveTime     time.Time `json:"last_active_time"`     // When agent was last in a channel
	RemovedFrom        []string  `json:"removed_from"`         // List of channels agent was removed from
}

// ChannelType classifies the purpose of a channel
type ChannelType string

const (
	ChannelTypePublic ChannelType = "public" // Visible to all (e.g. #general)
	ChannelTypeDM     ChannelType = "dm"     // 1:1 user-to-agent direct message
	ChannelTypeCustom ChannelType = "custom" // User-created channel with curated agents
)

// Channel represents a chat channel/room
type Channel struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Project     string      `json:"project,omitempty"`
	Type        ChannelType `json:"type"`
	CreatedBy   string      `json:"created_by,omitempty"`
	Created     time.Time   `json:"created"`
	Agents      []AgentInfo `json:"agents"`
	Members     []string    `json:"members,omitempty"` // Explicitly added agent IDs
	Tags        []string    `json:"tags,omitempty"`
}

// ThreadMetadata contains metadata about a message thread
type ThreadMetadata struct {
	ThreadID      string    `json:"thread_id"`
	ReplyCount    int       `json:"reply_count"`
	LastReplyTime time.Time `json:"last_reply_time"`
	Participants  []string  `json:"participants"` // Agent/user names who participated in thread
}

// CommandOutput represents the result of executing a CLI command
type CommandOutput struct {
	Command  string        `json:"command"`
	Plugin   string        `json:"plugin"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
	Success  bool          `json:"success"`
}

// CachedAgentInfo represents a cached agent that can be loaded
type CachedAgentInfo struct {
	Type      string                 `json:"type"`       // "repo", "helper", "confluence"
	Name      string                 `json:"name"`       // Agent name
	Path      string                 `json:"path"`       // Repository path, knowledge path, or space key
	LastUsed  string                 `json:"last_used"`  // ISO timestamp of last use
	CacheSize int64                  `json:"cache_size"` // Size in bytes
	Metadata  map[string]interface{} `json:"metadata"`   // Additional metadata
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, channel string, from AgentInfo, content string) *Message {
	msg := &Message{
		ID:        uuid.New().String(),
		Type:      msgType,
		Channel:   channel,
		From:      from,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
		Tags:      []string{},
		Mentions:  []string{},
	}

	// Parse mentions from content
	msg.Mentions = ParseMentions(content)

	return msg
}

// AddTag adds a tag to the message
func (m *Message) AddTag(tag string) {
	m.Tags = append(m.Tags, tag)
}

// Mention adds an agent mention
func (m *Message) Mention(agentID string) {
	m.Mentions = append(m.Mentions, agentID)
}

// IsMentioned checks if an agent is mentioned in the message
func (m *Message) IsMentioned(agentID string) bool {
	for _, id := range m.Mentions {
		if id == agentID {
			return true
		}
	}
	return false
}

// HasMentions checks if the message has any @mentions
func (m *Message) HasMentions() bool {
	return len(m.Mentions) > 0
}

// IsReviewRequest checks if the message contains review trigger phrases
func (m *Message) IsReviewRequest() bool {
	content := strings.ToLower(m.Content)

	reviewKeywords := []string{
		"thoughts?",
		"what do you think",
		"agree?",
		"disagree?",
		"review this",
		"opinion?",
		"your take?",
		"perspective?",
		"thoughts on this",
		"makes sense?",
		"sound right?",
		"your thoughts",
		"what's your take",
		"do you agree",
	}

	for _, keyword := range reviewKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// GetReviewDepth returns the depth of review chain (0 for original, 1 for first review, etc.)
func (m *Message) GetReviewDepth() int {
	if depth, ok := m.Metadata["review_depth"].(float64); ok {
		return int(depth)
	}
	if depth, ok := m.Metadata["review_depth"].(int); ok {
		return depth
	}
	return 0
}

// SetReviewDepth sets the review depth in metadata
func (m *Message) SetReviewDepth(depth int) {
	m.Metadata["review_depth"] = depth
}

// GetReviewedMessageID returns the ID of the message being reviewed, if any
func (m *Message) GetReviewedMessageID() string {
	if id, ok := m.Metadata["reviewed_message_id"].(string); ok {
		return id
	}
	return ""
}

// SetReviewedMessageID sets the ID of the message being reviewed
func (m *Message) SetReviewedMessageID(id string) {
	m.Metadata["reviewed_message_id"] = id
}

// GetOriginalQuestionID returns the ID of the original user question
func (m *Message) GetOriginalQuestionID() string {
	if id, ok := m.Metadata["original_question_id"].(string); ok {
		return id
	}
	return ""
}

// SetOriginalQuestionID sets the ID of the original user question
func (m *Message) SetOriginalQuestionID(id string) {
	m.Metadata["original_question_id"] = id
}

// IsInThread returns true if message is part of a thread
func (m *Message) IsInThread() bool {
	return m.ThreadID != ""
}

// GetThreadID returns the thread ID (parent message ID for thread messages)
func (m *Message) GetThreadID() string {
	return m.ThreadID
}

// IsUserCreatedAgent checks if an agent type is user-created (not system agent)
func IsUserCreatedAgent(agentType string) bool {
	return agentType == "repo" || agentType == "helper" || agentType == "confluence"
}

// CommandSuggestion represents a command suggested by an agent
type CommandSuggestion struct {
	ID          string    `json:"id"`          // Unique identifier for this suggestion
	Command     string    `json:"command"`     // The command to execute
	Plugin      string    `json:"plugin"`
	Description string    `json:"description"` // Human-readable description
	IsSafe      bool      `json:"is_safe"`     // Whether command is read-only/safe
	AgentName   string    `json:"agent_name"`  // Name of agent who suggested it
	MessageID   string    `json:"message_id"`  // ID of the message containing this suggestion
	CreatedAt   time.Time `json:"created_at"`
}

// TerminalCommand represents a command being executed in the terminal
type TerminalCommand struct {
	ID        string        `json:"id"`
	Command   string        `json:"command"`
	Status    string        `json:"status"` // "pending", "executing", "completed", "failed"
	ExitCode  int           `json:"exit_code"`
	Stdout    string        `json:"stdout"`
	Stderr    string        `json:"stderr"`
	Duration  time.Duration `json:"duration"`
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
}

// PendingReview tracks a message waiting for a repository agent to be created and respond
type PendingReview struct {
	OriginalMessage *Message
	RepoPath        string
	RepoAgentName   string
	CreatedAt       time.Time
}

// CommandArgument describes a single argument for a slash command
type CommandArgument struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"` // "string", "path", "provider", "model", "agent-name"
	Required    bool     `json:"required"`
	Options     []string `json:"options,omitempty"`
	Default     string   `json:"default,omitempty"`
}

// CommandDefinition describes a slash command exposed to the UI
type CommandDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Arguments   []CommandArgument `json:"arguments"`
}

// FileChangeProposal represents a file change proposal embedded in a message
type FileChangeProposal struct {
	ChangeID    string                 `json:"change_id"`
	Operation   string                 `json:"operation"` // "create", "edit", "delete", "move"
	FilePath    string                 `json:"file_path"`
	OldPath     string                 `json:"old_path,omitempty"`    // For move operations
	NewPath     string                 `json:"new_path,omitempty"`    // For move operations
	OldContent  string                 `json:"old_content,omitempty"` // For edit operations
	NewContent  string                 `json:"new_content,omitempty"` // For create/edit operations
	Agent       AgentInfo              `json:"agent"`
	Channel     string                 `json:"channel"`
	RequestedAt time.Time              `json:"requested_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	IsDelete    bool                   `json:"is_delete"` // Special flag for delete operations
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
