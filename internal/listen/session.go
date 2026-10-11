// Package listen implements Desktop Listen sessions: local STT chunking and finalize.
package listen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session state machine values.
const (
	StateIdle       = "idle"
	StateListening  = "listening"
	StateFinalizing = "finalizing"
	StateError      = "error"
)

// Err codes / sentinel errors for API mapping.
var (
	ErrAlreadyListening = errors.New("listen session already active")
	ErrNoSession        = errors.New("no active listen session")
	ErrSTTUnavailable   = errors.New("stt_unavailable")
	ErrChunkTooLarge    = errors.New("chunk too large")
	ErrInvalidState     = errors.New("invalid listen session state")
)

const (
	defaultMaxChunkBytes = 5 << 20 // 5 MiB
	defaultMaxChunks     = 500
)

// IngestFunc persists a finalized transcript as a meeting note.
// Returns note id, title, summary for the stop response.
type IngestFunc func(ctx context.Context, title, transcript string) (noteID, noteTitle, summary string, err error)

// Transcriber converts a local audio file (typically WAV) to text.
type Transcriber interface {
	Ready() (ready bool, hint string)
	Transcribe(ctx context.Context, audioPath string) (string, error)
}

// Status is a snapshot of the listen subsystem.
type Status struct {
	Active      bool   `json:"active"`
	SessionID   string `json:"session_id,omitempty"`
	State       string `json:"state"`
	STTReady    bool   `json:"stt_ready"`
	STTHint     string `json:"stt_hint,omitempty"`
	PartialText string `json:"partial_text,omitempty"`
	Error       string `json:"error,omitempty"`
	Title       string `json:"title,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// StopResult is returned after a successful stop + ingest.
type StopResult struct {
	SessionID   string `json:"session_id"`
	NoteID      string `json:"note_id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Transcript  string `json:"transcript"`
	PartialText string `json:"partial_text"`
}

type session struct {
	ID          string
	Title       string
	State       string
	PartialText string
	Chunks      int
	StartedAt   time.Time
	TmpDir      string
	LastError   string
}

// Manager owns at most one active listen session per hub.
type Manager struct {
	mu            sync.Mutex
	session       *session
	transcriber   Transcriber
	ingest        IngestFunc
	enabled       bool
	maxChunkBytes int
	maxChunks     int
	baseTmpDir    string
}

var (
	globalMu sync.Mutex
	global   *Manager
)

// SetGlobal registers the hub-wide listen manager (HTTP + slash commands).
func SetGlobal(m *Manager) {
	globalMu.Lock()
	defer globalMu.Unlock()
	global = m
}

// Global returns the hub-wide listen manager, if any.
func Global() *Manager {
	globalMu.Lock()
	defer globalMu.Unlock()
	return global
}

// NewManager creates a listen manager. tmpBase may be empty (uses os.TempDir).
func NewManager(t Transcriber, ingest IngestFunc, enabled bool, tmpBase string) *Manager {
	if tmpBase == "" {
		tmpBase = filepath.Join(os.TempDir(), "neural-junkie-listen")
	}
	return &Manager{
		transcriber:   t,
		ingest:        ingest,
		enabled:       enabled,
		maxChunkBytes: defaultMaxChunkBytes,
		maxChunks:     defaultMaxChunks,
		baseTmpDir:    tmpBase,
	}
}

// SetEnabled updates whether Desktop Listen is enabled.
func (m *Manager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// SetIngest replaces the ingest callback (used in tests / late wiring).
func (m *Manager) SetIngest(fn IngestFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ingest = fn
}

// SetTranscriber replaces the STT backend.
func (m *Manager) SetTranscriber(t Transcriber) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transcriber = t
}

// Status returns the current listen status.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	ready, hint := false, ""
	if m.transcriber != nil {
		ready, hint = m.transcriber.Ready()
	}
	st := Status{
		State:    StateIdle,
		STTReady: ready,
		STTHint:  hint,
		Enabled:  m.enabled,
	}
	if m.session != nil {
		st.Active = m.session.State == StateListening || m.session.State == StateFinalizing
		st.SessionID = m.session.ID
		st.State = m.session.State
		st.PartialText = m.session.PartialText
		st.Error = m.session.LastError
		st.Title = m.session.Title
	}
	return st
}

