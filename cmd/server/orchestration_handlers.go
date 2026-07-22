package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/orchestration"
)

func handleOrchestrationRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runID := strings.TrimSpace(r.URL.Query().Get("run_id"))
	if runID == "" {
		http.Error(w, "run_id is required", http.StatusBadRequest)
		return
	}
	var after int64
	if raw := strings.TrimSpace(r.URL.Query().Get("after_event_id")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			http.Error(w, "after_event_id must be a non-negative integer", http.StatusBadRequest)
			return
		}
		after = value
	}
	snapshot, err := chatHub.GetOrchestrationSnapshot(r.Context(), runID, after)
	if err != nil {
		if errors.Is(err, orchestration.ErrNotFound) {
			http.Error(w, "run not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

func handleOrchestrationDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var deployment orchestration.Deployment
	if err := json.NewDecoder(r.Body).Decode(&deployment); err != nil {
		http.Error(w, "invalid deployment JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !deployment.Enabled {
		deployment.Enabled = true
	}
	if deployment.NextRunAt.IsZero() && deployment.Schedule != "" {
		next, err := orchestration.NextScheduleTime(deployment.Schedule, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		deployment.NextRunAt = next
	}
	if err := chatHub.UpsertOrchestrationDeployment(r.Context(), deployment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(deployment)
}

func handleOrchestrationAutomations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var automation orchestration.Automation
	if err := json.NewDecoder(r.Body).Decode(&automation); err != nil {
		http.Error(w, "invalid automation JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !automation.Enabled {
		automation.Enabled = true
	}
	if err := chatHub.UpsertOrchestrationAutomation(r.Context(), automation); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(automation)
}

func handleOrchestrationWorkerRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var worker orchestration.Worker
	if err := json.NewDecoder(r.Body).Decode(&worker); err != nil {
		http.Error(w, "invalid worker JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	worker.Status = orchestration.WorkerReady
	if err := chatHub.RegisterOrchestrationWorker(r.Context(), worker); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleOrchestrationWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		WorkerID string `json:"worker_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.WorkerID) == "" {
		http.Error(w, "worker_id is required", http.StatusBadRequest)
		return
	}
	if err := chatHub.HeartbeatOrchestrationWorker(r.Context(), body.WorkerID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleOrchestrationWorkerClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		WorkerID     string   `json:"worker_id"`
		Queue        string   `json:"queue"`
		Capabilities []string `json:"capabilities"`
		LeaseSeconds int      `json:"lease_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.WorkerID) == "" {
		http.Error(w, "worker_id is required", http.StatusBadRequest)
		return
	}
	lease := time.Duration(body.LeaseSeconds) * time.Second
	task, attempt, err := chatHub.ClaimOrchestrationWork(r.Context(), orchestration.ClaimOptions{
		WorkerID: body.WorkerID, Queue: body.Queue, Capabilities: body.Capabilities, Lease: lease,
	})
	if err != nil {
		if errors.Is(err, orchestration.ErrNotReady) || errors.Is(err, orchestration.ErrConcurrencyFull) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"task":        task,
		"attempt":     attempt,
		"lease_token": attempt.LeaseToken,
	})
}

func handleOrchestrationWorkerComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		AttemptID   string         `json:"attempt_id"`
		LeaseToken  string         `json:"lease_token"`
		Value       string         `json:"value"`
		ContentType string         `json:"content_type"`
		Metadata    map[string]any `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		body.AttemptID == "" || body.LeaseToken == "" {
		http.Error(w, "attempt_id and lease_token are required", http.StatusBadRequest)
		return
	}
	if err := chatHub.CompleteOrchestrationWork(
		r.Context(), body.AttemptID, body.LeaseToken, []byte(body.Value), body.ContentType, body.Metadata,
	); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleOrchestrationWorkHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		AttemptID     string         `json:"attempt_id"`
		LeaseToken    string         `json:"lease_token"`
		ExtendSeconds int            `json:"extend_seconds"`
		Progress      map[string]any `json:"progress"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		body.AttemptID == "" || body.LeaseToken == "" {
		http.Error(w, "attempt_id and lease_token are required", http.StatusBadRequest)
		return
	}
	if err := chatHub.HeartbeatOrchestrationWork(
		r.Context(), body.AttemptID, body.LeaseToken,
		time.Duration(body.ExtendSeconds)*time.Second, body.Progress,
	); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleOrchestrationWorkSpawn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ParentTaskID string             `json:"parent_task_id"`
		Task         orchestration.Task `json:"task"`
		MaxTasks     int                `json:"max_tasks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		body.ParentTaskID == "" || body.Task.RunID == "" || body.Task.ID == "" {
		http.Error(w, "parent_task_id and task run/id are required", http.StatusBadRequest)
		return
	}
	if err := chatHub.SpawnOrchestrationTask(r.Context(), body.ParentTaskID, body.Task, body.MaxTasks); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func handleOrchestrationWorkerFail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		AttemptID  string `json:"attempt_id"`
		LeaseToken string `json:"lease_token"`
		Error      string `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		body.AttemptID == "" || body.LeaseToken == "" {
		http.Error(w, "attempt_id and lease_token are required", http.StatusBadRequest)
		return
	}
	if err := chatHub.FailOrchestrationWork(r.Context(), body.AttemptID, body.LeaseToken, body.Error); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
