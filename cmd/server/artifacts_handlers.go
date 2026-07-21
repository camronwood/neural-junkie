package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/artifacts"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const maxArtifactRequestBytes = artifacts.DefaultMaxPayloadBytes + (1 << 20)

var (
	canvasStoreOnce sync.Once
	canvasStore     *artifacts.Store
	canvasStoreErr  error
)

func neuralCanvasStore() (*artifacts.Store, error) {
	canvasStoreOnce.Do(func() {
		canvasStore, canvasStoreErr = artifacts.NewStore("")
	})
	return canvasStore, canvasStoreErr
}

func handleArtifacts(w http.ResponseWriter, r *http.Request) {
	store, err := neuralCanvasStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		channel := strings.TrimSpace(r.URL.Query().Get("channel_id"))
		if !ensureChannelReadAccess(w, r, channel) {
			return
		}
		items, err := store.List(artifacts.Filter{
			Kind:            r.URL.Query().Get("kind"),
			WorkspaceID:     r.URL.Query().Get("workspace_id"),
			ProjectID:       r.URL.Query().Get("project_id"),
			ChannelID:       channel,
			CollaborationID: r.URL.Query().Get("collaboration_id"),
			RendererID:      r.URL.Query().Get("renderer_id"),
			Capability:      r.URL.Query().Get("capability"),
		})
		writeArtifactJSON(w, items, err)
	case http.MethodPost:
		var artifact artifacts.Artifact
		if !decodeArtifactJSON(w, r, &artifact) {
			return
		}
		sess, ok := ensureMutationAccess(w, r, artifact.Links.ChannelID)
		if !ok {
			return
		}
		if len(artifact.Provenance) == 0 {
			artifact.Provenance = []artifacts.SourceReference{{
				Kind:  "user",
				Label: sess.Username,
			}}
		}
		created, err := store.Create(artifact)
		if err == nil {
			broadcastArtifactChange(created, "created")
		}
		writeArtifactJSON(w, created, err)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleArtifactByID(w http.ResponseWriter, r *http.Request) {
	store, err := neuralCanvasStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/artifacts/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "artifact id is required", http.StatusBadRequest)
		return
	}
	id := parts[0]
	if len(parts) > 1 {
		handleArtifactAction(w, r, store, id, parts[1:])
		return
	}
	current, getErr := store.Get(id)
	if getErr != nil {
		writeArtifactJSON(w, nil, getErr)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !ensureChannelReadAccess(w, r, current.Links.ChannelID) {
			return
		}
		writeArtifactJSON(w, current, nil)
	case http.MethodPut:
		if _, ok := ensureMutationAccess(w, r, current.Links.ChannelID); !ok {
			return
		}
		var update artifacts.Artifact
		if !decodeArtifactJSON(w, r, &update) {
			return
		}
		update.ID = id
		expected, ok := expectedArtifactRevision(w, r, update.Revision)
		if !ok {
			return
		}
		updated, err := store.Update(update, expected)
		if err == nil {
			broadcastArtifactChange(updated, "updated")
		}
		writeArtifactJSON(w, updated, err)
	case http.MethodDelete:
		if _, ok := ensureMutationAccess(w, r, current.Links.ChannelID); !ok {
			return
		}
		expected, ok := expectedArtifactRevision(w, r, current.Revision)
		if !ok {
			return
		}
		err := store.Delete(id, expected)
		if err == nil {
			broadcastArtifactChange(current, "deleted")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeArtifactJSON(w, nil, err)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleArtifactAction(w http.ResponseWriter, r *http.Request, store *artifacts.Store, id string, parts []string) {
	current, err := store.Get(id)
	if err != nil {
		writeArtifactJSON(w, nil, err)
		return
	}
	action := parts[0]
	switch action {
	case "revisions":
		if !ensureChannelReadAccess(w, r, current.Links.ChannelID) {
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if len(parts) == 1 {
			revisions, err := store.ListRevisions(id)
			writeArtifactJSON(w, revisions, err)
			return
		}
		revision, parseErr := strconv.ParseUint(parts[1], 10, 64)
		if parseErr != nil {
			http.Error(w, "invalid revision", http.StatusBadRequest)
			return
		}
		snapshot, err := store.GetRevision(id, revision)
		writeArtifactJSON(w, snapshot, err)
	case "duplicate":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if _, ok := ensureMutationAccess(w, r, current.Links.ChannelID); !ok {
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		if !decodeArtifactJSON(w, r, &body) {
			return
		}
		duplicate, err := store.Duplicate(id, strings.TrimSpace(body.ID))
		if err == nil {
			broadcastArtifactChange(duplicate, "created")
		}
		writeArtifactJSON(w, duplicate, err)
	case "assets":
		if len(parts) < 2 {
			http.Error(w, "asset name is required", http.StatusBadRequest)
			return
		}
		name := parts[1]
		if r.Method == http.MethodGet {
			if !ensureChannelReadAccess(w, r, current.Links.ChannelID) {
				return
			}
			data, err := store.GetAsset(id, name)
			if err != nil {
				writeArtifactJSON(w, nil, err)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(data)
			return
		}
		if r.Method == http.MethodPut {
			if _, ok := ensureMutationAccess(w, r, current.Links.ChannelID); !ok {
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, artifacts.DefaultMaxAssetBytes)
			data, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				http.Error(w, "asset exceeds size limit", http.StatusRequestEntityTooLarge)
				return
			}
			if err := store.PutAsset(id, name, data); err != nil {
				writeArtifactJSON(w, nil, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	case "export":
		handleArtifactExport(w, r, current)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func handleArtifactExport(w http.ResponseWriter, r *http.Request, artifact *artifacts.Artifact) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		WorkspaceID string `json:"workspace_id"`
		Path        string `json:"path"`
		Channel     string `json:"channel"`
	}
	if !decodeArtifactJSON(w, r, &body) {
		return
	}
	if _, ok := ensureMutationAccess(w, r, firstNonEmpty(body.Channel, artifact.Links.ChannelID)); !ok {
		return
	}
	workspace, ok := workspaceManager.GetWorkspace(strings.TrimSpace(body.WorkspaceID))
	if !ok || workspace == nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	relative := strings.TrimPrefix(filepath.Clean(strings.TrimSpace(body.Path)), string(filepath.Separator))
	if relative == "" || relative == "." {
		relative = artifact.ID + ".canvas.json"
	}
	target, err := pathutil.WithinRoot(workspace.Path, filepath.Join(workspace.Path, relative))
	if err != nil {
		http.Error(w, "export path is outside workspace", http.StatusBadRequest)
		return
	}
	content, err := artifactExportContent(artifact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	backend, code, message := backendForWorkspace(workspace.ID)
	if backend == nil || code != 0 {
		http.Error(w, message, http.StatusBadRequest)
		return
	}
	oldContent := ""
	operation := filechange.FileOperationCreate
	if data, readErr := backend.ReadFile(r.Context(), relative); readErr == nil {
		oldContent = string(data)
		operation = filechange.FileOperationEdit
	}
	manager := chatHub.GetFileChangeManager()
	change, err := manager.ProposeFileChange(
		operation,
		target,
		"",
		"",
		oldContent,
		content,
		protocol.AgentInfo{ID: "neural-canvas", Name: "Neural Canvas", Type: protocol.AgentTypeAssistant},
		firstNonEmpty(body.Channel, artifact.Links.ChannelID, "general"),
	)
	if err == nil {
		err = manager.BindExecutionContext(change.ID, workspace.Path, backendForWorkspaceRoot(workspace.Path))
	}
	writeArtifactJSON(w, change, err)
}

func artifactExportContent(artifact *artifacts.Artifact) (string, error) {
	if artifact.Fallback != nil && (artifact.Fallback.MediaType == "text/markdown" || artifact.Fallback.MediaType == "text/plain") {
		var text string
		if json.Unmarshal(artifact.Fallback.Data, &text) == nil {
			return text, nil
		}
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode artifact export: %w", err)
	}
	return string(data) + "\n", nil
}

func expectedArtifactRevision(w http.ResponseWriter, r *http.Request, fallback uint64) (uint64, bool) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	raw = strings.Trim(raw, `"`)
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("expected_revision"))
	}
	if raw == "" && fallback > 0 {
		return fallback, true
	}
	revision, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || revision == 0 {
		http.Error(w, "expected revision is required", http.StatusPreconditionRequired)
		return 0, false
	}
	return revision, true
}

func decodeArtifactJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxArtifactRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeArtifactJSON(w http.ResponseWriter, value any, err error) {
	if err != nil {
		switch {
		case errors.Is(err, artifacts.ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, artifacts.ErrConflict):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, artifacts.ErrTooLarge):
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		case errors.Is(err, artifacts.ErrInvalidID), errors.Is(err, artifacts.ErrInvalidAsset):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, artifacts.ErrCorrupt):
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func broadcastArtifactChange(artifact *artifacts.Artifact, action string) {
	if chatHub == nil || artifact == nil || strings.TrimSpace(artifact.Links.ChannelID) == "" {
		return
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeArtifactChanged,
		artifact.Links.ChannelID,
		protocol.AgentInfo{ID: "neural-canvas", Name: "Neural Canvas", Type: protocol.AgentTypeAssistant},
		artifact.Title,
	)
	msg.SetArtifactReference(protocol.ArtifactReference{
		ID:                 artifact.ID,
		Title:              artifact.Title,
		RendererID:         artifact.Renderer.ID,
		RendererAPIVersion: parseRendererAPIVersion(artifact.Renderer.APIVersion),
		MediaType:          artifact.Renderer.MediaType,
		Revision:           int64(artifact.Revision),
		WorkspaceID:        artifact.Links.WorkspaceID,
		Action:             action,
	})
	_ = chatHub.SendMessage(msg)
}

func parseRendererAPIVersion(value string) int {
	version, _ := strconv.Atoi(strings.TrimPrefix(value, "v"))
	return version
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
