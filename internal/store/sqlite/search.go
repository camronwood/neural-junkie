package sqlite

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// SearchOptions configures archive search.
type SearchOptions struct {
	Channel string
	Query   string
	Limit   int
	Before  int64 // created_at millis upper bound (exclusive)
}

// SearchWithOptions performs FTS5 search over persisted messages.
func (s *Store) SearchWithOptions(opts SearchOptions) ([]*protocol.Message, error) {
	q := strings.TrimSpace(opts.Query)
	if q == "" {
		return nil, fmt.Errorf("query required")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	ftsQuery := buildFTSQuery(q)
	s.mu.RLock()
	defer s.mu.RUnlock()

	sqlText := `
SELECT m.id, m.channel, m.thread_id, m.sender_id, m.sender_name, m.sender_type,
       m.content, m.msg_type, m.metadata_json, m.created_at
FROM messages_fts f
JOIN messages m ON m.id = f.message_id
WHERE messages_fts MATCH ?
`
	args := []any{ftsQuery}
	if ch := strings.TrimSpace(opts.Channel); ch != "" {
		sqlText += ` AND f.channel = ?`
		args = append(args, ch)
	}
	if opts.Before > 0 {
		sqlText += ` AND m.created_at < ?`
		args = append(args, opts.Before)
	}
	sqlText += ` ORDER BY m.created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(sqlText, args...)
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
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func buildFTSQuery(q string) string {
	parts := strings.Fields(q)
	if len(parts) == 0 {
		return `""`
	}
	escaped := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ReplaceAll(p, `"`, `""`)
		escaped = append(escaped, `"`+p+`"`)
	}
	return strings.Join(escaped, " AND ")
}

func (s *Store) indexMessageFTS(msgID, channel, content, senderName string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO messages_fts(message_id, channel, content, sender_name)
VALUES (?, ?, ?, ?)`, msgID, channel, content, senderName)
	return err
}

func (s *Store) backfillFTS() error {
	rows, err := s.db.Query(`SELECT id, channel, content, sender_name FROM messages`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, channel, content, sender string
		if err := rows.Scan(&id, &channel, &content, &sender); err != nil {
			return err
		}
		if err := s.indexMessageFTS(id, channel, content, sender); err != nil {
			return err
		}
	}
	return rows.Err()
}
