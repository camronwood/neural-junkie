package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/lora/export"
	"github.com/camronwood/neural-junkie/internal/lora/train"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var loraTrainMgr *train.Manager

type hubMessageAdapter struct{}

func (hubMessageAdapter) GetMessages(channel string, limit int) ([]*protocol.Message, error) {
	return chatHub.GetMessages(channel, limit)
}

func (hubMessageAdapter) GetThreadMessages(threadID string, limit int) ([]*protocol.Message, error) {
	return chatHub.GetThreadMessages(threadID, limit)
}

type collabSnapshotAdapter struct{}

func (collabSnapshotAdapter) GetCollaborationSnapshot(id string) (*collaboration.Collaboration, error) {
	return chatHub.GetCollaborationManager().GetCollaborationSnapshot(id)
}

func initLoraTrainManager() {
	if loraTrainMgr != nil {
		return
	}
	root := repoRoot()
	script := filepath.Join(root, "scripts", "lora_train.py")
	loraTrainMgr = train.NewManager("", script, train.ResolvePython(root), hubMessageAdapter{}, collabSnapshotAdapter{})
}

func repoRoot() string {
	if root := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_ROOT")); root != "" {
		return root
	}
	candidates := []string{
		".",
		filepath.Join("..", ".."),
		filepath.Join(filepath.Dir(os.Args[0]), "..", ".."),
	}
	for _, c := range candidates {
		p := filepath.Join(c, "scripts", "lora_train.py")
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "."
}

func handleLoraTrainRoute(w http.ResponseWriter, r *http.Request) {
	initLoraTrainManager()
	path := strings.TrimPrefix(r.URL.Path, "/api/lora/train")
	path = strings.Trim(path, "/")
	if path == "preview" {
		handleLoraTrainPreview(w, r)
		return
	}
	if path == "" {
		handleLoraTrainStart(w, r)
		return
	}
	handleLoraTrainByID(w, r, path)
}

func handleLoraTrainPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	req := export.Request{
		Source:    export.SourceKind(strings.TrimSpace(q.Get("source"))),
		SourceID:  strings.TrimSpace(q.Get("source_id")),
		ThreadID:  strings.TrimSpace(q.Get("thread_id")),
		AgentName: strings.TrimSpace(q.Get("agent_name")),
	}
	n, err := loraTrainMgr.Preview(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"row_count": n,
		"min_rows":  export.MinRows,
	})
}

func handleLoraTrainStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body train.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	job, err := loraTrainMgr.Start(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func handleLoraTrainByID(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		job, ok := loraTrainMgr.Get(id)
		if !ok {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, job)
	case http.MethodDelete:
		if err := loraTrainMgr.Cancel(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		job, _ := loraTrainMgr.Get(id)
		writeJSON(w, http.StatusOK, job)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
