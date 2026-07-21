package hub

import (
	"strings"
	"time"
)

// ChannelConversationState is durable, structured state for one channel.
// IDs tie derived state back to the messages and goals that produced it.
type ChannelConversationState struct {
	CurrentGoal            *ConversationGoal                `json:"current_goal,omitempty"`
	AnsweredDecisions      map[string]ConversationDecision  `json:"answered_decisions,omitempty"`
	Actions                map[string]ConversationAction    `json:"actions,omitempty"`
	Corrections            []ConversationCorrection         `json:"corrections,omitempty"`
	SupersededInstructions map[string]SupersededInstruction `json:"superseded_instructions,omitempty"`
	UpdatedAt              time.Time                        `json:"updated_at,omitempty"`
}

type ConversationGoal struct {
	ID            string    `json:"id"`
	MessageID     string    `json:"message_id,omitempty"`
	LastMessageID string    `json:"last_message_id,omitempty"`
	Text          string    `json:"text,omitempty"`
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
func (h *Hub) SetCurrentGoal(channel, goalID, messageID, text string) {
	if h == nil || strings.TrimSpace(channel) == "" || strings.TrimSpace(goalID) == "" {
		return
	}
	now := time.Now()
	h.mu.Lock()
	state := h.conversationStateLocked(channel)
	goalID = strings.TrimSpace(goalID)
	messageID = strings.TrimSpace(messageID)
	if state.CurrentGoal != nil && state.CurrentGoal.ID == goalID {
		state.CurrentGoal.LastMessageID = messageID
		state.CurrentGoal.UpdatedAt = now
	} else {
		state.CurrentGoal = &ConversationGoal{
			ID: goalID, MessageID: messageID, LastMessageID: messageID,
			Text: strings.TrimSpace(text), UpdatedAt: now,
		}
	}
	state.UpdatedAt = now
	h.mu.Unlock()
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
	state.UpdatedAt = now
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

func (h *Hub) RecordConversationActionPromise(channel, actionID, goalID, description, messageID string) {
	h.RecordPromisedAction(channel, ConversationAction{
		ID: actionID, GoalID: goalID, Description: description,
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
	return &out
}
