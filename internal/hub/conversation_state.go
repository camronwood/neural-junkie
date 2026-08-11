package hub

import (
	"sort"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/turnledger"
)

// ChannelConversationState is durable, structured state for one channel.
// IDs tie derived state back to the messages and goals that produced it.
type ChannelConversationState struct {
	CurrentGoal            *ConversationGoal                `json:"current_goal,omitempty"`
	AnsweredDecisions      map[string]ConversationDecision  `json:"answered_decisions,omitempty"`
	Actions                map[string]ConversationAction    `json:"actions,omitempty"`
	Corrections            []ConversationCorrection         `json:"corrections,omitempty"`
	SupersededInstructions map[string]SupersededInstruction `json:"superseded_instructions,omitempty"`
	OpenQuestions          []ConversationOpenQuestion       `json:"open_questions,omitempty"`
	NamedEntities          []ConversationNamedEntity        `json:"named_entities,omitempty"`
	UpdatedAt              time.Time                        `json:"updated_at,omitempty"`
}

type ConversationGoal struct {
	ID            string    `json:"id"`
	MessageID     string    `json:"message_id,omitempty"`
	LastMessageID string    `json:"last_message_id,omitempty"`
	Text          string    `json:"text,omitempty"`
	PinnedText    string    `json:"pinned_text,omitempty"` // original user task; never overwritten on refresh
	UpdatedAt     time.Time `json:"updated_at"`
}

type ConversationDecision struct {
	GoalID      string    `json:"goal_id"`
	DecisionKey string    `json:"decision_key"`
	QuestionID  string    `json:"question_id,omitempty"`
	MessageID   string    `json:"message_id,omitempty"`
	Answer      string    `json:"answer"`
	AnsweredAt  time.Time `json:"answered_at"`
}

