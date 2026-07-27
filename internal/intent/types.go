// Package intent owns the single semantic interpretation produced for a user turn.
// It deliberately contains no protocol, hub, or agent dependencies so every routing
// consumer can share the same validated decision.
package intent

import (
	"fmt"
	"slices"
	"strings"
)

const SchemaVersion = 1

type InteractionKind string

const (
	InteractionClosure      InteractionKind = "closure"
	InteractionCasual       InteractionKind = "casual"
	InteractionQuestion     InteractionKind = "question"
	InteractionTask         InteractionKind = "task"
	InteractionContinuation InteractionKind = "continuation"
	InteractionCorrection   InteractionKind = "correction"
)

type Action string

const (
	ActionAnswer   Action = "answer"
	ActionInspect  Action = "inspect"
	ActionPlan     Action = "plan"
	ActionDebug    Action = "debug"
	ActionEdit     Action = "edit"
	ActionRun      Action = "run"
	ActionContinue Action = "continue"
	ActionArtifact Action = "artifact"
	ActionImage    Action = "image"
	ActionAskUser  Action = "ask_user"
)

type RetrievalTarget string

const (
	RetrievalMemory         RetrievalTarget = "conversation_memory"
	RetrievalCodebase       RetrievalTarget = "codebase"
	RetrievalCodeGraph      RetrievalTarget = "code_graph"
	RetrievalPriorReference RetrievalTarget = "prior_reference"
	RetrievalCollaboration  RetrievalTarget = "collab_artifact"
)

type Mutation string

const (
	MutationNone      Mutation = "none"
	MutationExternal  Mutation = "external"
	MutationWorkspace Mutation = "workspace"
)

type Source string

const (
	SourceStructural     Source = "structural"
	SourceLocalModel     Source = "local_model"
	SourceSafeFallback   Source = "safe_fallback"
	SourceLegacyRollback Source = "legacy_rollback"
)

// TurnFeatures contains explicit state and bounded conversational facts. Free text
// is present only for semantic classification; policy must not infer permissions from it.
type TurnFeatures struct {
	Text                 string     `json:"text,omitempty"`
	ComposerMode         string     `json:"composer_mode,omitempty"`
	ExplicitAction       Action     `json:"explicit_action,omitempty"`
	ExplicitRecipient    string     `json:"explicit_recipient,omitempty"`
	ReplyTarget          string     `json:"reply_target,omitempty"`
	PendingActionID      string     `json:"pending_action_id,omitempty"`
	PendingAction        Action     `json:"pending_action,omitempty"`
	PendingDescription   string     `json:"pending_description,omitempty"`
	CollaborationPhase   string     `json:"collaboration_phase,omitempty"`
	RecentExchanges      []Exchange `json:"recent_exchanges,omitempty"`
	IsSlashCommand       bool       `json:"is_slash_command,omitempty"`
	IsDirectMessage      bool       `json:"is_direct_message,omitempty"`
	HasExplicitMention   bool       `json:"has_explicit_mention,omitempty"`
	HasWorkspace         bool       `json:"has_workspace,omitempty"`
	CanProposeFiles      bool       `json:"can_propose_files,omitempty"`
	CanRunImplementation bool       `json:"can_run_implementation,omitempty"`
	FrontierAllowed      bool       `json:"frontier_allowed,omitempty"`
	// Open canvas artifact in the channel (from recent artifact_changed), when present.
	// Classifier/policy use this for revisions without phrase matching.
	OpenArtifactID       string `json:"open_artifact_id,omitempty"`
	OpenArtifactRenderer string `json:"open_artifact_renderer,omitempty"`
	OpenArtifactTitle    string `json:"open_artifact_title,omitempty"`
}

type Exchange struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SemanticIntent is model output. It describes meaning, never authorization.
type SemanticIntent struct {
	SchemaVersion      int               `json:"schema_version"`
	Interaction        InteractionKind   `json:"interaction"`
	RequestedAction    Action            `json:"requested_action"`
	Domain             string            `json:"domain,omitempty"`
	RecipientType      string            `json:"recipient_type,omitempty"`
	Retrieval          []RetrievalTarget `json:"retrieval,omitempty"`
	MutationRequested  Mutation          `json:"mutation_requested"`
	ContinuationTarget string            `json:"continuation_target,omitempty"`
	Complexity         string            `json:"complexity,omitempty"`
	Confidence         float64           `json:"confidence"`
	Ambiguities        []string          `json:"ambiguities,omitempty"`
	ReasonCodes        []string          `json:"reason_codes,omitempty"`
}

