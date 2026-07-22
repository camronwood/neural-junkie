package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/chatcontext"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ContentHash returns a stable hash for chunk content.
func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:8])
}

// ChunkText splits long text into overlapping chunks.
func ChunkText(text string, chunkSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if overlap < 0 {
		overlap = 0
	}
	if len(text) <= chunkSize {
		return []string{text}
	}
	var out []string
	start := 0
	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}
		part := strings.TrimSpace(text[start:end])
		if part != "" {
			out = append(out, part)
		}
		if end >= len(text) {
			break
		}
		next := end - overlap
		if next <= start {
			next = end
		}
		start = next
	}
	return out
}

// MessageChunks builds indexable chunks from a chat message.
func MessageChunks(msg *protocol.Message) []Chunk {
	if msg == nil || chatcontext.OmitFromLLMHistory(msg) {
		return nil
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return nil
	}
	sender := msg.From.Name
	if sender == "" {
		sender = string(msg.From.Type)
	}
	parts := ChunkText(content, DefaultChunkSize, DefaultChunkOverlap)
	chunks := make([]Chunk, 0, len(parts))
	for i, part := range parts {
		id := fmt.Sprintf("msg:%s:%d", msg.ID, i)
		if len(parts) == 1 {
			id = "msg:" + msg.ID
		}
		chunks = append(chunks, Chunk{
			ID:           id,
			SourceType:   SourceMessage,
			SourceID:     msg.ID,
			Channel:      strings.TrimSpace(msg.Channel),
			ThreadID:     msg.GetThreadID(),
			GoalID:       messageMetadataString(msg, "original_goal_id", "goal_id"),
			IsCorrection: messageMetadataBool(msg, "is_correction"),
			SenderName:   sender,
			Content:      part,
			ContentHash:  ContentHash(part),
			CreatedAt:    msg.Timestamp,
		})
	}
	return chunks
}

func messageMetadataString(msg *protocol.Message, keys ...string) string {
	if msg == nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := msg.Metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func messageMetadataBool(msg *protocol.Message, key string) bool {
	if msg == nil {
		return false
	}
	value, _ := msg.Metadata[key].(bool)
	return value
}

// FileChunks builds indexable chunks from a collab markdown file.
func FileChunks(absPath, relPath, collaborationID, channel string, data []byte, modTime int64) []Chunk {
	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil
	}
	hash := ContentHash(content)
	parts := ChunkText(content, DefaultChunkSize, DefaultChunkOverlap)
	chunks := make([]Chunk, 0, len(parts))
	for i, part := range parts {
		id := fmt.Sprintf("file:%s:%s:%d", absPath, hash, i)
		if len(parts) == 1 {
			id = fmt.Sprintf("file:%s:%s", absPath, hash)
		}
		c := Chunk{
			ID:              id,
			SourceType:      SourceCollabArtifact,
			SourceID:        absPath,
			Channel:         strings.TrimSpace(channel),
			CollaborationID: strings.TrimSpace(collaborationID),
			RelPath:         strings.TrimSpace(relPath),
			Content:         part,
			ContentHash:     ContentHash(part),
		}
		if modTime > 0 {
			c.CreatedAt = time.UnixMilli(modTime)
		}
		chunks = append(chunks, c)
	}
	return chunks
}

// CollabIDFromRelPath extracts collaboration id from collabs/<id>/... paths.
func CollabIDFromRelPath(relPath string) string {
	relPath = strings.TrimSpace(strings.ReplaceAll(relPath, "\\", "/"))
	parts := strings.Split(strings.Trim(relPath, "/"), "/")
	if len(parts) < 2 || parts[0] != "collabs" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