// Start begins a new listen session.
func (m *Manager) Start(title string) (sessionID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		return "", fmt.Errorf("%w: desktop listen disabled", ErrSTTUnavailable)
	}
	if m.transcriber == nil {
		return "", fmt.Errorf("%w: no transcriber configured", ErrSTTUnavailable)
	}
	if ready, hint := m.transcriber.Ready(); !ready {
		if hint == "" {
			hint = "install whisper.cpp and place model under ~/.neural-junkie/models/whisper/"
		}
		return "", fmt.Errorf("%w: %s", ErrSTTUnavailable, hint)
	}
	if m.session != nil && (m.session.State == StateListening || m.session.State == StateFinalizing) {
		return "", ErrAlreadyListening
	}

	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("Desktop listen %s", time.Now().Format("2006-01-02 15:04"))
	}

	id := uuid.NewString()
	tmpDir := filepath.Join(m.baseTmpDir, id)
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return "", fmt.Errorf("create session temp dir: %w", err)
	}

	m.session = &session{
		ID:        id,
		Title:     title,
		State:     StateListening,
		StartedAt: time.Now(),
		TmpDir:    tmpDir,
	}
	return id, nil
}

// Chunk accepts audio bytes, transcribes, and appends to the partial transcript.
func (m *Manager) Chunk(ctx context.Context, sessionID string, audio []byte) (partial string, err error) {
	m.mu.Lock()
	sess := m.session
	if sess == nil || sess.State != StateListening {
		m.mu.Unlock()
		return "", ErrNoSession
	}
	if sessionID != "" && sessionID != sess.ID {
		m.mu.Unlock()
		return "", fmt.Errorf("%w: session mismatch", ErrNoSession)
	}
	if len(audio) == 0 {
		partial = sess.PartialText
		m.mu.Unlock()
		return partial, nil
	}
	if len(audio) > m.maxChunkBytes {
		m.mu.Unlock()
		return "", ErrChunkTooLarge
	}
	if sess.Chunks >= m.maxChunks {
		m.mu.Unlock()
		return "", fmt.Errorf("%w: too many chunks", ErrInvalidState)
	}
	t := m.transcriber
	tmpDir := sess.TmpDir
	chunkIdx := sess.Chunks
	sess.Chunks++
	m.mu.Unlock()

	path := filepath.Join(tmpDir, fmt.Sprintf("chunk-%04d.wav", chunkIdx))
	if err := os.WriteFile(path, audio, 0o600); err != nil {
		m.mu.Lock()
		if m.session != nil && m.session.ID == sess.ID {
			m.session.LastError = err.Error()
		}
		m.mu.Unlock()
		return "", err
	}
	defer func() { _ = os.Remove(path) }()

	text, err := t.Transcribe(ctx, path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil || m.session.ID != sess.ID {
		return "", ErrNoSession
	}
	if err != nil {
		m.session.LastError = err.Error()
		return m.session.PartialText, err
	}
	text = strings.TrimSpace(text)
	if text != "" {
		if m.session.PartialText == "" {
			m.session.PartialText = text
		} else {
			m.session.PartialText = strings.TrimSpace(m.session.PartialText + " " + text)
		}
	}
	return m.session.PartialText, nil
}

// Stop finalizes the session, ingests the transcript, and clears temp audio.
func (m *Manager) Stop(ctx context.Context, sessionID string) (*StopResult, error) {
	m.mu.Lock()
	sess := m.session
	if sess == nil || sess.State != StateListening {
		m.mu.Unlock()
		return nil, ErrNoSession
	}
	if sessionID != "" && sessionID != sess.ID {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w: session mismatch", ErrNoSession)
	}
	sess.State = StateFinalizing
	transcript := strings.TrimSpace(sess.PartialText)
	title := sess.Title
	id := sess.ID
	tmpDir := sess.TmpDir
	ingest := m.ingest
	m.mu.Unlock()

	defer func() {
		_ = os.RemoveAll(tmpDir)
		m.mu.Lock()
		if m.session != nil && m.session.ID == id {
			m.session = nil
		}
		m.mu.Unlock()
	}()

	if transcript == "" {
		return nil, fmt.Errorf("%w: empty transcript", ErrInvalidState)
	}
	if ingest == nil {
		return nil, fmt.Errorf("ingest not configured")
	}

	noteID, noteTitle, summary, err := ingest(ctx, title, transcript)
	if err != nil {
		m.mu.Lock()
		if m.session != nil && m.session.ID == id {
			m.session.State = StateError
			m.session.LastError = err.Error()
		}
		m.mu.Unlock()
		return nil, err
	}
	if noteTitle == "" {
		noteTitle = title
	}
	return &StopResult{
		SessionID:   id,
		NoteID:      noteID,
		Title:       noteTitle,
		Summary:     summary,
		Transcript:  transcript,
		PartialText: transcript,
	}, nil
}

// Cancel aborts the session without ingesting.
func (m *Manager) Cancel(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil {
		return ErrNoSession
	}
	if sessionID != "" && sessionID != m.session.ID {
		return fmt.Errorf("%w: session mismatch", ErrNoSession)
	}
	tmpDir := m.session.TmpDir
	m.session = nil
	_ = os.RemoveAll(tmpDir)
	return nil
}
