package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
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
		slimMessageMetadataForPersist(out.Metadata)
	}
	return &out
}

const maxPersistFileTreeBytes = 12000

func slimMessageMetadataForPersist(md map[string]interface{}) {
	if md == nil {
		return
	}
	slimWorkspaceContextMetadata(md)
	slimGrantedHubDataAccessMetadata(md)
	delete(md, "prompt_attachments")
	delete(md, "user_images")
}

func slimWorkspaceContextMetadata(md map[string]interface{}) {
	rawCtx, ok := md["workspace_context"]
	if !ok {
		return
	}
	ctxMap, ok := rawCtx.(map[string]interface{})
	if !ok {
		delete(md, "workspace_context")
		return
	}
	safeCtx := map[string]interface{}{}
	if workspaceName, ok := ctxMap["workspace_name"].(string); ok {
		safeCtx["workspace_name"] = workspaceName
	}
	if workspacePath, ok := ctxMap["workspace_path"].(string); ok {
		safeCtx["workspace_path"] = workspacePath
	}
	if scope, ok := ctxMap["context_scope"].(string); ok && scope != "" {
		safeCtx["context_scope"] = scope
	}
	if fileTree, ok := ctxMap["file_tree"].(string); ok && fileTree != "" {
		if len(fileTree) > maxPersistFileTreeBytes {
			fileTree = fileTree[:maxPersistFileTreeBytes] + "\n... (truncated)"
		}
		safeCtx["file_tree"] = fileTree
	}
	if len(safeCtx) == 0 {
		delete(md, "workspace_context")
		return
	}
	md["workspace_context"] = safeCtx
	md[agent.MetadataContextScope] = agent.ContextScopeOutline
}

