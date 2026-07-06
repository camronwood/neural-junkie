package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hfhub"
	"github.com/camronwood/neural-junkie/internal/lora/eval"
	"github.com/camronwood/neural-junkie/internal/lora/registry"
	"github.com/camronwood/neural-junkie/internal/lora/train"
	"github.com/camronwood/neural-junkie/internal/packs"
)

var loraAdapterRegistry *registry.Store

func initLoraAdapterRegistry() {
	if loraAdapterRegistry != nil {
		return
	}
	reg, err := registry.NewStore("")
	if err != nil {
		return
	}
	loraAdapterRegistry = reg
}

func initLoraTrainManager() {
	if loraTrainMgr != nil {
		return
	}
	initLoraAdapterRegistry()
	root := repoRoot()
	script := filepath.Join(root, "scripts", "lora_train.py")
	loraTrainMgr = train.NewManager("", script, train.ResolvePython(root), hubMessageAdapter{}, collabSnapshotAdapter{})
	if loraAdapterRegistry != nil {
		loraTrainMgr.SetRegistry(loraAdapterRegistry)
	}
	loraTrainMgr.SetOnComplete(loraTrainOnComplete)
	loraTrainMgr.SetSidecarBaseURL(func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackSpecialistTuning)
	})
}

func loraTrainOnComplete(ctx context.Context, job *train.Job, artifactDir, datasetPath string) error {
	if loraAdapterRegistry == nil {
		return nil
	}
	hash, _ := registry.DatasetHashFile(datasetPath)
	entry, err := loraAdapterRegistry.Register(registry.RegisterInput{
		OllamaTag:     job.OllamaTag,
		BaseOllamaTag: job.BaseOllamaTag,
		AgentID:       job.AgentID,
		Source:        string(job.Source),
		SourceID:      job.SourceID,
		JobID:         job.ID,
		RowCount:      job.RowCount,
		DatasetHash:   hash,
		ArtifactDir:   artifactDir,
		EvalScore:     job.EvalScore,
	})
	if err != nil {
		return err
	}
	job.AdapterID = entry.ID
	if job.EvalScore == 0 {
		score := runLoRAEval(ctx, job)
		job.EvalScore = score.Score
		_ = loraAdapterRegistry.SetEvalScore(entry.ID, score.Score)
		if appConfig != nil {
			policy := appConfig.ResolvedLoRAPolicy()
			if policy.RequireEvalToAssign && !score.Passed && score.Score < policy.EvalMinScore {
				return fmt.Errorf("eval score %.2f below minimum %.2f", score.Score, policy.EvalMinScore)
			}
		}
	}
	return nil
}

func loraEvalQuestions(job *train.Job) []eval.Question {
	agentType := ""
	if job != nil && appConfig != nil && job.AgentID != "" {
		if info, err := chatHub.GetAgent(job.AgentID); err == nil && info != nil {
			agentType = string(info.Type)
		}
	}
	packDir := ""
	probeRel := ""
	if appConfig != nil {
		probeRel = appConfig.LoRAEvalProbesForAgentType(agentType)
		if dir, err := packs.InstalledPackDir(config.PackSpecialistTuning); err == nil {
			packDir = dir
		}
	}
	if job == nil {
		return nil
	}
	return eval.QuestionsForAgentType(agentType, job.AgentID, packDir, probeRel)
}

func runLoRAEval(ctx context.Context, job *train.Job) eval.Result {
	questions := loraEvalQuestions(job)
	min := eval.DefaultMinScore
	if appConfig != nil {
		min = appConfig.ResolvedLoRAPolicy().EvalMinScore
		if min <= 0 {
			min = eval.DefaultMinScore
		}
	}
	if len(questions) == 0 {
		return eval.Result{Score: 1, Passed: true, MinScore: min}
	}
	if appConfig == nil {
		return eval.Result{Score: 1, Passed: true, MinScore: min}
	}
	pcfg := appConfig.GetProvider("ollama-local")
	if pcfg == nil {
		return eval.Result{Score: 0.5, Passed: true}
	}
	util := *pcfg
	util.Model = job.OllamaTag
	prov, err := ai.ProviderFromConfig(&util)
	if err != nil {
		return eval.Result{Score: 0.5, Passed: true}
	}
	res := eval.Run(ctx, prov, job.OllamaTag, questions)
	res.MinScore = min
	res.Passed = res.Score >= min
	return res
}

func handleLoraAdaptersRoute(w http.ResponseWriter, r *http.Request) {
	if !requireLoRACapability(w, capLoRATraining) {
		return
	}
	initLoraAdapterRegistry()
	if loraAdapterRegistry == nil {
		http.Error(w, "registry unavailable", http.StatusServiceUnavailable)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/lora/adapters")
	path = strings.Trim(path, "/")
	if path == "" {
		handleLoraAdaptersList(w, r)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]
	if len(parts) == 2 && parts[1] == "activate" {
		handleLoraAdapterActivate(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "eval" {
		handleLoraAdapterEval(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "publish" {
		handleLoraAdapterPublish(w, r, id)
		return
	}
	handleLoraAdapterGet(w, r, id)
}

func handleLoraAdaptersList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	list := loraAdapterRegistry.List(q.Get("agent_id"), q.Get("ollama_tag"))
	writeJSON(w, http.StatusOK, map[string]any{"adapters": list})
}

func handleLoraAdapterGet(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entry, ok := loraAdapterRegistry.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func handleLoraAdapterActivate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entry, err := loraAdapterRegistry.Activate(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if appConfig != nil {
		policy := appConfig.ResolvedLoRAPolicy()
		if policy.RequireEvalToAssign && entry.EvalScore > 0 && entry.EvalScore < policy.EvalMinScore {
			http.Error(w, fmt.Sprintf("eval score %.2f below minimum %.2f — retrain or disable require_eval_to_assign", entry.EvalScore, policy.EvalMinScore), http.StatusBadRequest)
			return
		}
	}
	adapterPath := filepath.Join(entry.ArtifactDir, "adapter_model.safetensors")
	if err := hfhub.ImportAdapterToOllama(r.Context(), entry.BaseOllamaTag, adapterPath, entry.OllamaTag); err != nil {
		http.Error(w, "compose: "+err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func handleLoraAdapterEval(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entry, ok := loraAdapterRegistry.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	job := &train.Job{OllamaTag: entry.OllamaTag, AgentID: entry.AgentID, EvalScore: entry.EvalScore}
	res := runLoRAEval(r.Context(), job)
	writeJSON(w, http.StatusOK, res)
}

func handleLoraAdapterPublish(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entry, ok := loraAdapterRegistry.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		RepoID string `json:"repo_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token := hfToken()
	if err := hfhub.PublishAdapter(r.Context(), token, body.RepoID, entry.ArtifactDir); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"published": true, "repo_id": body.RepoID})
}

func hfToken() string {
	if appConfig != nil && appConfig.HF.Token != "" {
		return appConfig.HF.Token
	}
	return strings.TrimSpace(os.Getenv("HF_TOKEN"))
}
