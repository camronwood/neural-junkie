package config

import (
	"github.com/camronwood/neural-junkie/internal/packs"
)

// ResolvedLoRAPolicy returns merged lora_policy from the enabled specialist-tuning pack.
func (c *Config) ResolvedLoRAPolicy() packs.LoRAPolicy {
	if c == nil || !c.IsPackEnabled(PackSpecialistTuning) {
		return packs.LoRAPolicy{}.Resolved()
	}
	m, err := c.InstalledPackManifestByID(PackSpecialistTuning)
	if err != nil || m == nil {
		return packs.LoRAPolicy{}.Resolved()
	}
	return m.LoRAPolicy.Resolved()
}

// LoRAEvalProbesForAgentType returns pack eval probe path for a bootstrap adapter agent_type.
func (c *Config) LoRAEvalProbesForAgentType(agentType string) string {
	if c == nil {
		return ""
	}
	m, err := c.InstalledPackManifestByID(PackSpecialistTuning)
	if err != nil || m == nil {
		return ""
	}
	for _, la := range m.LoRAAdapters {
		if la.AgentType == agentType && la.EvalProbes != "" {
			return la.EvalProbes
		}
	}
	return ""
}
