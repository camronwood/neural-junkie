package config

// WizardTrack identifies first-run setup paths in the desktop wizard.
type WizardTrack string

const (
	WizardTrackDeveloper    WizardTrack = "developer"
	WizardTrackLifeSciences WizardTrack = "lifeSciences"
	WizardTrackGeneral      WizardTrack = "general"
)

// WizardProfile holds model and agent defaults for a setup track.
type WizardProfile struct {
	Track            WizardTrack
	OllamaModel      string
	ModelsToEnsure   []string
	HFHostedModel    string // Hub repo id for huggingface provider
	CloudAnthropic   string
	DefaultAgents    []AgentConfig
}

const (
	// BioOllamaChatModel is the recommended local OpenBio chat model (Ollama hub; Llama 3 template).
	BioOllamaChatModel = "koesn/llama3-openbiollm-8b:latest"
	// BioOllamaToolModel runs MCP biology tools when the chat model lacks native tool calling.
	BioOllamaToolModel = "qwen2.5:7b"
	// UtilityOllamaModel is the hub utility tier for session summaries and similar background tasks.
	UtilityOllamaModel = "qwen2.5:7b"
	// BioOllamaTag is the canonical Ollama model name for OpenBioLLM (import via HF GGUF).
	BioOllamaTag = "nj-bio:8b"
	// BioHFRepo is the Hugging Face instruct model for hosted inference.
	BioHFRepo = "aaditya/Llama3-OpenBioLLM-8B"
	// BioHFGGUFRepo is the GGUF catalog repo for local import.
	BioHFGGUFRepo = "aaditya/OpenBioLLM-Llama3-8B-GGUF"
)

// WizardProfileFor returns defaults for developer, life sciences, or general setup.
func WizardProfileFor(track WizardTrack) WizardProfile {
	switch track {
	case WizardTrackLifeSciences:
		return WizardProfile{
			Track:          WizardTrackLifeSciences,
			OllamaModel:    BioOllamaChatModel,
			ModelsToEnsure: []string{BioOllamaChatModel, BioOllamaToolModel},
			HFHostedModel:  BioHFRepo,
			CloudAnthropic: "claude-3-5-sonnet-20241022",
			DefaultAgents: []AgentConfig{
				{Type: "biology", Name: "BiologyExpert", Enabled: true},
				{Type: "assistant", Name: "Assistant", Enabled: true},
			},
		}
	case WizardTrackGeneral:
		return WizardProfile{
			Track:          WizardTrackGeneral,
			OllamaModel:    UtilityOllamaModel,
			ModelsToEnsure: []string{UtilityOllamaModel},
			HFHostedModel:  "",
			CloudAnthropic: "claude-3-5-sonnet-20241022",
			DefaultAgents: []AgentConfig{
				{Type: "assistant", Name: "Assistant", Enabled: true},
			},
		}
	default:
		return WizardProfile{
			Track:          WizardTrackDeveloper,
			OllamaModel:    DevOllamaCodeModel,
			ModelsToEnsure: []string{DevOllamaCodeModel, UtilityOllamaModel},
			HFHostedModel:  "",
			CloudAnthropic: "claude-3-5-sonnet-20241022",
			DefaultAgents: []AgentConfig{
				{Type: "assistant", Name: "Assistant", Enabled: true},
			},
		}
	}
}

// ApplyWizardProfile mutates cfg for a completed wizard track (Ollama local path).
func (c *Config) ApplyWizardProfile(track WizardTrack, ollamaLocal bool) {
	p := WizardProfileFor(track)
	c.Agents = append([]AgentConfig(nil), p.DefaultAgents...)
	if c.Packs.Enabled == nil {
		c.Packs = DefaultPacksConfig()
	}
	switch track {
	case WizardTrackLifeSciences:
		c.Packs.Enabled[PackLifeSciences] = true
		c.Packs.Enabled[PackSoftwareDevelopment] = false
	case WizardTrackDeveloper:
		c.Packs.Enabled[PackLifeSciences] = false
		c.Packs.Enabled[PackSoftwareDevelopment] = true
	default:
		c.Packs.Enabled[PackLifeSciences] = false
		c.Packs.Enabled[PackSoftwareDevelopment] = false
	}
	c.SyncAgentsFromPacks()
	c.Ollama.ModelsToEnsure = append([]string(nil), c.Ollama.ModelsToEnsure...)

	if ollamaLocal {
		c.AI.DefaultProviderID = "ollama-local"
		for i := range c.AI.Providers {
			if c.AI.Providers[i].ID == "ollama-local" {
				c.AI.Providers[i].Model = p.OllamaModel
				return
			}
		}
		c.AI.Providers = []ProviderConfig{{
			ID:       "ollama-local",
			Type:     "ollama",
			Name:     "Local Ollama",
			Endpoint: "http://localhost:11434",
			Model:    p.OllamaModel,
		}}
	}
}
