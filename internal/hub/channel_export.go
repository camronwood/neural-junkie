package hub

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ExportChannelMessages returns channel history merged from SQLite (when enabled) and in-memory store.
func (h *Hub) ExportChannelMessages(channelName string) []*protocol.Message {
	if h == nil || channelName == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []*protocol.Message
	add := func(msg *protocol.Message) {
		if msg == nil || msg.ID == "" {
			return
		}
		if _, ok := seen[msg.ID]; ok {
			return
		}
		seen[msg.ID] = struct{}{}
		out = append(out, msg)
	}

	if h.persistentStore != nil {
		const page = 500
		beforeID := ""
		for {
			batch, err := h.persistentStore.ListChannelMessages(channelName, page, beforeID)
			if err != nil || len(batch) == 0 {
				break
			}
			for _, msg := range batch {
				add(msg)
			}
			if len(batch) < page {
				break
			}
			oldest := batch[len(batch)-1]
			if oldest == nil || oldest.ID == "" || oldest.ID == beforeID {
				break
			}
			beforeID = oldest.ID
		}
	}

	h.mu.RLock()
	for _, msg := range h.messages[channelName] {
		add(msg)
	}
	h.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		ti := out[i].Timestamp
		tj := out[j].Timestamp
		if ti.IsZero() && tj.IsZero() {
			return out[i].ID < out[j].ID
		}
		if ti.IsZero() {
			return true
		}
		if tj.IsZero() {
			return false
		}
		return ti.Before(tj)
	})
	return out
}

// FormatChannelExportMarkdown renders messages as a plain-text transcript.
func FormatChannelExportMarkdown(channel string, msgs []*protocol.Message) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Channel export: #%s\n\n", channel))
	b.WriteString(fmt.Sprintf("Exported %s · %d messages\n\n", time.Now().Format(time.RFC3339), len(msgs)))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		ts := msg.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		from := strings.TrimSpace(msg.From.Name)
		if from == "" {
			from = msg.From.ID
		}
		b.WriteString(fmt.Sprintf("## %s · %s\n\n", ts.Format(time.RFC3339), from))
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			content = "_empty_"
		}
		b.WriteString(content)
		b.WriteString("\n\n---\n\n")
	}
	return b.String()
}

// IsChannelDurableLocked reports durable flag; caller must hold h.mu read lock if concurrent.
func (h *Hub) IsChannelDurable(channelName string) bool {
	return h.isChannelDurable(channelName)
}
