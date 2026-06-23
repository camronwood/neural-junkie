package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	_ "modernc.org/sqlite"
)

// Store persists chat messages in SQLite.
type Store struct {
	mu sync.RWMutex
	db *sql.DB
}

// Open creates or opens ~/.neural-junkie/messages.db
func Open(path string) (*Store, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".neural-junkie", "messages.db")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  channel TEXT NOT NULL,
  thread_id TEXT NOT NULL DEFAULT '',
  sender_id TEXT,
  sender_name TEXT,
  sender_type TEXT,
  content TEXT NOT NULL,
  msg_type TEXT NOT NULL,
  metadata_json TEXT,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_channel_created ON messages(channel, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_thread_created ON messages(thread_id, created_at);
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
  message_id UNINDEXED,
  channel UNINDEXED,
  content,
  sender_name,
  tokenize='unicode61'
);
`)
	if err != nil {
		return err
	}
	return s.backfillFTS()
}

func (s *Store) Close() error {
	return s.db.Close()
}

// InsertMessage persists a chat message.
func (s *Store) InsertMessage(msg *protocol.Message) error {
	if msg == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, _ := json.Marshal(msg.Metadata)
	threadID := msg.GetThreadID()
	created := msg.Timestamp.UnixMilli()
	if msg.Timestamp.IsZero() {
		created = time.Now().UnixMilli()
	}
	_, err := s.db.Exec(`INSERT OR REPLACE INTO messages
(id, channel, thread_id, sender_id, sender_name, sender_type, content, msg_type, metadata_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.Channel, threadID, msg.From.ID, msg.From.Name, string(msg.From.Type),
		msg.Content, string(msg.Type), string(meta), created)
	if err != nil {
		return err
	}
	return s.indexMessageFTS(msg.ID, msg.Channel, msg.Content, msg.From.Name)
}

// ListChannelMessages returns messages for a channel with optional cursor pagination.
func (s *Store) ListChannelMessages(channel string, limit int, beforeID string) ([]*protocol.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := `SELECT id, channel, thread_id, sender_id, sender_name, sender_type, content, msg_type, metadata_json, created_at
FROM messages WHERE channel = ? AND thread_id = ''`
	args := []any{channel}
	if beforeID != "" {
		var beforeTS int64
		if err := s.db.QueryRow(`SELECT created_at FROM messages WHERE id = ?`, beforeID).Scan(&beforeTS); err == nil {
			query += ` AND created_at < ?`
			args = append(args, beforeTS)
		}
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*protocol.Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	// reverse to chronological
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMessage(rows rowScanner) (*protocol.Message, error) {
	var id, channel, threadID, senderID, senderName, senderType, content, msgType, metaJSON string
	var created int64
	if err := rows.Scan(&id, &channel, &threadID, &senderID, &senderName, &senderType, &content, &msgType, &metaJSON, &created); err != nil {
		return nil, err
	}
	msg := &protocol.Message{
		ID:      id,
		Channel: channel,
		Content: content,
		Type:    protocol.MessageType(msgType),
		From: protocol.AgentInfo{
			ID:   senderID,
			Name: senderName,
			Type: protocol.AgentType(senderType),
		},
		Timestamp: time.UnixMilli(created),
	}
	if threadID != "" {
		msg.ThreadID = threadID
		msg.IsThreadReply = true
	}
	if metaJSON != "" {
		_ = json.Unmarshal([]byte(metaJSON), &msg.Metadata)
	}
	return msg, nil
}

// LoadRecentChannel loads the most recent N messages into memory on hub startup.
func (s *Store) LoadRecentChannel(channel string, limit int) ([]*protocol.Message, error) {
	return s.ListChannelMessages(channel, limit, "")
}

// ClearChannelMessages removes all persisted rows for a channel.
func (s *Store) ClearChannelMessages(channel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM messages WHERE channel = ?`, channel)
	return err
}

// DeleteChannel removes all messages for a channel.
func (s *Store) DeleteChannel(channel string) error {
	return s.ClearChannelMessages(channel)
}

// Stats returns message count for a channel.
func (s *Store) Stats(channel string) (int, error) {
	s.mu.RLock()
	defer s.mu.Unlock()
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE channel = ?`, channel).Scan(&n)
	return n, err
}

// PathDefault returns default DB path.
func PathDefault() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "messages.db"), nil
}

// ErrNotFound is returned when a message id is missing.
var ErrNotFound = fmt.Errorf("message not found")
