package hub

import (
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// EditorSessionTurn is one user/assistant exchange in an editor agent session.
type EditorSessionTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	At      string `json:"at"`
}

// EditorSession state for workspace-bound editor agent (v3).
type EditorSession struct {
	SessionID   string              `json:"session_id"`
	WorkspaceID string              `json:"workspace_id"`
	Channel     string              `json:"channel"`
	AgentType   string              `json:"agent_type"`
	Mode        string              `json:"mode"`
	Turns       []EditorSessionTurn `json:"turns"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// EditorSessionStore holds in-memory editor sessions.
type EditorSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*EditorSession // key: session_id
	byWS     map[string]string         // workspace_id -> latest session_id
}

func NewEditorSessionStore() *EditorSessionStore {
	return &EditorSessionStore{
		sessions: make(map[string]*EditorSession),
		byWS:     make(map[string]string),
	}
}

func (s *EditorSessionStore) GetOrCreate(workspaceID, channel, agentType, mode, sessionID string) *EditorSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sessionID != "" {
		if ex, ok := s.sessions[sessionID]; ok {
			return ex
		}
	}
	if sessionID == "" {
		if sid, ok := s.byWS[workspaceID]; ok {
			if ex, ok := s.sessions[sid]; ok {
				return ex
			}
		}
		sessionID = uuid.New().String()
	}
	sess := &EditorSession{
		SessionID:   sessionID,
		WorkspaceID: workspaceID,
		Channel:     channel,
		AgentType:   agentType,
		Mode:        mode,
		Turns:       nil,
		UpdatedAt:   time.Now(),
	}
	s.sessions[sessionID] = sess
	s.byWS[workspaceID] = sessionID
	return sess
}

func (s *EditorSessionStore) AppendTurn(sessionID, role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		return
	}
	sess.Turns = append(sess.Turns, EditorSessionTurn{
		Role: role, Content: content, At: time.Now().UTC().Format(time.RFC3339),
	})
	if len(sess.Turns) > 24 {
		sess.Turns = sess.Turns[len(sess.Turns)-24:]
	}
	sess.UpdatedAt = time.Now()
}

func (s *EditorSessionStore) HistoryMessages(sessionID string) []*protocol.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	var out []*protocol.Message
	for _, t := range sess.Turns {
		from := protocol.AgentInfo{ID: "human", Name: "Developer", Type: "human"}
		if t.Role == "assistant" {
			from = protocol.AgentInfo{ID: "editor-agent", Name: "Editor Agent", Type: protocol.AgentTypeGeneral}
		}
		m := protocol.NewMessage(protocol.MessageTypeChat, sess.Channel, from, t.Content)
		out = append(out, m)
	}
	return out
}
