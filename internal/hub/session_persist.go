package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	// MaxSessionRestoreBytes: files larger than this are auto-archived on startup (not loaded).
	MaxSessionRestoreBytes = 64 * 1024 * 1024
	// Disk persistence uses stricter caps than in-memory hub history (5000/2000).
	MaxSessionPersistChannelHistory = 500
	MaxSessionPersistThreadHistory  = 200
	maxPersistDiscussionMessages    = 20
	maxPersistTerminalCollaborations = 10
)

func isTerminalCollaborationPhase(p collaboration.CollaborationPhase) bool {
	return p == collaboration.PhaseCompleted || p == collaboration.PhaseCancelled
}

func cloneMessageForSessionPersist(m *protocol.Message) *protocol.Message {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return m
	}
	var out protocol.Message
	if err := json.Unmarshal(data, &out); err != nil {
		return m
	}
	if out.Metadata != nil {
		delete(out.Metadata, "collaboration_data")
	}
	return &out
}

func slimCollaborationForPersist(c *collaboration.Collaboration) *collaboration.Collaboration {
	if c == nil {
		return nil
	}
	data, err := json.Marshal(c)
	if err != nil {
		return c
	}
	var out collaboration.Collaboration
	if err := json.Unmarshal(data, &out); err != nil {
		return c
	}
	if out.Discussion != nil && len(out.Discussion.Messages) > maxPersistDiscussionMessages {
		msgs := out.Discussion.Messages
		out.Discussion.Messages = append([]*protocol.Message(nil), msgs[len(msgs)-maxPersistDiscussionMessages:]...)
	}
	return &out
}

// prepareSessionSnapshotForPersist trims history and strips bulky metadata before JSON encode.
func prepareSessionSnapshotForPersist(snapshot *SessionSnapshot) {
	if snapshot == nil {
		return
	}
	for name, ch := range snapshot.Channels {
		if ch == nil {
			continue
		}
		trimmed := make([]*protocol.Message, 0, len(ch.Messages))
		for _, m := range ch.Messages {
			trimmed = append(trimmed, cloneMessageForSessionPersist(m))
		}
		ch.Messages = keepLastPtrSlice(trimmed, MaxSessionPersistChannelHistory)
		snapshot.Channels[name] = ch
	}
	for tid, th := range snapshot.Threads {
		if th == nil {
			continue
		}
		trimmed := make([]*protocol.Message, 0, len(th.Messages))
		for _, m := range th.Messages {
			trimmed = append(trimmed, cloneMessageForSessionPersist(m))
		}
		th.Messages = keepLastPtrSlice(trimmed, MaxSessionPersistThreadHistory)
		snapshot.Threads[tid] = th
	}
	// Mirror collab discussion transcripts into their channels so the UI can scroll/search them.
	syncCollabDiscussionIntoSnapshotChannels(snapshot)
	if len(snapshot.Collaborations) > 0 {
		snapshot.Collaborations = trimCollaborationsForPersist(snapshot.Collaborations)
	}
	dedupeSnapshotChannelMessages(snapshot)
}

// trimCollaborationsForPersist keeps all active collaborations and only the most
// recent terminal ones so last-session.json does not accumulate every past collab.
func trimCollaborationsForPersist(collabs map[string]*collaboration.Collaboration) map[string]*collaboration.Collaboration {
	if len(collabs) == 0 {
		return collabs
	}
	active := make(map[string]*collaboration.Collaboration)
	var terminal []*collaboration.Collaboration
	for id, c := range collabs {
		if c == nil {
			continue
		}
		slim := slimCollaborationForPersist(c)
		if isTerminalCollaborationPhase(slim.Phase) {
			terminal = append(terminal, slim)
		} else {
			active[id] = slim
		}
	}
	if len(terminal) > maxPersistTerminalCollaborations {
		sort.Slice(terminal, func(i, j int) bool {
			return terminal[i].UpdatedAt.After(terminal[j].UpdatedAt)
		})
		terminal = terminal[:maxPersistTerminalCollaborations]
	}
	out := make(map[string]*collaboration.Collaboration, len(active)+len(terminal))
	for id, c := range active {
		out[id] = c
	}
	for _, c := range terminal {
		out[c.ID] = c
	}
	return out
}

