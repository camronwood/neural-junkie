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
	"github.com/camronwood/neural-junkie/internal/collaboration"
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

// BackfillCollabDirs indexes markdown under collaboration asset dirs:
//   - <assetsBase>/reviews/<id>/*.md  (legacy review layout)
//   - <assetsBase>/collabs/<id>/*.md  (assets-root collab layout, when present)
func BackfillCollabDirs(ctx context.Context, assetsBase string) error {
	if !memoryEnabled() || strings.TrimSpace(assetsBase) == "" {
		return nil
	}
	if err := backfillCollabTree(ctx, filepath.Join(assetsBase, "reviews"), "reviews"); err != nil {
		return err
	}
	return backfillCollabTree(ctx, filepath.Join(assetsBase, collaboration.ProjectCollabsDirName), "collabs")
}

// ScheduleBackfill runs message and collab backfill in the background.
// Optional workspaceRoots also index <root>/collabs/<id>/*.md (findings.md etc.).
func ScheduleBackfill(assetsBase string, workspaceRoots ...string) {
	if !memoryEnabled() {
		return
	}
	roots := uniqueNonEmpty(workspaceRoots)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := BackfillMessages(ctx); err != nil {
			log.Printf("[memory] backfill messages: %v", err)
		}
		if err := BackfillCollabDirs(ctx, assetsBase); err != nil {
			log.Printf("[memory] backfill collab dirs: %v", err)
		}
		for _, root := range roots {
			if err := BackfillWorkspaceCollabs(ctx, root); err != nil {
				log.Printf("[memory] backfill workspace %s: %v", root, err)
			}
		}
	}()
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
