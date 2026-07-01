package profile

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hfhub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ResolvedProfile is the runtime view of chat/tool/compose models.
type ResolvedProfile struct {
	ChatModel   string `json:"chat_model"`
	ToolModel   string `json:"tool_model"`
	ComposeBase string `json:"compose_base"`
	ComposedTag string `json:"composed_tag,omitempty"`
	UseComposed bool   `json:"use_composed_for_chat"`
}

// ResolveForAgent resolves model profile for an agent.
func ResolveForAgent(cfg *config.Config, info *protocol.AgentInfo, agentCfg *config.AgentConfig) ResolvedProfile {
	out := ResolvedProfile{
		ComposeBase: hfhub.DefaultLoRABaseTag,
	}
	if cfg != nil {
		out.ToolModel = cfg.ToolModelForAgent(string(info.Type))
		if out.ToolModel == "" {
			out.ToolModel = config.UtilityOllamaModel
		}
	}
	if agentCfg != nil && agentCfg.ModelProfile != nil {
		mp := agentCfg.ModelProfile
		if m := strings.TrimSpace(mp.InferenceModel); m != "" {
			out.ChatModel = m
		}
		if m := strings.TrimSpace(mp.LoRAComposeBase); m != "" {
			out.ComposeBase = m
		}
		if m := strings.TrimSpace(mp.ComposedTag); m != "" {
			out.ComposedTag = m
		}
		out.UseComposed = mp.UseComposedForChat
	}
	if out.ChatModel == "" {
		if agentCfg != nil && strings.TrimSpace(agentCfg.Model) != "" {
			m := strings.TrimSpace(agentCfg.Model)
			if strings.HasPrefix(m, "nj-") {
				out.ComposedTag = m
				out.UseComposed = true
			} else {
				out.ChatModel = m
			}
		}
	}
	if out.ChatModel == "" && info != nil {
		if m := strings.TrimSpace(info.Model); m != "" {
			if strings.HasPrefix(m, "nj-") {
				out.ComposedTag = m
				out.UseComposed = true
			} else {
				out.ChatModel = m
			}
		}
	}
	if out.ChatModel == "" && cfg != nil {
		out.ChatModel = cfg.ChatModelForAgent(string(info.Type), "")
	}
	if out.ComposedTag == "" && cfg != nil {
		if tag := cfg.LoRATagForAgent(string(info.Type)); tag != "" {
			out.ComposedTag = tag
		}
	}
	if out.UseComposed && out.ComposedTag != "" {
		out.ChatModel = out.ComposedTag
	}
	return out
}

// EffectiveChatModel returns the model tag for chat turns.
func EffectiveChatModel(p ResolvedProfile) string {
	if p.UseComposed && strings.TrimSpace(p.ComposedTag) != "" {
		return strings.TrimSpace(p.ComposedTag)
	}
	return strings.TrimSpace(p.ChatModel)
}