// syncCollabDiscussionIntoSnapshotChannels copies discussion.messages into the collab
// channel timeline (deduped by message ID) so restored sessions show collab history in-chat.
func syncCollabDiscussionIntoSnapshotChannels(snapshot *SessionSnapshot) {
	if snapshot == nil || len(snapshot.Collaborations) == 0 {
		return
	}
	for _, c := range snapshot.Collaborations {
		if c == nil || c.Channel == "" || c.Discussion == nil || len(c.Discussion.Messages) == 0 {
			continue
		}
		ch := snapshot.Channels[c.Channel]
		if ch == nil {
			ch = &ChannelSnapshot{Name: c.Channel, Messages: []*protocol.Message{}}
			snapshot.Channels[c.Channel] = ch
		}
		seen := make(map[string]struct{}, len(ch.Messages))
		for _, m := range ch.Messages {
			if m != nil && m.ID != "" {
				seen[m.ID] = struct{}{}
			}
		}
		for _, dm := range c.Discussion.Messages {
			if dm == nil || dm.ID == "" {
				continue
			}
			if _, ok := seen[dm.ID]; ok {
				continue
			}
			ch.Messages = append(ch.Messages, cloneMessageForSessionPersist(dm))
			seen[dm.ID] = struct{}{}
		}
	}
}

func dedupeSnapshotChannelMessages(snapshot *SessionSnapshot) {
	if snapshot == nil {
		return
	}
	for name, ch := range snapshot.Channels {
		if ch == nil || len(ch.Messages) == 0 {
			continue
		}
		ch.Messages = dedupeMessagesByID(ch.Messages)
		snapshot.Channels[name] = ch
	}
}

func dedupeMessagesByID(msgs []*protocol.Message) []*protocol.Message {
	if len(msgs) == 0 {
		return msgs
	}
	seen := make(map[string]struct{}, len(msgs))
	out := make([]*protocol.Message, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		if m.ID == "" {
			out = append(out, m)
			continue
		}
		if _, ok := seen[m.ID]; ok {
			continue
		}
		seen[m.ID] = struct{}{}
		out = append(out, m)
	}
	return out
}

// hydrateCollabChannelsFromCollaborationsLocked merges persisted discussion transcripts into
// live channel history after session restore. Caller must hold h.mu (write lock).
func (h *Hub) hydrateCollabChannelsFromCollaborationsLocked(collabs map[string]*collaboration.Collaboration) {
	if h == nil || len(collabs) == 0 {
		return
	}
	for _, c := range collabs {
		if c == nil || c.Channel == "" || c.Discussion == nil || len(c.Discussion.Messages) == 0 {
			continue
		}
		if _, ok := h.channels[c.Channel]; !ok {
			continue
		}
		seen := make(map[string]struct{})
		for _, m := range h.messages[c.Channel] {
			if m != nil && m.ID != "" {
				seen[m.ID] = struct{}{}
			}
		}
		for _, dm := range c.Discussion.Messages {
			if dm == nil || dm.ID == "" {
				continue
			}
			if _, ok := seen[dm.ID]; ok {
				continue
			}
			h.messages[c.Channel] = append(h.messages[c.Channel], dm)
			seen[dm.ID] = struct{}{}
		}
	}
}

// archiveUnusableSessionFile moves path aside so the hub can start fresh without user action.
func archiveUnusableSessionFile(path, reason string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	archived := filepath.Join(dir, fmt.Sprintf("last-session.archived-%s.json", time.Now().Format("20060102-150405")))
	if err := os.Rename(path, archived); err != nil {
		return err
	}
	log.Printf("💾 Archived unusable session file (%s, was %.1f MiB) → %s",
		reason, float64(fi.Size())/(1024*1024), archived)
	return nil
}

func sessionFileReadyToLoad(path string) (load bool, size int64, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, 0, err
	}
	if fi.Size() <= MaxSessionRestoreBytes {
		return true, fi.Size(), nil
	}
	if os.Getenv("NEURAL_JUNKIE_FORCE_SESSION_RESTORE") == "1" {
		return true, fi.Size(), nil
	}
	if err := archiveUnusableSessionFile(path, fmt.Sprintf("over %d MiB", MaxSessionRestoreBytes/(1024*1024))); err != nil {
		return false, fi.Size(), err
	}
	return false, fi.Size(), nil
}
