package hub

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

const (
	UserQuestionTTL         = 10 * time.Minute
	UserQuestionCleanupInt  = 30 * time.Second
	UserQuestionDedupWindow = 30 * time.Minute
)

type UserQuestionStatus string

const (
	UserQuestionPending  UserQuestionStatus = "pending"
	UserQuestionAnswered UserQuestionStatus = "answered"
	UserQuestionExpired  UserQuestionStatus = "expired"
)

// UserQuestion represents a pending question an agent needs the user to answer.
type UserQuestion struct {
	ID         string             `json:"id"`
	AgentID    string             `json:"agent_id"`
	AgentName  string             `json:"agent_name"`
	Channel    string             `json:"channel"`
	Question   string             `json:"question"`
	Options    []string           `json:"options,omitempty"`
	Status     UserQuestionStatus `json:"status"`
	Answer     string             `json:"answer,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	ResolvedAt *time.Time         `json:"resolved_at,omitempty"`
}

// UserQuestionManager manages agent-to-user question prompts.
type UserQuestionManager struct {
	mu        sync.Mutex
	questions map[string]*UserQuestion
	waiters   map[string][]chan string

	hub         *Hub
	stopCleanup chan struct{}
}

func NewUserQuestionManager(hub *Hub) *UserQuestionManager {
	uqm := &UserQuestionManager{
		questions:   make(map[string]*UserQuestion),
		waiters:     make(map[string][]chan string),
		hub:         hub,
		stopCleanup: make(chan struct{}),
	}
	go uqm.cleanupLoop()
	return uqm
}

func (uqm *UserQuestionManager) Stop() {
	close(uqm.stopCleanup)
}

// Ask creates a question, broadcasts it to the channel, and blocks until the user answers or timeout.
func (uqm *UserQuestionManager) Ask(agentID, agentName, channel, question string, options []string, timeout time.Duration) (string, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return "", fmt.Errorf("question is required")
	}
	if channel == "" {
		channel = "general"
	}
	if timeout <= 0 {
		timeout = UserQuestionTTL
	}

	waiter := make(chan string, 1)
	uqm.mu.Lock()

	// Reuse a recent answer to the same/similar question instead of re-prompting.
	if answer, ok := uqm.findRecentAnswerLocked(channel, question, time.Now()); ok {
		uqm.mu.Unlock()
		log.Printf("[UserQuestion] Reusing recent answer on %s for similar question from %s", channel, agentName)
		return answer, nil
	}

	// Coalesce concurrent equivalent pending questions. Every caller gets its own
	// waiter so one user answer resumes all agents without duplicate cards.
	for _, pending := range uqm.questions {
		if pending != nil && pending.Channel == channel && pending.Status == UserQuestionPending &&
			questionsSimilar(question, pending.Question) {
			uqm.waiters[pending.ID] = append(uqm.waiters[pending.ID], waiter)
			id := pending.ID
			uqm.mu.Unlock()
			log.Printf("[UserQuestion] Joining pending question %s on %s for %s", id, channel, agentName)
			return uqm.waitForAnswer(id, waiter, timeout)
		}
	}

	q := &UserQuestion{
		ID:        uuid.New().String()[:8],
		AgentID:   agentID,
		AgentName: agentName,
		Channel:   channel,
		Question:  question,
		Options:   cleanQuestionOptions(options),
		Status:    UserQuestionPending,
		CreatedAt: time.Now(),
	}
	uqm.questions[q.ID] = q
	uqm.waiters[q.ID] = []chan string{waiter}
	broadcastQ := *q
	broadcastQ.Options = append([]string(nil), q.Options...)
	uqm.mu.Unlock()

	// Pause peer agents before the question hits the timeline so they don't
	// race-reply to the ask_user card (or keep "responding" in the UI).
	if uqm.hub != nil {
		uqm.hub.pauseAgentsForUserQuestion(q.Channel, agentID)
	}

	uqm.broadcastQuestion(&broadcastQ)

	return uqm.waitForAnswer(q.ID, waiter, timeout)
}

func (uqm *UserQuestionManager) waitForAnswer(questionID string, waiter chan string, timeout time.Duration) (string, error) {
	select {
	case answer, ok := <-waiter:
		if !ok {
			return "", fmt.Errorf("timed out waiting for user response")
		}
		return answer, nil
	case <-time.After(timeout):
		uqm.mu.Lock()
		var waiters []chan string
		if existing, ok := uqm.questions[questionID]; ok && existing.Status == UserQuestionPending {
			now := time.Now()
			existing.Status = UserQuestionExpired
			existing.ResolvedAt = &now
			existing.Answer = "timed out waiting for user response"
			waiters = uqm.waiters[questionID]
			delete(uqm.waiters, questionID)
		}
		uqm.mu.Unlock()
		for _, ch := range waiters {
			close(ch)
		}
		uqm.broadcastQuestionUpdate(questionID)
		return "", fmt.Errorf("timed out waiting for user response")
	}
}

// Answer resolves a pending question with the user's response.
func (uqm *UserQuestionManager) Answer(questionID, answer string) error {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return fmt.Errorf("answer is required")
	}

	uqm.mu.Lock()

	q, ok := uqm.questions[questionID]
	if !ok {
		uqm.mu.Unlock()
		return fmt.Errorf("question not found: %s", questionID)
	}
	if q.Status != UserQuestionPending {
		uqm.mu.Unlock()
		return fmt.Errorf("question already resolved: %s", q.Status)
	}

	now := time.Now()
	q.Status = UserQuestionAnswered
	q.Answer = answer
	q.ResolvedAt = &now

	waiters := uqm.waiters[questionID]
	delete(uqm.waiters, questionID)

	uqm.mu.Unlock()
	for _, ch := range waiters {
		ch <- answer
		close(ch)
	}
	uqm.broadcastQuestionUpdate(questionID)
	return nil
}

// ListPending returns all pending user questions.
func (uqm *UserQuestionManager) ListPending() []*UserQuestion {
	uqm.mu.Lock()
	defer uqm.mu.Unlock()
	var pending []*UserQuestion
	for _, q := range uqm.questions {
		if q.Status == UserQuestionPending {
			pending = append(pending, q)
		}
	}
	return pending
}

// HasPendingOnChannel reports whether any ask_user question is still awaiting an answer.
func (uqm *UserQuestionManager) HasPendingOnChannel(channel string) bool {
	if uqm == nil || channel == "" {
		return false
	}
	uqm.mu.Lock()
	defer uqm.mu.Unlock()
	for _, q := range uqm.questions {
		if q.Status == UserQuestionPending && q.Channel == channel {
			return true
		}
	}
	return false
}

// PendingAgentIDsOnChannel returns agent IDs that currently have a pending question on channel.
func (uqm *UserQuestionManager) PendingAgentIDsOnChannel(channel string) []string {
	if uqm == nil || channel == "" {
		return nil
	}
	uqm.mu.Lock()
	defer uqm.mu.Unlock()
	seen := make(map[string]bool)
	var ids []string
	for _, q := range uqm.questions {
		if q.Status != UserQuestionPending || q.Channel != channel || q.AgentID == "" {
			continue
		}
		if seen[q.AgentID] {
			continue
		}
		seen[q.AgentID] = true
		ids = append(ids, q.AgentID)
	}
	return ids
}

func (uqm *UserQuestionManager) broadcastQuestion(q *UserQuestion) {
	msg := &protocol.Message{
		ID:      uuid.New().String(),
		Type:    protocol.MessageTypeUserQuestion,
		Channel: q.Channel,
		From: protocol.AgentInfo{
			ID:   q.AgentID,
			Name: q.AgentName,
			Type: protocol.AgentTypeGeneral,
		},
		Content:   fmt.Sprintf("**%s** has a question:\n\n%s", q.AgentName, q.Question),
		Timestamp: q.CreatedAt,
		Metadata: map[string]interface{}{
			"question_id": q.ID,
			"question":    q.Question,
			"options":     q.Options,
			"status":      string(q.Status),
		},
	}
	if err := uqm.hub.SendMessage(msg); err != nil {
		log.Printf("[UserQuestion] Failed to broadcast question %s: %v", q.ID, err)
	}
}

func (uqm *UserQuestionManager) broadcastQuestionUpdate(questionID string) {
	uqm.mu.Lock()
	q, ok := uqm.questions[questionID]
	uqm.mu.Unlock()
	if !ok {
		return
	}

	content := fmt.Sprintf("Question **%s**", q.Status)
	if q.Status == UserQuestionAnswered {
		content = fmt.Sprintf("**Answer:** %s", q.Answer)
	}

	msg := &protocol.Message{
		ID:      uuid.New().String(),
		Type:    protocol.MessageTypeUserQuestion,
		Channel: q.Channel,
		From: protocol.AgentInfo{
			ID:   q.AgentID,
			Name: q.AgentName,
			Type: protocol.AgentTypeGeneral,
		},
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"question_id": q.ID,
			"question":    q.Question,
			"options":     q.Options,
			"status":      string(q.Status),
			"answer":      q.Answer,
		},
	}
	if uqm.hub != nil && uqm.hub.upsertUserQuestionMessage(msg) {
		return
	}
	if uqm.hub == nil {
		return
	}
	if err := uqm.hub.SendMessage(msg); err != nil {
		log.Printf("[UserQuestion] Failed to broadcast update %s: %v", q.ID, err)
	}
}

// findRecentAnswer returns a prior answer when the same/similar question was
// already resolved on this channel within UserQuestionDedupWindow.
func (uqm *UserQuestionManager) findRecentAnswer(channel, question string) (string, bool) {
	if uqm == nil {
		return "", false
	}
	uqm.mu.Lock()
	defer uqm.mu.Unlock()
	return uqm.findRecentAnswerLocked(channel, question, time.Now())
}

func (uqm *UserQuestionManager) findRecentAnswerLocked(channel, question string, now time.Time) (string, bool) {
	var bestAnswer string
	var bestAt time.Time
	found := false
	for _, q := range uqm.questions {
		if q == nil || q.Channel != channel || q.Status != UserQuestionAnswered {
			continue
		}
		if strings.TrimSpace(q.Answer) == "" || q.Answer == "timed out waiting for user response" {
			continue
		}
		resolved := q.CreatedAt
		if q.ResolvedAt != nil {
			resolved = *q.ResolvedAt
		}
		if now.Sub(resolved) > UserQuestionDedupWindow {
			continue
		}
		if !questionsSimilar(question, q.Question) {
			continue
		}
		if !found || resolved.After(bestAt) {
			bestAnswer = q.Answer
			bestAt = resolved
			found = true
		}
	}
	if !found {
		return "", false
	}
	return bestAnswer, true
}

func questionsSimilar(a, b string) bool {
	na, nb := normalizeUserQuestion(a), normalizeUserQuestion(b)
	if na == "" || nb == "" {
		return false
	}
	if na == nb {
		return true
	}
	if strings.Contains(na, nb) || strings.Contains(nb, na) {
		return true
	}
	// Common preference prompts that models rephrase slightly.
	if looksLikePlatformQuestion(na) && looksLikePlatformQuestion(nb) {
		return true
	}
	return false
}

func looksLikePlatformQuestion(norm string) bool {
	return strings.Contains(norm, "platform") &&
		(strings.Contains(norm, "desktop") || strings.Contains(norm, "web") ||
			strings.Contains(norm, "mobile") || strings.Contains(norm, "target"))
}

func normalizeUserQuestion(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if !prevSpace {
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func (uqm *UserQuestionManager) cleanupLoop() {
	ticker := time.NewTicker(UserQuestionCleanupInt)
	defer ticker.Stop()
	for {
		select {
		case <-uqm.stopCleanup:
			return
		case <-ticker.C:
			uqm.expireStale()
		}
	}
}

func (uqm *UserQuestionManager) expireStale() {
	now := time.Now()
	uqm.mu.Lock()
	var expiredIDs []string
	var expiredWaiters []chan string
	for id, q := range uqm.questions {
		if q.Status != UserQuestionPending {
			if q.ResolvedAt != nil && now.Sub(*q.ResolvedAt) > UserQuestionTTL*2 {
				delete(uqm.questions, id)
			}
			continue
		}
		if now.Sub(q.CreatedAt) > UserQuestionTTL {
			q.Status = UserQuestionExpired
			q.ResolvedAt = &now
			q.Answer = "timed out waiting for user response"
			if waiters, ok := uqm.waiters[id]; ok {
				expiredWaiters = append(expiredWaiters, waiters...)
				delete(uqm.waiters, id)
			}
			expiredIDs = append(expiredIDs, id)
		}
	}
	uqm.mu.Unlock()
	for _, ch := range expiredWaiters {
		close(ch)
	}
	for _, id := range expiredIDs {
		uqm.broadcastQuestionUpdate(id)
	}
}

func cleanQuestionOptions(options []string) []string {
	if len(options) == 0 {
		return nil
	}
	out := make([]string, 0, len(options))
	seen := make(map[string]bool)
	for _, opt := range options {
		opt = strings.TrimSpace(opt)
		if opt == "" || seen[opt] {
			continue
		}
		seen[opt] = true
		out = append(out, opt)
	}
	return out
}