// TurnDecision is the canonical, policy-resolved contract consumed by routing
// and execution. Action and Mutation are authoritative; RequestedAction records
// what the user meant before deterministic safety policy was applied.
type TurnDecision struct {
	SchemaVersion       int               `json:"schema_version"`
	Interaction         InteractionKind   `json:"interaction"`
	RequestedAction     Action            `json:"requested_action"`
	Action              Action            `json:"action"`
	Domain              string            `json:"domain,omitempty"`
	RecipientType       string            `json:"recipient_type,omitempty"`
	Retrieval           []RetrievalTarget `json:"retrieval,omitempty"`
	Mutation            Mutation          `json:"mutation"`
	ContinuationTarget  string            `json:"continuation_target,omitempty"`
	Complexity          string            `json:"complexity,omitempty"`
	Confidence          float64           `json:"confidence"`
	Source              Source            `json:"source"`
	ReasonCodes         []string          `json:"reason_codes,omitempty"`
	PolicyOverrides     []string          `json:"policy_overrides,omitempty"`
	ClassifierModel     string            `json:"classifier_model,omitempty"`
	ClassifierLatencyMS int64             `json:"classifier_latency_ms,omitempty"`
	AbstentionReason    string            `json:"abstention_reason,omitempty"`
}

func (s SemanticIntent) Validate() error {
	if s.SchemaVersion != SchemaVersion {
		return fmt.Errorf("semantic intent schema_version=%d, want %d", s.SchemaVersion, SchemaVersion)
	}
	if !validInteraction(s.Interaction) {
		return fmt.Errorf("invalid interaction %q", s.Interaction)
	}
	if !validAction(s.RequestedAction) {
		return fmt.Errorf("invalid requested_action %q", s.RequestedAction)
	}
	if !validMutation(s.MutationRequested) {
		return fmt.Errorf("invalid mutation_requested %q", s.MutationRequested)
	}
	if !validDomain(s.Domain) {
		return fmt.Errorf("invalid domain %q", s.Domain)
	}
	if !validRecipient(s.RecipientType) {
		return fmt.Errorf("invalid recipient_type %q", s.RecipientType)
	}
	if !validComplexity(s.Complexity) {
		return fmt.Errorf("invalid complexity %q", s.Complexity)
	}
	if s.Confidence < 0 || s.Confidence > 1 {
		return fmt.Errorf("confidence %v outside [0,1]", s.Confidence)
	}
	for _, target := range s.Retrieval {
		if !validRetrieval(target) {
			return fmt.Errorf("invalid retrieval target %q", target)
		}
	}
	return nil
}

func (d TurnDecision) Validate() error {
	if d.SchemaVersion != SchemaVersion {
		return fmt.Errorf("turn decision schema_version=%d, want %d", d.SchemaVersion, SchemaVersion)
	}
	if !validInteraction(d.Interaction) || !validAction(d.RequestedAction) || !validAction(d.Action) {
		return fmt.Errorf("invalid decision interaction/action")
	}
	if !validMutation(d.Mutation) {
		return fmt.Errorf("invalid decision mutation %q", d.Mutation)
	}
	if !validDomain(d.Domain) || !validRecipient(d.RecipientType) || !validComplexity(d.Complexity) {
		return fmt.Errorf("invalid decision domain or recipient")
	}
	if d.Source != SourceStructural && d.Source != SourceLocalModel &&
		d.Source != SourceSafeFallback && d.Source != SourceLegacyRollback {
		return fmt.Errorf("invalid decision source %q", d.Source)
	}
	if d.Confidence < 0 || d.Confidence > 1 {
		return fmt.Errorf("decision confidence %v outside [0,1]", d.Confidence)
	}
	return nil
}

func validInteraction(value InteractionKind) bool {
	return slices.Contains([]InteractionKind{
		InteractionClosure, InteractionCasual, InteractionQuestion, InteractionTask,
		InteractionContinuation, InteractionCorrection,
	}, value)
}

func validAction(value Action) bool {
	return slices.Contains([]Action{
		ActionAnswer, ActionInspect, ActionPlan, ActionDebug, ActionEdit, ActionRun,
		ActionContinue, ActionArtifact, ActionImage, ActionAskUser,
	}, value)
}

func validMutation(value Mutation) bool {
	return value == MutationNone || value == MutationExternal || value == MutationWorkspace
}

func validRetrieval(value RetrievalTarget) bool {
	return slices.Contains([]RetrievalTarget{
		RetrievalMemory, RetrievalCodebase, RetrievalCodeGraph,
		RetrievalPriorReference, RetrievalCollaboration,
	}, value)
}

func validDomain(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	return slices.Contains([]string{
		"general", "security", "biology", "frontend", "backend", "devops",
		"architecture", "code_review", "database", "rust", "cad",
	}, value)
}

func validRecipient(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	return slices.Contains([]string{
		"general", "assistant", "frontend", "backend", "devops", "architecture",
		"code-review", "database", "security", "biology", "rust", "cad",
	}, value)
}

func validComplexity(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || value == "cheap" || value == "standard" || value == "heavy"
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
