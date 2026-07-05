package ai

import (
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

// ImageGenConfig describes the active image generation backend from environment.
type ImageGenConfig struct {
	Provider     string `json:"provider"` // ollama, openai, none
	Model        string `json:"model"`
	Endpoint     string `json:"endpoint,omitempty"`
	OpenAIKeySet bool   `json:"openai_key_set"`
}

// ImageGenStatus reports whether hub image generation is ready.
type ImageGenStatus struct {
	Ready         bool   `json:"ready"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	Endpoint      string `json:"endpoint,omitempty"`
	Disabled      bool   `json:"disabled"`
	OllamaRunning bool   `json:"ollama_running"`
	ModelPulled   bool   `json:"model_pulled"`
	OpenAIKeySet  bool   `json:"openai_key_set"`
	PullCommand   string `json:"pull_command,omitempty"`
}

// ImageGenConfigFromEnv resolves provider, model, and endpoint from config with env overrides.
func ImageGenConfigFromEnv() ImageGenConfig {
	return ImageGenConfigFromSettings(config.AppConfig().ResolvedImageGen())
}

// ImageGenConfigFromSettings builds runtime image gen config from stored settings.
func ImageGenConfigFromSettings(st config.ImageGenSettings) ImageGenConfig {
	provider := strings.ToLower(strings.TrimSpace(st.Provider))
	if provider == "" {
		provider = "ollama"
	}
	switch provider {
	case "openai":
		return openAIImageConfig(st)
	case "none", "disabled", "off":
		return ImageGenConfig{Provider: "none"}
	default:
		return ollamaImageConfig(st)
	}
}

func openAIImageConfig(st config.ImageGenSettings) ImageGenConfig {
	cfg := ImageGenConfig{Provider: "openai"}
	cfg.Model = strings.TrimSpace(st.Model)
	if cfg.Model == "" {
		cfg.Model = "dall-e-3"
	}
	endpoint := strings.TrimSpace(st.OpenAIBaseURL)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	}
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	cfg.Endpoint = strings.TrimRight(endpoint, "/")
	cfg.OpenAIKeySet = strings.TrimSpace(st.OpenAIAPIKey) != ""
	return cfg
}

func ollamaImageConfig(st config.ImageGenSettings) ImageGenConfig {
	cfg := ImageGenConfig{Provider: "ollama"}
	endpoint := strings.TrimSpace(os.Getenv("OLLAMA_ENDPOINT"))
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if ep := config.AppConfig().FirstOllamaEndpoint(); ep != "" {
		endpoint = ep
	}
	cfg.Endpoint = strings.TrimRight(endpoint, "/")
	cfg.Model = strings.TrimSpace(st.OllamaModel)
	if cfg.Model == "" {
		cfg.Model = strings.TrimSpace(st.Model)
	}
	if cfg.Model == "" {
		cfg.Model = DefaultOllamaImageModel
	}
	return cfg
}

func imageProviderFromEnv() string {
	st := config.AppConfig().ResolvedImageGen()
	switch strings.ToLower(strings.TrimSpace(st.Provider)) {
	case "openai":
		return "openai"
	case "none", "disabled", "off":
		return "none"
	default:
		return "ollama"
	}
}

// BuildImageGenStatus combines config with live Ollama checks.
func BuildImageGenStatus(cfg ImageGenConfig, ollamaRunning, modelPulled bool) ImageGenStatus {
	st := ImageGenStatus{
		Provider:      cfg.Provider,
		Model:         cfg.Model,
		Endpoint:      cfg.Endpoint,
		OpenAIKeySet:  cfg.OpenAIKeySet,
		OllamaRunning: ollamaRunning,
		ModelPulled:   modelPulled,
	}
	switch cfg.Provider {
	case "none":
		st.Disabled = true
	case "openai":
		st.Ready = cfg.OpenAIKeySet
	default:
		st.PullCommand = "ollama pull " + cfg.Model
		st.Ready = ollamaRunning && modelPulled
	}
	return st
}
