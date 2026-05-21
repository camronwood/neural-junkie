package config

// DelegationConfig controls cross-agent consultation inside the hub.
type DelegationConfig struct {
	Enabled            bool   `json:"enabled"`
	MaxConsultsPerTurn int    `json:"max_consults_per_turn"`
	MaxDepth           int    `json:"max_depth"`
	MinRelevanceScore  int    `json:"min_relevance_score"`
	Visibility         string `json:"visibility"` // silent, metadata, visible
	Classifier         string `json:"classifier"` // rules (v1), llm (future)
	OrchestratorModel  string `json:"orchestrator_model,omitempty"`
	BiologyChatModel   string `json:"biology_chat_model,omitempty"`
	BiologyToolModel   string `json:"biology_tool_model,omitempty"`
}

// DefaultDelegationConfig returns safe v1 defaults (feature off).
func DefaultDelegationConfig() DelegationConfig {
	return DelegationConfig{
		Enabled:            false,
		MaxConsultsPerTurn: 2,
		MaxDepth:           1,
		MinRelevanceScore:  2,
		Visibility:         "silent",
		Classifier:         "rules",
		BiologyChatModel:   BioOllamaChatModel,
		BiologyToolModel:   BioOllamaToolModel,
	}
}

// Normalized returns delegation settings with defaults filled in.
func (d DelegationConfig) Normalized() DelegationConfig {
	out := d
	if out.MaxConsultsPerTurn <= 0 {
		out.MaxConsultsPerTurn = 2
	}
	if out.MaxDepth <= 0 {
		out.MaxDepth = 1
	}
	if out.MinRelevanceScore <= 0 {
		out.MinRelevanceScore = 2
	}
	if out.Visibility == "" {
		out.Visibility = "silent"
	}
	if out.Classifier == "" {
		out.Classifier = "rules"
	}
	if out.BiologyChatModel == "" {
		out.BiologyChatModel = BioOllamaChatModel
	}
	if out.BiologyToolModel == "" {
		out.BiologyToolModel = BioOllamaToolModel
	}
	return out
}
