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
	UserQuestionTTL        = 10 * time.Minute
	UserQuestionCleanupInt = 30 * time.Second
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
	waiters   map[string]chan string

	hub         *Hub
	stopCleanup chan struct{}
}

func NewUserQuestionManager(hub *Hub) *UserQuestionManager {
	uqm := &UserQuestionManager{
		questions:   make(map[string]*UserQuestion),
		waiters:     make(map[string]chan string),
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

	uqm.mu.Lock()
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
	uqm.waiters[q.ID] = make(chan string, 1)
	uqm.mu.Unlock()

	uqm.broadcastQuestion(q)

	uqm.mu.Lock()
	ch := uqm.waiters[q.ID]
	uqm.mu.Unlock()

	select {
	case answer := <-ch:
		return answer, nil
	case <-time.After(timeout):
		uqm.mu.Lock()
		if existing, ok := uqm.questions[q.ID]; ok && existing.Status == UserQuestionPending {
			now := time.Now()
			existing.Status = UserQuestionExpired
			existing.ResolvedAt = &now
			existing.Answer = "timed out waiting for user response"
		}
		delete(uqm.waiters, q.ID)
		uqm.mu.Unlock()
		uqm.broadcastQuestionUpdate(q.ID)
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

	if ch, ok := uqm.waiters[questionID]; ok {
		ch <- answer
		delete(uqm.waiters, questionID)
	}

	uqm.mu.Unlock()
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
	if err := uqm.hub.SendMessage(msg); err != nil {
		log.Printf("[UserQuestion] Failed to broadcast update %s: %v", q.ID, err)
	}
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
	defer uqm.mu.Unlock()
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
			if ch, ok := uqm.waiters[id]; ok {
				close(ch)
				delete(uqm.waiters, id)
			}
		}
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
