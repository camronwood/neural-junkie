package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/listen"
)

var (
	listenManagerMu sync.Mutex
	listenManager   *listen.Manager
)

func getListenManager() *listen.Manager {
	listenManagerMu.Lock()
	defer listenManagerMu.Unlock()
	if listenManager != nil {
		return listenManager
	}
	enabled := true
	whisperCfg := listen.DefaultWhisperConfig()
	if a := getAssistantAgent(); a != nil {
		if cfg := a.GetConfig(); cfg != nil {
			enabled = cfg.DesktopListenEnabled
			if strings.TrimSpace(cfg.WhisperCommand) != "" {
				whisperCfg.Command = strings.TrimSpace(cfg.WhisperCommand)
			}
			if strings.TrimSpace(cfg.WhisperModelPath) != "" {
				whisperCfg.ModelPath = strings.TrimSpace(cfg.WhisperModelPath)
			}
		}
	}
	t := &listen.WhisperTranscriber{Cfg: whisperCfg}
	m := listen.NewManager(t, nil, enabled, "")
	m.SetIngest(func(ctx context.Context, title, transcript string) (string, string, string, error) {
		asst := getAssistantAgent()
		if asst == nil {
			return "", "", "", errors.New("assistant agent not available")
		}
		note, err := asst.IngestDesktopListenTranscript(ctx, title, transcript)
		if err != nil {
			return "", "", "", err
		}
		return note.ID, note.Title, note.Summary, nil
	})
	listenManager = m
	listen.SetGlobal(m)
	return listenManager
}

func writeListenError(w http.ResponseWriter, err error, fallbackStatus int) {
	status := fallbackStatus
	code := "error"
	switch {
	case errors.Is(err, listen.ErrSTTUnavailable):
		status = http.StatusServiceUnavailable
		code = "stt_unavailable"
	case errors.Is(err, listen.ErrAlreadyListening):
		status = http.StatusConflict
		code = "already_listening"
	case errors.Is(err, listen.ErrNoSession):
		status = http.StatusNotFound
		code = "no_session"
	case errors.Is(err, listen.ErrChunkTooLarge):
		status = http.StatusRequestEntityTooLarge
		code = "chunk_too_large"
	case errors.Is(err, listen.ErrInvalidState):
		status = http.StatusBadRequest
		code = "invalid_state"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
		"code":  code,
	})
}

func handleAssistantListenStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := getListenManager().Status()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func handleAssistantListenStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	sessionID, err := getListenManager().Start(body.Title)
	if err != nil {
		writeListenError(w, err, http.StatusBadRequest)
		return
	}
	st := getListenManager().Status()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"session_id": sessionID,
		"status":     st.State,
		"stt_ready":  st.STTReady,
	})
}

func handleAssistantListenChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(r.Header.Get("X-NJ-Listen-Session"))
	}
	const maxBody = 6 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	audio, err := io.ReadAll(r.Body)
	if err != nil {
		writeListenError(w, listen.ErrChunkTooLarge, http.StatusRequestEntityTooLarge)
		return
	}
	partial, err := getListenManager().Chunk(r.Context(), sessionID, audio)
	if err != nil {
		writeListenError(w, err, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"partial_text": partial,
		"session_id":   sessionID,
	})
}

func handleAssistantListenStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	result, err := getListenManager().Stop(r.Context(), body.SessionID)
	if err != nil {
		writeListenError(w, err, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func handleAssistantListenCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	if err := getListenManager().Cancel(body.SessionID); err != nil {
		writeListenError(w, err, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}
