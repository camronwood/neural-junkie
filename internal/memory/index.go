package memory

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/embed"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	globalStore       *Store
	embedClient       *embed.Client
	embedModel        = embed.DefaultModel
	enabledChecker    func() bool
	collabResolver    func(channel string) string
)

// SetStore wires the global memory store.
func SetStore(s *Store) { globalStore = s }

// SetEmbedClient configures the Ollama embed client.
func SetEmbedClient(c *embed.Client, model string) {
	embedClient = c
	if strings.TrimSpace(model) != "" {
		embedModel = model
	}
}

// SetEnabledChecker returns true when conversation memory is active.
func SetEnabledChecker(fn func() bool) { enabledChecker = fn }

// SetCollabResolver resolves collaboration id from channel name.
func SetCollabResolver(fn func(channel string) string) { collabResolver = fn }

func memoryEnabled() bool {
	return enabledChecker != nil && enabledChecker() && globalStore != nil
}

// ResolveCollabID returns collaboration id for a channel.
func ResolveCollabID(channel string) string {
	if collabResolver == nil {
		return ""
	}
	return collabResolver(channel)
}

// IndexMessage indexes a persisted chat message asynchronously.
func IndexMessage(msg *protocol.Message) {
	if !memoryEnabled() || msg == nil {
		return
	}
	go func(m *protocol.Message) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := indexMessage(ctx, m); err != nil {
			log.Printf("[memory] index message %s: %v", m.ID, err)
		}
	}(msg)
}

func indexMessage(ctx context.Context, msg *protocol.Message) error {
	chunks := MessageChunks(msg)
	if len(chunks) == 0 {
		return nil
	}
	collabID := ResolveCollabID(msg.Channel)
	for i := range chunks {
		if collabID != "" {
			chunks[i].CollaborationID = collabID
		}
		if err := upsertChunkWithEmbed(ctx, chunks[i]); err != nil {
			return err
		}
	}
	return nil
}

// IndexCollabFile indexes a markdown collab artifact.
func IndexCollabFile(absPath, relPath, collaborationID, channel string) {
	if !memoryEnabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := indexCollabFile(ctx, absPath, relPath, collaborationID, channel); err != nil {
			log.Printf("[memory] index file %s: %v", relPath, err)
		}
	}()
}

func indexCollabFile(ctx context.Context, absPath, relPath, collaborationID, channel string) error {
	absPath = strings.TrimSpace(absPath)
	if absPath == "" {
		return nil
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	if collaborationID == "" {
		collaborationID = CollabIDFromRelPath(relPath)
	}
	if channel == "" && collaborationID != "" && collabResolver != nil {
		// channel may be unknown for file-only indexing; collab scope still works
	}
	_ = globalStore.DeleteBySource(absPath)
	chunks := FileChunks(absPath, relPath, collaborationID, channel, data, info.ModTime().UnixMilli())
	for _, ch := range chunks {
		if err := upsertChunkWithEmbed(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

func upsertChunkWithEmbed(ctx context.Context, ch Chunk) error {
	if embedClient != nil {
		vec, err := embedClient.Embed(ctx, ch.Content, true)
		if err == nil && len(vec) > 0 {
			ch.Vector = vec
			ch.EmbeddingModel = embedModel
		}
	}
	return globalStore.UpsertChunk(ch)
}

// DeleteByChannel removes all memory for a channel.
func DeleteByChannel(channel string) error {
	if globalStore == nil {
		return nil
	}
	return globalStore.DeleteByChannel(channel)
}

// GlobalStore returns the wired global store (for backfill / tests).
func GlobalStore() *Store { return globalStore }