func slimGrantedHubDataAccessMetadata(md map[string]interface{}) {
	raw, ok := md[agent.MetadataGrantedHubDataAccess]
	if !ok || raw == nil {
		return
	}
	root, ok := raw.(map[string]interface{})
	if !ok {
		delete(md, agent.MetadataGrantedHubDataAccess)
		return
	}
	entries, ok := root["entries"].([]interface{})
	if !ok || len(entries) == 0 {
		delete(md, agent.MetadataGrantedHubDataAccess)
		return
	}
	trimmed := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		item, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		out := map[string]interface{}{}
		for k, v := range item {
			if k == "content" {
				continue
			}
			out[k] = v
		}
		if len(out) > 0 {
			trimmed = append(trimmed, out)
		}
	}
	if len(trimmed) == 0 {
		delete(md, agent.MetadataGrantedHubDataAccess)
		return
	}
	md[agent.MetadataGrantedHubDataAccess] = map[string]interface{}{"entries": trimmed}
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
	if out.Discussion != nil {
		for i, dm := range out.Discussion.Messages {
			out.Discussion.Messages[i] = cloneMessageForSessionPersist(dm)
		}
	}
	if out.SourceWorkspaceContext != nil {
		wsMD := map[string]interface{}{"workspace_context": out.SourceWorkspaceContext}
		slimWorkspaceContextMetadata(wsMD)
		if slim, ok := wsMD["workspace_context"].(map[string]interface{}); ok {
			out.SourceWorkspaceContext = slim
		} else {
			out.SourceWorkspaceContext = nil
		}
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

// --- Session snapshot types and I/O ---

// --- Session Recording ---

// SessionSnapshot captures the full state of a chat session for debugging/review.
type SessionSnapshot struct {
	SavedAt        time.Time                               `json:"saved_at"`
	Channels       map[string]*ChannelSnapshot             `json:"channels"`
	Threads        map[string]*ThreadSnapshot              `json:"threads"`
	Agents         []*protocol.AgentInfo                   `json:"agents"`
	Collaborations map[string]*collaboration.Collaboration `json:"collaborations,omitempty"`
}

// SessionSaveHealth tracks snapshot save freshness and failures.
type SessionSaveHealth struct {
	LastSavedAt    time.Time `json:"last_saved_at,omitempty"`
	LastFailureAt  time.Time `json:"last_failure_at,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	LastSizeBytes  int       `json:"last_size_bytes,omitempty"`
	LastDurationMs int64     `json:"last_duration_ms,omitempty"`
	SaveCount      int64     `json:"save_count"`
	FailureCount   int64     `json:"failure_count"`
	LastPath       string    `json:"last_path,omitempty"`
}

// ChannelSnapshot holds all messages for a single channel.
type ChannelSnapshot struct {
	Name             string               `json:"name"`
	Description      string               `json:"description"`
	Type             protocol.ChannelType `json:"type"`
	CreatedBy        string               `json:"created_by,omitempty"`
	Members          []string             `json:"members,omitempty"`
	Messages         []*protocol.Message  `json:"messages"`
	SessionSummary   string               `json:"session_summary,omitempty"`
	SessionSummaryAt time.Time            `json:"session_summary_at,omitempty"`
}

// ThreadSnapshot holds all messages and metadata for a single thread.
type ThreadSnapshot struct {
	ThreadID string                   `json:"thread_id"`
	Metadata *protocol.ThreadMetadata `json:"metadata"`
	Messages []*protocol.Message      `json:"messages"`
}

// TakeSessionSnapshot returns a deep copy of the current session state.
func (h *Hub) TakeSessionSnapshot() *SessionSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	snapshot := &SessionSnapshot{
		SavedAt:        time.Now(),
		Channels:       make(map[string]*ChannelSnapshot),
		Threads:        make(map[string]*ThreadSnapshot),
		Agents:         make([]*protocol.AgentInfo, 0),
		Collaborations: make(map[string]*collaboration.Collaboration),
	}

	// Capture channel messages
	for name, ch := range h.channels {
		cs := &ChannelSnapshot{
			Name:        ch.Name,
			Description: ch.Description,
			Type:        ch.Type,
			CreatedBy:   ch.CreatedBy,
			Members:     ch.Members,
			Messages:    make([]*protocol.Message, 0),
		}
		if h.channelContext != nil {
			if ctx := h.channelContext[name]; ctx != nil && ctx.Summary != "" {
				cs.SessionSummary = ctx.Summary
				cs.SessionSummaryAt = ctx.UpdatedAt
			}
		}
		if msgs, ok := h.messages[name]; ok {
			for _, m := range msgs {
				if m.Type == protocol.MessageTypeAgentStatus ||
					m.Type == protocol.MessageTypeStreamDelta ||
					m.Type == protocol.MessageTypeStreamEnd {
					continue
				}
				cs.Messages = append(cs.Messages, m)
			}
		}
		snapshot.Channels[name] = cs
	}

	// Capture threads
	for threadID, msgs := range h.threads {
		ts := &ThreadSnapshot{
			ThreadID: threadID,
			Messages: make([]*protocol.Message, len(msgs)),
		}
		copy(ts.Messages, msgs)
		if meta, ok := h.threadMetadata[threadID]; ok {
			ts.Metadata = meta
		}
		snapshot.Threads[threadID] = ts
	}

	// Capture active agents
	for _, a := range h.agents {
		snapshot.Agents = append(snapshot.Agents, a)
	}
	if h.collabManager != nil {
		snapshot.Collaborations = h.collabManager.Snapshot()
	}

	return snapshot
}

// LoadSessionFromFile restores channels/messages/threads and collaboration
// state from a previous snapshot. It is safe to call on startup.
func (h *Hub) LoadSessionFromFile(path string) error {
	load, _, statErr := sessionFileReadyToLoad(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return nil
		}
		return fmt.Errorf("failed stat session file: %w", statErr)
	}
	if !load {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed reading session file: %w", err)
	}

	var snapshot SessionSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		if archErr := archiveUnusableSessionFile(path, "corrupt JSON"); archErr != nil {
			return fmt.Errorf("failed to unmarshal session file: %w (archive failed: %v)", err, archErr)
		}
		return nil
	}

	h.mu.Lock()
	h.channels = make(map[string]*protocol.Channel)
	h.messages = make(map[string][]*protocol.Message)
	h.threads = make(map[string][]*protocol.Message)
	h.threadMetadata = make(map[string]*protocol.ThreadMetadata)
	h.subscribers = make(map[string][]chan *protocol.Message)
	h.threadSubscribers = make(map[string][]chan *protocol.Message)
	h.channelContext = make(map[string]*ChannelContextState)
	h.channelSummaryRefreshGen = make(map[string]uint64)

	for name, ch := range snapshot.Channels {
		if ch == nil {
			continue
		}
		channel := &protocol.Channel{
			ID:          uuid.New().String(),
			Name:        ch.Name,
			Description: ch.Description,
			Type:        inferChannelTypeForName(ch.Name, ch.Type),
			CreatedBy:   ch.CreatedBy,
			Created:     snapshot.SavedAt,
			Agents:      []protocol.AgentInfo{},
			Members:     append([]string(nil), ch.Members...),
			Tags:        []string{},
		}
		h.channels[name] = channel
		h.messages[name] = append([]*protocol.Message(nil), ch.Messages...)
		h.subscribers[name] = []chan *protocol.Message{}
		if strings.TrimSpace(ch.SessionSummary) != "" {
			h.channelContext[name] = &ChannelContextState{
				Summary:   ch.SessionSummary,
				UpdatedAt: ch.SessionSummaryAt,
			}
		}
	}
	if len(h.channels) == 0 {
		channel := &protocol.Channel{
			ID:          uuid.New().String(),
			Name:        "general",
			Description: "General discussion",
			Type:        protocol.ChannelTypePublic,
			CreatedBy:   "system",
			Created:     time.Now(),
			Agents:      []protocol.AgentInfo{},
			Members:     []string{},
			Tags:        []string{},
		}
		h.channels[channel.Name] = channel
		h.messages[channel.Name] = []*protocol.Message{}
		h.subscribers[channel.Name] = []chan *protocol.Message{}
	}
	for threadID, thread := range snapshot.Threads {
		if thread == nil {
			continue
		}
		h.threads[threadID] = append([]*protocol.Message(nil), thread.Messages...)
		if thread.Metadata != nil {
			h.threadMetadata[threadID] = thread.Metadata
		}
		h.threadSubscribers[threadID] = []chan *protocol.Message{}
	}
	h.repairChannelTypesLocked()
	h.hydrateCollabChannelsFromCollaborationsLocked(snapshot.Collaborations)
	h.trimAllChannelAndThreadHistoryLocked()
	h.mu.Unlock()

	if h.collabManager != nil && snapshot.Collaborations != nil {
		h.collabManager.Restore(snapshot.Collaborations)
		pruned := h.collabManager.PruneTerminalCollaborations(maxPersistTerminalCollaborations)
		if pruned > 0 {
			log.Printf("💾 Pruned %d terminal collaboration(s) from memory after restore", pruned)
		}
	}

	log.Printf("💾 Session restored from %s (%d channels, %d threads, %d collaborations)",
		path, len(snapshot.Channels), len(snapshot.Threads), len(snapshot.Collaborations))
	return nil
}

// SaveSessionToFile writes the current session snapshot to a JSON file.
// It writes to a temp file first, then renames for atomic replacement.
func (h *Hub) SaveSessionToFile(path string) error {
	h.sessionSaveMu.Lock()
	defer h.sessionSaveMu.Unlock()
	startedAt := time.Now()
	snapshot := h.TakeSessionSnapshot()
	prepareSessionSnapshotForPersist(snapshot)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Write to a unique temp file, fsync it, then rename atomically.
	tmpFile, err := os.CreateTemp(dir, "last-session-*.tmp")
	if err != nil {
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to create temp session file: %w", err)
	}
	tmpPath := tmpFile.Name()
	enc := json.NewEncoder(tmpFile)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snapshot); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to encode session file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to sync session file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to close session file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to set session file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to finalize session file: %w", err)
	}
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}

	written := 0
	if fi, err := os.Stat(path); err == nil {
		written = int(fi.Size())
	}
	h.recordSessionSaveSuccess(path, startedAt, written)
	log.Printf("💾 Session saved to %s (%d bytes)", path, written)
	return nil
}

func (h *Hub) recordSessionSaveSuccess(path string, startedAt time.Time, size int) {
	h.sessionHealthMu.Lock()
	defer h.sessionHealthMu.Unlock()
	h.sessionHealth.LastSavedAt = time.Now()
	h.sessionHealth.LastSizeBytes = size
	h.sessionHealth.LastDurationMs = time.Since(startedAt).Milliseconds()
	h.sessionHealth.LastPath = path
	h.sessionHealth.SaveCount++
	h.sessionHealth.LastError = ""
}

func (h *Hub) recordSessionSaveFailure(path string, startedAt time.Time, err error) {
	h.sessionHealthMu.Lock()
	defer h.sessionHealthMu.Unlock()
	h.sessionHealth.LastFailureAt = time.Now()
	h.sessionHealth.LastDurationMs = time.Since(startedAt).Milliseconds()
	h.sessionHealth.LastPath = path
	h.sessionHealth.FailureCount++
	h.sessionHealth.LastError = err.Error()
}

// GetSessionSaveHealth returns the latest session save diagnostics including
// freshness (age in seconds) for observability endpoints.
func (h *Hub) GetSessionSaveHealth() map[string]interface{} {
	h.sessionHealthMu.RLock()
	health := h.sessionHealth
	h.sessionHealthMu.RUnlock()

	ageSeconds := int64(-1)
	if !health.LastSavedAt.IsZero() {
		ageSeconds = int64(time.Since(health.LastSavedAt).Seconds())
	}
	return map[string]interface{}{
		"last_saved_at":    health.LastSavedAt,
		"last_failure_at":  health.LastFailureAt,
		"last_error":       health.LastError,
		"last_size_bytes":  health.LastSizeBytes,
		"last_duration_ms": health.LastDurationMs,
		"save_count":       health.SaveCount,
		"failure_count":    health.FailureCount,
		"last_path":        health.LastPath,
		"age_seconds":      ageSeconds,
	}
}

// DefaultSessionPath returns the default path for the last session file.
func DefaultSessionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	return filepath.Join(home, ".neural-junkie", "last-session.json")
}
