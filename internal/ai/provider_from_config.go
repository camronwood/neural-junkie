package ai

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

// ProviderFromConfig builds an AIProvider from a persisted provider row.
func ProviderFromConfig(pcfg *config.ProviderConfig) (AIProvider, error) {
	if pcfg == nil {
		return nil, fmt.Errorf("nil provider config")
	}
	switch pcfg.Type {
	case "ollama":
		endpoint := pcfg.Endpoint
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		model := pcfg.Model
		if model == "" {
			model = "llama3.1"
		}
		return NewOllamaProviderWithConfig(endpoint, model), nil

	case "anthropic":
		if pcfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic provider %q has no API key", pcfg.ID)
		}
		return NewClaudeProviderWithConfig(pcfg.APIKey, false, "", pcfg.Model), nil

	case "openai-compatible":
		endpoint := pcfg.Endpoint
		if endpoint == "" {
			return nil, fmt.Errorf("openai-compatible provider %q has no endpoint", pcfg.ID)
		}
		model := pcfg.Model
		return NewOpenAICompatProvider(endpoint, pcfg.APIKey, model, pcfg.Headers), nil

	case "cursor-cli":
		workDir := pcfg.WorkDir
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		var opts []CLIAgentOption
		if pcfg.TimeoutSeconds > 0 {
			opts = append(opts, WithTimeout(time.Duration(pcfg.TimeoutSeconds)*time.Second))
		}
		return NewCursorCLIProvider(workDir, pcfg.APIKey, opts...), nil

	case "huggingface":
		token := ResolveHFToken(pcfg.APIKey)
		if token == "" {
			return nil, fmt.Errorf("huggingface provider %q has no API key (set api_key or HF_TOKEN)", pcfg.ID)
		}
		endpoint := pcfg.Endpoint
		if endpoint == "" {
			endpoint = DefaultHuggingFaceRouterEndpoint
		}
		model := strings.TrimSpace(pcfg.Model)
		if model == "" {
			return nil, fmt.Errorf("huggingface provider %q has no model (Hub repo id, e.g. Qwen/Qwen2.5-Coder-7B-Instruct)", pcfg.ID)
		}
		return NewHuggingFaceProvider(endpoint, token, model), nil

	case "gemini-cli":
		workDir := pcfg.WorkDir
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		var opts []CLIAgentOption
		if pcfg.TimeoutSeconds > 0 {
			opts = append(opts, WithTimeout(time.Duration(pcfg.TimeoutSeconds)*time.Second))
		}
		model := strings.TrimSpace(pcfg.Model)
		if model == "" {
			model = strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
		}
		if model == "" {
			model = "gemini-2.5-flash"
		}
		opts = append(opts, WithEnv("GEMINI_MODEL", model), WithModel(model))
		return NewGeminiCLIProvider(workDir, opts...), nil

	default:
		return nil, fmt.Errorf("unknown provider type %q", pcfg.Type)
	}
}
