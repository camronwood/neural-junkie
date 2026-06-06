package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	climgr "github.com/camronwood/neural-junkie/internal/cli"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var cliMgr = climgr.NewManager()

func cliProviderAPIKey(providerName string) string {
	if appConfig == nil {
		return ""
	}
	if p := appConfig.GetProvider(providerName); p != nil {
		return strings.TrimSpace(p.APIKey)
	}
	return ""
}

func handleCLIAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	statuses := cliMgr.ListStatus(cliProviderAPIKey)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": statuses,
	})
}

func handleCLIAgentsSubRoute(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/cli-agents/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	cliType := parts[0]
	action := parts[1]

	switch action {
	case "install":
		handleCLIAgentInstall(w, r, cliType)
	case "auth":
		handleCLIAgentAuth(w, r, cliType)
	case "probe":
		handleCLIAgentProbe(w, r, cliType)
	case "activate":
		handleCLIAgentActivate(w, r, cliType)
	default:
		http.NotFound(w, r)
	}
}

func handleCLIAgentInstall(w http.ResponseWriter, r *http.Request, cliType string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	err := cliMgr.Install(r.Context(), cliType, func(msg string) {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		flusher.Flush()
	})
	if err != nil {
		fmt.Fprintf(w, "data: ERROR: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	fmt.Fprintf(w, "data: DONE\n\n")
	flusher.Flush()
}

func handleCLIAgentAuth(w http.ResponseWriter, r *http.Request, cliType string) {
	switch r.Method {
	case http.MethodGet:
		info, err := cliMgr.AuthLoginInfo(cliType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)

	case http.MethodPost:
		var body struct {
			APIKey string `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		cfg, ok := agent.GetCLIAgentConfig(cliType)
		if !ok {
			http.Error(w, "unknown CLI type", http.StatusBadRequest)
			return
		}
		if err := climgr.ApplyAPIKey(cfg, body.APIKey); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if appConfig != nil {
			key := strings.TrimSpace(body.APIKey)
			if appConfig.GetProvider(cfg.ProviderName) == nil {
				_ = appConfig.AddProvider(config.ProviderConfig{
					ID:     cfg.ProviderName,
					Type:   cfg.ProviderName,
					Name:   cfg.DefaultName + " (Auto-detected)",
					APIKey: key,
				})
			} else {
				for i := range appConfig.AI.Providers {
					if appConfig.AI.Providers[i].ID == cfg.ProviderName {
						appConfig.AI.Providers[i].APIKey = key
						break
					}
				}
			}
			_ = appConfig.Save()
		}
		st, _ := cliMgr.StatusFor(cliType, cliProviderAPIKey)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(st)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCLIAgentProbe(w http.ResponseWriter, r *http.Request, cliType string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st, err := cliMgr.StatusFor(cliType, cliProviderAPIKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}

func handleCLIAgentActivate(w http.ResponseWriter, r *http.Request, cliType string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, ok := agent.GetCLIAgentConfig(cliType)
	if !ok {
		http.Error(w, "unknown CLI type", http.StatusBadRequest)
		return
	}

	st, _ := cliMgr.StatusFor(cliType, cliProviderAPIKey)
	if !st.Installed {
		http.Error(w, fmt.Sprintf("%s CLI is not installed", cfg.DefaultName), http.StatusBadRequest)
		return
	}
	if st.AuthState == climgr.AuthNeedsAuth {
		http.Error(w, fmt.Sprintf("%s CLI needs authentication first", cfg.DefaultName), http.StatusBadRequest)
		return
	}

	defaultWorkDir, err := os.Getwd()
	if err != nil {
		defaultWorkDir = "."
	}

	activated, activateErr := activateCLIAgentFromConfig(cfg, defaultWorkDir)
	resp := map[string]interface{}{
		"activated": activated,
		"type":      cliType,
		"name":      cfg.DefaultName,
	}
	if activateErr != nil {
		resp["error"] = activateErr.Error()
	}
	if cliAgentAlreadyActive(cfg) {
		resp["already_active"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	if activateErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(resp)
}

func cliAgentAlreadyActive(cfg agent.CLIAgentConfig) bool {
	if chatHub == nil {
		return false
	}
	for _, a := range chatHub.ListAgents() {
		if a == nil {
			continue
		}
		if strings.EqualFold(a.Name, cfg.DefaultName) && a.Type == protocol.AgentTypeCLI {
			return true
		}
	}
	return false
}

// activateCLIAgentFromConfig registers and starts a CLI agent if not already active.
func activateCLIAgentFromConfig(cfg agent.CLIAgentConfig, defaultWorkDir string) (bool, error) {
	if cliAgentAlreadyActive(cfg) {
		return false, nil
	}

	resolved, found := agent.ResolveCLIWithPATH(cfg, pathutil.EnhancedPATH())
	if !found {
		resolved, found = agent.ResolveCLI(cfg)
	}
	if !found {
		return false, fmt.Errorf("%s CLI (%s) not found on PATH", cfg.DefaultName, agent.CLIProbeLabel(cfg))
	}

	workDir := defaultWorkDir
	if cfg.WorkDirEnv != "" {
		if envDir := os.Getenv(cfg.WorkDirEnv); envDir != "" {
			workDir = envDir
		}
	}

	opts := []ai.CLIAgentOption{
		ai.WithBaseArgs(resolved.BaseArgs),
		ai.WithModel(cfg.ModelName),
	}

	model := resolveCLIProviderModel(cfg)
	if model != "" {
		opts = append(opts, ai.WithModel(model))
		ai.SetCLIProviderModelOverride(cfg.ProviderName, model)
	}
	if cfg.Type == "gemini" && model != "" {
		_ = os.Setenv("GEMINI_MODEL", model)
		opts = append(opts, ai.WithEnv("GEMINI_MODEL", model))
	}
	provider := ai.NewCLIAgentProvider(resolved.Command, workDir, cfg.ProviderName, opts...)

	for _, envKey := range cfg.EnvVars {
		if val := os.Getenv(envKey); val != "" {
			provider.Env[envKey] = val
		}
	}
	if appConfig != nil {
		if p := appConfig.GetProvider(cfg.ProviderName); p != nil && strings.TrimSpace(p.APIKey) != "" {
			for _, envKey := range cfg.EnvVars {
				provider.Env[envKey] = strings.TrimSpace(p.APIKey)
			}
		}
	}

	if appConfig != nil && appConfig.GetProvider(cfg.ProviderName) == nil {
		autoProvider := config.ProviderConfig{
			ID:      cfg.ProviderName,
			Type:    cfg.ProviderName,
			Name:    cfg.DefaultName + " (Auto-detected)",
			WorkDir: workDir,
		}
		if model != "" {
			autoProvider.Model = model
		}
		if err := appConfig.AddProvider(autoProvider); err == nil {
			_ = appConfig.Save()
		}
	}

	if cfg.Type == "gemini" {
		configureGeminiApprovalHook()
	}

	cliAgent := agent.NewCLIAgentFromConfig(cfg, cfg.DefaultName, provider, chatHub)
	cliAgent.SetCollabClient(chatHub.NewCollaborationClientAdapter())

	if cfg.ApprovalMode != "" {
		cliAgent.Info.ApprovalMode = cfg.ApprovalMode
	}

	if err := chatHub.RegisterAgent(&cliAgent.Info); err != nil {
		return false, fmt.Errorf("register agent: %w", err)
	}
	if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
		if ch, ok := commandHandler.(*hub.CommandHandler); ok {
			ch.RegisterRuntimeAgent(cliAgent)
		}
	}

	if err := chatHub.JoinChannel(cliAgent.Info.ID, "general", cfg.JoinMessage); err != nil {
		return false, fmt.Errorf("join channel: %w", err)
	}

	ctx := context.Background()
	go func() {
		if err := cliAgent.Start(ctx, "general"); err != nil {
			log.Printf("❌ Failed to start %s CLI agent: %v", cfg.DefaultName, err)
		}
	}()

	log.Printf("✅ %s CLI agent activated (workDir: %s)", cfg.DefaultName, workDir)
	return true, nil
}
