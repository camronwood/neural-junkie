package config

import "strings"

// SpecialistComposeEntry is per-agent-type compose overrides (from packs or config).
type SpecialistComposeEntry struct {
	ChatModel       string   `json:"chat_model,omitempty"`
	ToolModel       string   `json:"tool_model,omitempty"`
	LoRATag         string   `json:"lora_tag,omitempty"`
	ConsultTriggers []string `json:"consult_triggers,omitempty"`
}

// ChatModelForAgent resolves the chat model for an agent type.
func (c *Config) ChatModelForAgent(agentType string, agentModel string) string {
	if c == nil {
		return strings.TrimSpace(agentModel)
	}
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	if m := strings.TrimSpace(agentModel); m != "" {
		return m
	}
	// User MCP/delegation overrides win over pack compose defaults.
	switch agentType {
	case "biology", "genomics", "structural-biology", "cheminformatics":
		if c.IsPackEnabled(PackLifeSciences) {
			if m := c.biologyChatModelUnlocked(); m != "" {
				return m
			}
		}
	case "cad":
		if c.IsPackEnabled(PackCAD) {
			if m := c.cadChatModelUnlocked(); m != "" {
				return m
			}
		}
	}
	if entry, ok := c.SpecialistCompose[agentType]; ok {
		if m := strings.TrimSpace(entry.ChatModel); m != "" {
			return m
		}
	}
	switch agentType {
	case "biology", "genomics", "structural-biology", "cheminformatics":
		if c.IsPackEnabled(PackLifeSciences) {
			return BioOllamaChatModel
		}
	case "cad":
		if c.IsPackEnabled(PackCAD) {
			return c.CadMCPSettings().ChatModelOrDefault()
		}
	case "music":
		if c.IsPackEnabled(PackMusicCreation) {
			if entry, ok := c.SpecialistCompose[agentType]; ok {
				if m := strings.TrimSpace(entry.ChatModel); m != "" {
					return m
				}
			}
			return "qwen2.5:7b"
		}
	}
	return ""
}

func (c *Config) biologyChatModelUnlocked() string {
	if c == nil {
		return BioOllamaChatModel
	}
	if m := strings.TrimSpace(c.MCP.Biology.ChatModel); m != "" {
		return m
	}
	if m := strings.TrimSpace(c.Delegation.BiologyChatModel); m != "" {
		return m
	}
	return BioOllamaChatModel
}

func (c *Config) biologyToolModelUnlocked() string {
	if c == nil {
		return BioOllamaToolModel
	}
	if m := strings.TrimSpace(c.MCP.Biology.ToolModel); m != "" {
		return m
	}
	if m := strings.TrimSpace(c.Delegation.BiologyToolModel); m != "" {
		return m
	}
	return BioOllamaToolModel
}

func (c *Config) cadChatModelUnlocked() string {
	if c == nil {
		return ""
	}
	if m := strings.TrimSpace(c.MCP.CAD.ChatModel); m != "" {
		return m
	}
	return c.CadMCPSettings().ChatModelOrDefault()
}

// ToolModelForAgent resolves the tool-loop model for an agent type.
func (c *Config) ToolModelForAgent(agentType string) string {
	if c == nil {
		return UtilityOllamaModel
	}
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	if entry, ok := c.SpecialistCompose[agentType]; ok {
		if m := strings.TrimSpace(entry.ToolModel); m != "" {
			return m
		}
	}
	switch agentType {
	case "biology", "genomics", "structural-biology", "cheminformatics":
		return c.BiologyToolModelOrDefault()
	case "cad":
		return c.CadMCPSettings().ToolModelOrDefault()
	}
	return UtilityOllamaModel
}

// LoRATagForAgent returns a configured LoRA tag for an agent type when set.
func (c *Config) LoRATagForAgent(agentType string) string {
	if c == nil {
		return ""
	}
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	if entry, ok := c.SpecialistCompose[agentType]; ok {
		return strings.TrimSpace(entry.LoRATag)
	}
	return ""
}
