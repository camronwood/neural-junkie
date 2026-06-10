package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/pathutil"
)

func syncCLIProviderModelsFromConfig() {
	if appConfig == nil {
		return
	}
	for i := range appConfig.AI.Providers {
		p := appConfig.AI.Providers[i]
		syncCLIProviderModelToRuntime(&p)
	}
}

// initializeCLIAgents creates and starts any CLI-backed agents based on environment configuration.
// Each CLI agent is independent; if one binary is missing, the others still start.

func initializeCLIAgents() {
	defaultWorkDir, err := os.Getwd()
	if err != nil {
		log.Printf("⚠️  Failed to get working directory for CLI agents: %v", err)
		return
	}

	for _, cliType := range agent.ListCLIAgentTypes() {
		cfg, _ := agent.GetCLIAgentConfig(cliType)
		initCLIAgentFromConfig(cfg, defaultWorkDir)
	}
}

func resolveCLIProviderModel(cfg agent.CLIAgentConfig) string {
	model := ""
	if appConfig != nil {
		if p := appConfig.GetProvider(cfg.ProviderName); p != nil {
			model = strings.TrimSpace(p.Model)
		}
	}
	if cfg.Type == "gemini" {
		if model == "" {
			model = strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
		}
		if model == "" {
			model = "gemini-2.5-flash"
		}
	}
	return model
}

func syncCLIProviderModelToRuntime(p *config.ProviderConfig) {
	if p == nil {
		return
	}
	model := strings.TrimSpace(p.Model)
	switch p.Type {
	case "gemini-cli":
		if model == "" {
			model = "gemini-2.5-flash"
			p.Model = model
		}
		_ = os.Setenv("GEMINI_MODEL", model)
		ai.SetCLIProviderModelOverride(p.Type, model)
	case "cursor-cli", "claude-cli", "codex-cli", "copilot-cli", "aider-cli", "opencode-cli":
		ai.SetCLIProviderModelOverride(p.Type, model)
	default:
		if strings.HasSuffix(p.Type, "-cli") {
			ai.SetCLIProviderModelOverride(p.Type, model)
		}
	}
}

func initCLIAgentFromConfig(cfg agent.CLIAgentConfig, defaultWorkDir string) {
	log.Printf("🤖 Checking for %s CLI agent (%s)...", cfg.DefaultName, agent.CLIProbeLabel(cfg))

	if cliAgentAlreadyActive(cfg) {
		log.Printf("ℹ️  %s CLI agent already active", cfg.DefaultName)
		return
	}

	pathEnv := pathutil.EnhancedPATH()
	if _, found := agent.ResolveCLIWithPATH(cfg, pathEnv); !found {
		if _, found = agent.ResolveCLI(cfg); !found {
			log.Printf("ℹ️  %s CLI (%s) not found on PATH — skipping. %s", cfg.DefaultName, agent.CLIProbeLabel(cfg), cfg.InstallHint)
			return
		}
	}

	activated, err := activateCLIAgentFromConfig(cfg, defaultWorkDir)
	if err != nil {
		log.Printf("❌ Failed to activate %s CLI agent: %v", cfg.DefaultName, err)
		return
	}
	if activated {
		log.Printf("✅ %s CLI agent started (workDir: %s)", cfg.DefaultName, defaultWorkDir)
	}
}

// configureGeminiApprovalHook installs the Neural Junkie BeforeTool hook into
// Gemini CLI's settings.json so tool calls are routed through the approval UI.

func configureGeminiApprovalHook() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("⚠️  Cannot determine home directory for Gemini hook config: %v", err)
		return
	}

	settingsDir := filepath.Join(homeDir, ".gemini")
	settingsPath := filepath.Join(settingsDir, "settings.json")

	// Find the hook binary
	hookBin, err := exec.LookPath("tool-approval-hook")
	if err != nil {
		// Try relative to server binary
		exePath, _ := os.Executable()
		hookBin = filepath.Join(filepath.Dir(exePath), "tool-approval-hook")
		if _, err := os.Stat(hookBin); err != nil {
			// Try building it from source
			hookBin = filepath.Join("cmd", "tool-approval-hook", "tool-approval-hook")
			if _, err := os.Stat(hookBin); err != nil {
				log.Printf("ℹ️  tool-approval-hook binary not found — Gemini will use default approval mode. Build it with: go build -o tool-approval-hook ./cmd/tool-approval-hook")
				return
			}
		}
	}

	hookBinAbs, errAbs := filepath.Abs(hookBin)
	if errAbs != nil {
		log.Printf("⚠️  Could not resolve absolute path for tool-approval-hook: %v", errAbs)
		return
	}
	serverURL := fmt.Sprintf("http://%s", hubPublicHost(*addr))
	hookCommand := fmt.Sprintf("%s --server %s --agent Gemini --agent-id gemini-cli --mode interactive", hookBinAbs, serverURL)

	// Read existing settings or start fresh
	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if jerr := json.Unmarshal(data, &settings); jerr != nil {
			log.Printf("⚠️  Could not parse Gemini settings.json: %v", jerr)
			settings = nil
		}
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// Check if our hook is already configured
	if hooks, ok := settings["hooks"].(map[string]interface{}); ok {
		if beforeTool, ok := hooks["BeforeTool"].([]interface{}); ok {
			for _, group := range beforeTool {
				if g, ok := group.(map[string]interface{}); ok {
					if hookList, ok := g["hooks"].([]interface{}); ok {
						for _, h := range hookList {
							if hm, ok := h.(map[string]interface{}); ok {
								if name, _ := hm["name"].(string); name == "neural-junkie-approval" {
									// Update the command in case path changed
									hm["command"] = hookCommand
									writeGeminiSettings(settingsPath, settings)
									log.Println("✅ Gemini BeforeTool hook already configured (updated)")
									return
								}
							}
						}
					}
				}
			}
		}
	}

	// Install the hook
	hookEntry := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":        "command",
				"command":     hookCommand,
				"name":        "neural-junkie-approval",
				"timeout":     180000,
				"description": "Routes tool approval through Neural Junkie chat UI",
			},
		},
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	beforeTool, _ := hooks["BeforeTool"].([]interface{})
	beforeTool = append(beforeTool, hookEntry)
	hooks["BeforeTool"] = beforeTool
	settings["hooks"] = hooks

	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		log.Printf("⚠️  Failed to create Gemini settings dir: %v", err)
		return
	}

	writeGeminiSettings(settingsPath, settings)
	log.Printf("✅ Installed Neural Junkie BeforeTool hook in %s", settingsPath)
}

func writeGeminiSettings(path string, settings map[string]interface{}) {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Printf("⚠️  Failed to marshal Gemini settings: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("⚠️  Failed to write Gemini settings to %s: %v", path, err)
	}
}

// ── Ollama management endpoints ─────────────────────────────────────
