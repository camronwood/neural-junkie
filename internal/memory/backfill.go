package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/chatcontext"
	"github.com/camronwood/neural-junkie/internal/protocol"
	sqlitestore "github.com/camronwood/neural-junkie/internal/store/sqlite"
	_ "modernc.org/sqlite"
)

// BackfillMessages indexes unindexed rows from messages.db.
func BackfillMessages(ctx context.Context) error {
	if !memoryEnabled() {
		return nil
	}
	path, err := sqlitestore.PathDefault()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, channel, thread_id, sender_id, sender_name, sender_type,
content, msg_type, metadata_json, created_at FROM messages ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var indexed, skipped int
	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, err := scanBackfillMessage(rows)
		if err != nil || msg == nil {
			continue
		}
		if chatcontext.OmitFromLLMHistory(msg) {
			skipped++
			continue
		}
		has, err := globalStore.HasSource(msg.ID)
		if err != nil || has {
			skipped++
			continue
		}
		if err := indexMessage(ctx, msg); err != nil {
			log.Printf("[memory] backfill %s: %v", msg.ID, err)
			continue
		}
		indexed++
	}
	if indexed > 0 {
		log.Printf("[memory] backfill indexed %d messages (%d skipped)", indexed, skipped)
	}
	return rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBackfillMessage(rows rowScanner) (*protocol.Message, error) {
	var id, channel, threadID, senderID, senderName, senderType, content, msgType, metaJSON string
	var created int64
	if err := rows.Scan(&id, &channel, &threadID, &senderID, &senderName, &senderType,
		&content, &msgType, &metaJSON, &created); err != nil {
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

// BackfillCollabDirs indexes markdown under ~/.neural-junkie collaboration review dirs.
func BackfillCollabDirs(ctx context.Context, assetsBase string) error {
	if !memoryEnabled() || assetsBase == "" {
		return nil
	}
	reviews := filepath.Join(assetsBase, "reviews")
	entries, err := os.ReadDir(reviews)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		collabID := e.Name()
		dir := filepath.Join(reviews, collabID)
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}
			rel, _ := filepath.Rel(dir, path)
			rel = filepath.ToSlash(rel)
			has, _ := globalStore.HasSource(path)
			if has {
				return nil
			}
			return indexCollabFile(ctx, path, rel, collabID, "")
		})
	}
	return nil
}

// ScheduleBackfill runs message and collab backfill in the background.
func ScheduleBackfill(assetsBase string) {
	if !memoryEnabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := BackfillMessages(ctx); err != nil {
			log.Printf("[memory] backfill messages: %v", err)
		}
		if err := BackfillCollabDirs(ctx, assetsBase); err != nil {
			log.Printf("[memory] backfill collab dirs: %v", err)
		}
	}()
}