type ConversationAction struct {
	ID                   string     `json:"id"`
	GoalID               string     `json:"goal_id,omitempty"`
	Action               string     `json:"action,omitempty"`
	Description          string     `json:"description"`
	PromisedMessageID    string     `json:"promised_message_id,omitempty"`
	LastPromiseMessageID string     `json:"last_promise_message_id,omitempty"`
	CompletedMessageID   string     `json:"completed_message_id,omitempty"`
	PromisedAt           time.Time  `json:"promised_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
}

type ConversationCorrection struct {
	GoalID               string    `json:"goal_id,omitempty"`
	MessageID            string    `json:"message_id"`
	Instruction          string    `json:"instruction"`
	SupersedesMessageIDs []string  `json:"supersedes_message_ids,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

type SupersededInstruction struct {
	MessageID             string    `json:"message_id"`
	SupersededByMessageID string    `json:"superseded_by_message_id"`
	GoalID                string    `json:"goal_id,omitempty"`
	SupersededAt          time.Time `json:"superseded_at"`
}

type ConversationOpenQuestion struct {
	ID        string `json:"id"`
	Text      string `json:"text,omitempty"`
	GoalID    string `json:"goal_id,omitempty"`
	MessageID string `json:"message_id,omitempty"`
}

type ConversationNamedEntity struct {
	Name              string `json:"name"`
	Kind              string `json:"kind,omitempty"`
	LastSeenMessageID string `json:"last_seen_message_id,omitempty"`
}

const (
	maxNamedEntities = 8
	maxOpenQuestions = 4
)

func newChannelConversationState() *ChannelConversationState {
	return &ChannelConversationState{
		AnsweredDecisions:      make(map[string]ConversationDecision),
		Actions:                make(map[string]ConversationAction),
		SupersededInstructions: make(map[string]SupersededInstruction),
	}
}

func decisionStateKey(goalID, decisionKey string) string {
	return strings.TrimSpace(goalID) + "\x00" + strings.TrimSpace(decisionKey)
}

func (h *Hub) conversationStateLocked(channel string) *ChannelConversationState {
	if h.conversationState == nil {
		h.conversationState = make(map[string]*ChannelConversationState)
	}
	state := h.conversationState[channel]
	if state == nil {
		state = newChannelConversationState()
		h.conversationState[channel] = state
	}
	if state.AnsweredDecisions == nil {
		state.AnsweredDecisions = make(map[string]ConversationDecision)
	}
	if state.Actions == nil {
		state.Actions = make(map[string]ConversationAction)
	}
	if state.SupersededInstructions == nil {
		state.SupersededInstructions = make(map[string]SupersededInstruction)
	}
	return state
}

// SetCurrentGoal records a new goal, or refreshes the same goal after an
// approval/continuation message. An empty goal ID is ignored.
// PinnedText is set once on first create and never overwritten on refresh so the
// original user task survives later affirmations ("ok", "fix the app").
func (h *Hub) SetCurrentGoal(channel, goalID, messageID, text string) {
	if h == nil || strings.TrimSpace(channel) == "" || strings.TrimSpace(goalID) == "" {
		return
	}
	now := time.Now()
	h.mu.Lock()
	state := h.conversationStateLocked(channel)
	goalID = strings.TrimSpace(goalID)
	messageID = strings.TrimSpace(messageID)
	text = strings.TrimSpace(text)
	if state.CurrentGoal != nil && state.CurrentGoal.ID == goalID {
		state.CurrentGoal.LastMessageID = messageID
		state.CurrentGoal.UpdatedAt = now
		if state.CurrentGoal.PinnedText == "" {
			if state.CurrentGoal.Text != "" {
				state.CurrentGoal.PinnedText = state.CurrentGoal.Text
			} else if text != "" {
				state.CurrentGoal.PinnedText = text
			}
		}
	} else {
		pinnedText := text
		pinnedMessageID := messageID
		if state.CurrentGoal != nil && shouldRetainPinnedTask(state.CurrentGoal.PinnedText, text) {
			pinnedText = state.CurrentGoal.PinnedText
			if state.CurrentGoal.MessageID != "" {
				pinnedMessageID = state.CurrentGoal.MessageID
			}
		}
		state.CurrentGoal = &ConversationGoal{
			ID: goalID, MessageID: pinnedMessageID, LastMessageID: messageID,
			Text: text, PinnedText: pinnedText, UpdatedAt: now,
		}
	}
	rememberEntitiesLocked(state, messageID, text)
	rememberOpenQuestionLocked(state, goalID, messageID, text)
	state.UpdatedAt = now
	h.mu.Unlock()
}

// shouldRetainPinnedTask keeps the original user task across goal-id churn for
// short fix/continue follow-ups so history eviction cannot drop the real ask.
func shouldRetainPinnedTask(prev, next string) bool {
	prev = strings.TrimSpace(prev)
	if prev == "" {
		return false
	}
	next = strings.TrimSpace(next)
	if next == "" || strings.EqualFold(prev, next) {
		return true
	}
	lower := strings.ToLower(next)
	if len(strings.Fields(next)) > 16 {
		return false
	}
	retainCues := []string{
		"fix the", "fix it", "fix this", "repair", "continue", "keep going",
		"yes", "ok ", "okay", "please", "do that", "go ahead", "can you",
	}
	for _, cue := range retainCues {
		if strings.Contains(lower, cue) || lower == "ok" || lower == "yes" {
			return true
		}
	}
	return false
}

// PersistConversationGoal is the primitive-argument wrapper used by agent
// packages without importing hub-owned state types.
func (h *Hub) PersistConversationGoal(channel, goalID, messageID, text string) {
	h.SetCurrentGoal(channel, goalID, messageID, text)
}

// ResolveConversationGoalID preserves an explicit/original goal ID for
// continuations, otherwise returning the channel's current goal.
func (h *Hub) ResolveConversationGoalID(channel, explicitGoalID string) string {
	if id := strings.TrimSpace(explicitGoalID); id != "" {
		return id
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if state := h.conversationState[channel]; state != nil && state.CurrentGoal != nil {
		return state.CurrentGoal.ID
	}
	return ""
}

// RecordConversationDecision stores a resolved decision under its goal/key.
func (h *Hub) RecordConversationDecision(channel string, decision ConversationDecision) {
	if h == nil || channel == "" || decision.DecisionKey == "" {
		return
	}
	if decision.AnsweredAt.IsZero() {
		decision.AnsweredAt = time.Now()
	}
	h.mu.Lock()
	state := h.conversationStateLocked(channel)
	state.AnsweredDecisions[decisionStateKey(decision.GoalID, decision.DecisionKey)] = decision
	state.UpdatedAt = decision.AnsweredAt
	h.mu.Unlock()
}

// RecordConversationCorrection records a correction and atomically marks the
// referenced older instructions as superseded.
func (h *Hub) RecordConversationCorrection(channel, goalID, messageID, instruction string, supersedesMessageIDs []string) {
	if h == nil || channel == "" || strings.TrimSpace(messageID) == "" {
		return
	}
	now := time.Now()
	correction := ConversationCorrection{
		GoalID: strings.TrimSpace(goalID), MessageID: strings.TrimSpace(messageID),
		Instruction: strings.TrimSpace(instruction), CreatedAt: now,
	}
	h.mu.Lock()
	state := h.conversationStateLocked(channel)
	for _, existing := range state.Corrections {
		if existing.MessageID == correction.MessageID {
			h.mu.Unlock()
			return
		}
	}
	for _, priorID := range supersedesMessageIDs {
		priorID = strings.TrimSpace(priorID)
		if priorID == "" || priorID == correction.MessageID {
			continue
		}
		correction.SupersedesMessageIDs = append(correction.SupersedesMessageIDs, priorID)
		state.SupersededInstructions[priorID] = SupersededInstruction{
			MessageID: priorID, SupersededByMessageID: correction.MessageID,
			GoalID: correction.GoalID, SupersededAt: now,
		}
	}
	state.Corrections = append(state.Corrections, correction)
	rememberEntitiesLocked(state, correction.MessageID, correction.Instruction)
	state.UpdatedAt = now
	h.mu.Unlock()
}

// RememberConversationSurface seeds named entities and open questions from a
// user turn without requiring a new goal. Safe to call from the turn ledger path.
func (h *Hub) RememberConversationSurface(channel, goalID, messageID, text string) {
	if h == nil || strings.TrimSpace(channel) == "" || strings.TrimSpace(text) == "" {
		return
	}
	h.mu.Lock()
	state := h.conversationStateLocked(channel)
	rememberEntitiesLocked(state, messageID, text)
	rememberOpenQuestionLocked(state, goalID, messageID, text)
	state.UpdatedAt = time.Now()
	h.mu.Unlock()
}

func (h *Hub) RecordPromisedAction(channel string, action ConversationAction) {
	if h == nil || channel == "" || strings.TrimSpace(action.ID) == "" {
		return
	}
	if action.PromisedAt.IsZero() {
		action.PromisedAt = time.Now()
	}
	h.mu.Lock()
	state := h.conversationStateLocked(channel)
	if existing, ok := state.Actions[action.ID]; ok {
		// A new correction/approval turn reopens the same goal action while
		// retaining the original promise correlation.
		if existing.GoalID == "" {
			existing.GoalID = action.GoalID
		}
		if existing.Description == "" {
			existing.Description = action.Description
		}
		if action.Action != "" {
			existing.Action = action.Action
		}
		if action.PromisedMessageID != "" && action.PromisedMessageID != existing.LastPromiseMessageID {
			existing.LastPromiseMessageID = action.PromisedMessageID
			existing.CompletedAt = nil
			existing.CompletedMessageID = ""
		}
		state.Actions[action.ID] = existing
	} else {
		action.LastPromiseMessageID = action.PromisedMessageID
		state.Actions[action.ID] = action
	}
	state.UpdatedAt = action.PromisedAt
	h.mu.Unlock()
}

func (h *Hub) RecordConversationActionPromise(channel, actionID, goalID, action, description, messageID string) {
	h.RecordPromisedAction(channel, ConversationAction{
		ID: actionID, GoalID: goalID, Action: action, Description: description,
		PromisedMessageID: messageID,
	})
}

func (h *Hub) CompletePromisedAction(channel, actionID, messageID string) bool {
	if h == nil || channel == "" || actionID == "" {
		return false
	}
	now := time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	state := h.conversationStateLocked(channel)
	action, ok := state.Actions[actionID]
	if !ok {
		return false
	}
	action.CompletedAt = &now
	action.CompletedMessageID = strings.TrimSpace(messageID)
	state.Actions[actionID] = action
	state.UpdatedAt = now
	return true
}

func (h *Hub) CompleteConversationAction(channel, actionID, messageID string) bool {
	return h.CompletePromisedAction(channel, actionID, messageID)
}

// GetChannelConversationState returns an isolated copy safe for callers to mutate.
func (h *Hub) GetChannelConversationState(channel string) *ChannelConversationState {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return cloneConversationState(h.conversationState[channel])
}

// GetTurnConversationContext exposes a transport-safe snapshot to the agent
// pipeline without making agent depend on hub-owned persistence types.
func (h *Hub) GetTurnConversationContext(channel string) protocol.TurnContextEnvelope {
	state := h.GetChannelConversationState(channel)
	out := protocol.TurnContextEnvelope{
		Version:        1,
		Channel:        channel,
		SectionBudgets: map[string]int{"durable_state": 6000, "recent_exchanges": 8000},
	}
	if state == nil {
		return out
	}
	if state.CurrentGoal != nil {
		out.Goal = &protocol.TurnContextGoal{
			ID: state.CurrentGoal.ID, MessageID: state.CurrentGoal.MessageID,
			LastMessageID: state.CurrentGoal.LastMessageID, Text: state.CurrentGoal.Text,
			PinnedText: state.CurrentGoal.PinnedText,
		}
		out.Provenance = append(out.Provenance, protocol.TurnContextProvenance{
			ID: state.CurrentGoal.MessageID, Section: "goal", Source: "channel_conversation_state",
		})
	}
	decisionKeys := make([]string, 0, len(state.AnsweredDecisions))
	for key := range state.AnsweredDecisions {
		decisionKeys = append(decisionKeys, key)
	}
	sort.Strings(decisionKeys)
	for _, key := range decisionKeys {
		d := state.AnsweredDecisions[key]
		out.Decisions = append(out.Decisions, protocol.TurnContextDecision{
			GoalID: d.GoalID, DecisionKey: d.DecisionKey, MessageID: d.MessageID,
			Answer: d.Answer, AnsweredAt: d.AnsweredAt,
		})
	}
	actionKeys := make([]string, 0, len(state.Actions))
	for key, action := range state.Actions {
		if action.CompletedAt == nil {
			actionKeys = append(actionKeys, key)
		}
	}
	sort.Strings(actionKeys)
	for _, key := range actionKeys {
		action := state.Actions[key]
		out.UnresolvedActions = append(out.UnresolvedActions, protocol.TurnContextAction{
			ID: action.ID, GoalID: action.GoalID, Action: action.Action, Description: action.Description,
			LastPromiseMessageID: action.LastPromiseMessageID, PromisedAt: action.PromisedAt,
		})
	}
	for _, correction := range state.Corrections {
		out.Corrections = append(out.Corrections, protocol.TurnContextCorrection{
			GoalID: correction.GoalID, MessageID: correction.MessageID,
			Instruction:          correction.Instruction,
			SupersedesMessageIDs: append([]string(nil), correction.SupersedesMessageIDs...),
		})
	}
	for id := range state.SupersededInstructions {
		out.SupersededMessageIDs = append(out.SupersededMessageIDs, id)
	}
	sort.Strings(out.SupersededMessageIDs)
	for _, q := range state.OpenQuestions {
		out.OpenQuestions = append(out.OpenQuestions, protocol.TurnContextOpenQuestion{
			ID: q.ID, Text: q.Text, GoalID: q.GoalID, MessageID: q.MessageID,
		})
	}
	for _, e := range state.NamedEntities {
		out.NamedEntities = append(out.NamedEntities, protocol.TurnContextNamedEntity{
			Name: e.Name, Kind: e.Kind, LastSeenMessageID: e.LastSeenMessageID,
		})
	}
	return out
}

func cloneConversationState(state *ChannelConversationState) *ChannelConversationState {
	if state == nil {
		return nil
	}
	out := *state
	if state.CurrentGoal != nil {
		goal := *state.CurrentGoal
		out.CurrentGoal = &goal
	}
	out.AnsweredDecisions = make(map[string]ConversationDecision, len(state.AnsweredDecisions))
	for key, value := range state.AnsweredDecisions {
		out.AnsweredDecisions[key] = value
	}
	out.Actions = make(map[string]ConversationAction, len(state.Actions))
	for key, value := range state.Actions {
		if value.CompletedAt != nil {
			completed := *value.CompletedAt
			value.CompletedAt = &completed
		}
		out.Actions[key] = value
	}
	out.Corrections = make([]ConversationCorrection, len(state.Corrections))
	for i, correction := range state.Corrections {
		out.Corrections[i] = correction
		out.Corrections[i].SupersedesMessageIDs = append([]string(nil), correction.SupersedesMessageIDs...)
	}
	out.SupersededInstructions = make(map[string]SupersededInstruction, len(state.SupersededInstructions))
	for key, value := range state.SupersededInstructions {
		out.SupersededInstructions[key] = value
	}
	out.OpenQuestions = append([]ConversationOpenQuestion(nil), state.OpenQuestions...)
	out.NamedEntities = append([]ConversationNamedEntity(nil), state.NamedEntities...)
	return &out
}

func rememberEntitiesLocked(state *ChannelConversationState, messageID, text string) {
	if state == nil {
		return
	}
	for _, name := range turnledger.ExtractEntities(text) {
		upsertNamedEntityLocked(state, name, "mention", messageID)
	}
}

func upsertNamedEntityLocked(state *ChannelConversationState, name, kind, messageID string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	for i := range state.NamedEntities {
		if strings.EqualFold(state.NamedEntities[i].Name, name) {
			state.NamedEntities[i].LastSeenMessageID = strings.TrimSpace(messageID)
			if kind != "" {
				state.NamedEntities[i].Kind = kind
			}
			return
		}
	}
	state.NamedEntities = append(state.NamedEntities, ConversationNamedEntity{
		Name: name, Kind: kind, LastSeenMessageID: strings.TrimSpace(messageID),
	})
	if len(state.NamedEntities) > maxNamedEntities {
		state.NamedEntities = state.NamedEntities[len(state.NamedEntities)-maxNamedEntities:]
	}
}

func rememberOpenQuestionLocked(state *ChannelConversationState, goalID, messageID, text string) {
	if state == nil || !looksLikeOpenQuestion(text) {
		return
	}
	text = strings.TrimSpace(text)
	for _, existing := range state.OpenQuestions {
		if strings.EqualFold(existing.Text, text) || existing.MessageID == messageID {
			return
		}
	}
	id := strings.TrimSpace(messageID)
	if id == "" {
		id = "q-" + strings.ReplaceAll(strings.ToLower(text[:min(12, len(text))]), " ", "-")
	}
	state.OpenQuestions = append(state.OpenQuestions, ConversationOpenQuestion{
		ID: id, Text: text, GoalID: strings.TrimSpace(goalID), MessageID: strings.TrimSpace(messageID),
	})
	if len(state.OpenQuestions) > maxOpenQuestions {
		state.OpenQuestions = state.OpenQuestions[len(state.OpenQuestions)-maxOpenQuestions:]
	}
}

func looksLikeOpenQuestion(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) < 8 {
		return false
	}
	lower := strings.ToLower(text)
	if strings.Contains(text, "?") {
		return true
	}
	for _, prefix := range []string{"should we ", "which ", "what about ", "do we still "} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
