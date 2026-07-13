package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	streamint "github.com/camronwood/neural-junkie/internal/integrations/stream"
)

var (
	streamManager   *streamint.Manager
	streamStore     *streamint.Store
	streamBridgeCtx context.Context
)

func startStreamManager(ctx context.Context) {
	if chatHub == nil {
		return
	}
	store, err := streamint.NewStore()
	if err != nil {
		log.Printf("[stream] store init: %v", err)
		return
	}
	streamStore = store
	dispatcher := streamint.NewDispatcher(chatHub)
	mgr := streamint.NewManager(store, dispatcher)
	if err := mgr.Start(ctx); err != nil {
		log.Printf("[stream] manager start: %v", err)
		return
	}
	streamManager = mgr
}

func stopStreamManager() {
	if streamManager != nil {
		streamManager.Stop()
		streamManager = nil
	}
}

func ensureStreamManager(ctx context.Context) (*streamint.Manager, error) {
	if streamManager != nil {
		return streamManager, nil
	}
	if streamStore == nil {
		store, err := streamint.NewStore()
		if err != nil {
			return nil, err
		}
		streamStore = store
	}
	dispatcher := streamint.NewDispatcher(chatHub)
	mgr := streamint.NewManager(streamStore, dispatcher)
	streamManager = mgr
	if ctx != nil {
		if err := mgr.Start(ctx); err != nil {
			return nil, err
		}
	}
	return streamManager, nil
}

func handleStreamStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if streamManager == nil {
		writeJSON(w, http.StatusOK, streamint.ManagerStatus{Running: false, Subscriptions: []streamint.SubStatus{}})
		return
	}
	writeJSON(w, http.StatusOK, streamManager.Status())
}

func handleStreamRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	ctx := streamBridgeCtx
	if ctx == nil {
		ctx = context.Background()
	}
	stopStreamManager()
	startStreamManager(ctx)
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func handleStreamSubscriptions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/stream/subscriptions")
	path = strings.Trim(path, "/")

	store, err := streamint.NewStore()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	streamStore = store

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, store.List())
		case http.MethodPost:
			if _, ok := ensureMutationAccess(w, r, ""); !ok {
				return
			}
			var sub streamint.Subscription
			if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
				return
			}
			saved, err := store.Upsert(sub)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			restartStreamAfterChange()
			writeJSON(w, http.StatusOK, saved)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	if rest == "test" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var body struct {
			Payload string `json:"payload"`
			Topic   string `json:"topic"`
		}
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		if body.Payload == "" {
			body.Payload = "{}"
		}
		mgr, err := ensureStreamManager(streamBridgeCtx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		res, err := mgr.HandleTest(r.Context(), id, body.Payload, body.Topic)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
		return
	}

	if rest != "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPut:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var sub streamint.Subscription
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		sub.ID = id
		saved, err := store.Upsert(sub)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		restartStreamAfterChange()
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		if err := store.Delete(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		restartStreamAfterChange()
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func restartStreamAfterChange() {
	if streamManager == nil {
		return
	}
	ctx := streamBridgeCtx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := streamManager.OnSubscriptionsChanged(ctx); err != nil {
		log.Printf("[stream] reload after change: %v", err)
	}
}
