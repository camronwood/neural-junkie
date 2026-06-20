package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const geminiAPIKeyEnv = "GEMINI_API_KEY"

type geminiUserSettings struct {
	Security struct {
		Auth struct {
			SelectedType string `json:"selectedType"`
		} `json:"auth"`
	} `json:"security"`
}

// appendGeminiCLIEnv adds Gemini CLI subprocess environment for headless hub runs.
// When GEMINI_API_KEY is configured and ~/.gemini still selects oauth-personal,
// GEMINI_CLI_HOME is pointed at an isolated config dir so stale OAuth does not block API-key auth.
func appendGeminiCLIEnv(cmdEnv []string, p *CLIAgentProvider) []string {
	model := p.EffectiveCLIModel()
	if model == "" {
		model = strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	}
	if model != "" {
		cmdEnv = append(cmdEnv, fmt.Sprintf("GEMINI_MODEL=%s", model))
	}

	apiKey := geminiCLIAPIKey(p)
	if apiKey == "" {
		return cmdEnv
	}
	cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", geminiAPIKeyEnv, apiKey))
	if shouldUseGeminiHeadlessHome() {
		if home := resolveGeminiCLIHeadlessHome(); home != "" {
			cmdEnv = append(cmdEnv, fmt.Sprintf("GEMINI_CLI_HOME=%s", home))
		}
	}
	return cmdEnv
}

func geminiCLIAPIKey(p *CLIAgentProvider) string {
	if p != nil {
		if key := strings.TrimSpace(p.Env[geminiAPIKeyEnv]); key != "" {
			return key
		}
	}
	return strings.TrimSpace(os.Getenv(geminiAPIKeyEnv))
}

func shouldUseGeminiHeadlessHome() bool {
	if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_GEMINI_CLI_HOME")) != "" {
		return true
	}
	selected := readGeminiUserAuthType()
	if selected == "" || selected == "oauth-personal" || selected == "login_with_google" {
		return true
	}
	return false
}

func readGeminiUserAuthType() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".gemini", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var settings geminiUserSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return ""
	}
	return strings.TrimSpace(settings.Security.Auth.SelectedType)
}

func resolveGeminiCLIHeadlessHome() string {
	if v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_GEMINI_CLI_HOME")); v != "" {
		if hasGeminiHeadlessSettings(v) {
			return v
		}
	}
	candidates := []string{
		"scripts/gemini-headless-home",
		filepath.Join("..", "scripts", "gemini-headless-home"),
		filepath.Join("..", "..", "scripts", "gemini-headless-home"),
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "scripts", "gemini-headless-home"))
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if hasGeminiHeadlessSettings(abs) {
			return abs
		}
	}
	return ""
}

func hasGeminiHeadlessSettings(home string) bool {
	fi, err := os.Stat(filepath.Join(home, ".gemini", "settings.json"))
	return err == nil && fi != nil && !fi.IsDir()
}
