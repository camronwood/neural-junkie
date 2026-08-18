package agent

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const contextProfileMetadata = "context_profile"

const (
	constrainedNumCtxCeiling     = 8192
	constrainedMaxPromptBytesCap = 24 * 1024
	constrainedMaxPromptBytesMin = 8 * 1024
	constrainedDefaultNumCtx     = 8192
	constrainedSizeBCeiling      = 9
)

var ollamaSizeTagRE = regexp.MustCompile(`(?i)(?:^|[:/\-])(\d+(?:\.\d+)?)b(?:[-_]|$)`)

// TurnContextProfile is the per-turn compiler decision for prompt size, tools, and retrieval.
type TurnContextProfile struct {
	Constrained      bool `json:"constrained"`
	MaxPromptBytes   int  `json:"max_prompt_bytes"`
	NativeToolsOnly  bool `json:"native_tools_only"`
	EagerCodebase    bool `json:"eager_codebase"`
	EagerRepoConsult bool `json:"eager_repo_consult"`
	EmptyRetry       bool `json:"empty_retry"`
}

func isIDEComposerTurn(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(protocol.ComposerModeFromMessage(msg))) {
	case "ask", "plan", "agent", "edit":
		return true
	default:
		return false
	}
}

func isOllamaProviderName(provider string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(provider)), "ollama")
}

func ollamaParameterSizeB(model string) (float64, bool) {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return 0, false
	}
	loc := ollamaSizeTagRE.FindStringSubmatch(m)
	if len(loc) < 2 {
		return 0, false
	}
	n, err := strconv.ParseFloat(loc[1], 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func constrainedMaxPromptBytes(numCtx int) int {
	if numCtx <= 0 {
		numCtx = constrainedDefaultNumCtx
	}
	// ~55% of the window at ~3 bytes/token, leaving room for generation and tool JSON.
	n := (numCtx * 3 * 55) / 100
	if n < constrainedMaxPromptBytesMin {
		n = constrainedMaxPromptBytesMin
	}
	if n > constrainedMaxPromptBytesCap {
		n = constrainedMaxPromptBytesCap
	}
	return n
}

func resolveConstrainedLocal(provider, model string, numCtx int) bool {
	if !isOllamaProviderName(provider) {
		return false
	}
	if ai.OllamaModelPrefersCompactPrompt(model) || ai.OllamaSmallChatModel(model) {
		return true
	}
	if numCtx > 0 && numCtx <= constrainedNumCtxCeiling {
		return true
	}
	if size, ok := ollamaParameterSizeB(model); ok && size <= constrainedSizeBCeiling {
		return true
	}
	return false
}

func resolveNativeToolsOnly(a *Agent, constrained bool, model string) bool {
	if a != nil {
		if tp, ok := a.AI.(ai.ToolCapableProvider); ok {
			return tp.SupportsTools()
		}
	}
	if constrained && !ai.OllamaModelPrefersCompactPrompt(model) {
		return true
	}
	return false
}

func resolveTurnContextProfile(a *Agent, msg *protocol.Message, numCtx int) TurnContextProfile {
	provider, model := "", ""
	if a != nil {
		provider = a.Info.AIProvider
		model = a.Info.AIModel
	}
	constrained := resolveConstrainedLocal(provider, model, numCtx)
	explicitCodebase := msg != nil && codebaseMentionRE.MatchString(msg.Content)
	citedPath := msg != nil && len(mentionedSourcePaths(msg.Content)) > 0
	eager := !constrained || explicitCodebase || citedPath
	p := TurnContextProfile{
		Constrained:      constrained,
		NativeToolsOnly:  resolveNativeToolsOnly(a, constrained, model),
		EagerCodebase:    eager,
		EagerRepoConsult: !constrained,
		EmptyRetry:       constrained,
	}
	if constrained {
		p.MaxPromptBytes = constrainedMaxPromptBytes(numCtx)
	}
	return p
}

func stampTurnContextProfile(msg *protocol.Message, p TurnContextProfile) {
	if msg == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata[contextProfileMetadata] = map[string]interface{}{
		"constrained":        p.Constrained,
		"max_prompt_bytes":   p.MaxPromptBytes,
		"native_tools_only":  p.NativeToolsOnly,
		"eager_codebase":     p.EagerCodebase,
		"eager_repo_consult": p.EagerRepoConsult,
		"empty_retry":        p.EmptyRetry,
	}
}

func turnContextProfileFromMetadata(msg *protocol.Message) (TurnContextProfile, bool) {
	if msg == nil || msg.Metadata == nil {
		return TurnContextProfile{}, false
	}
	raw, ok := msg.Metadata[contextProfileMetadata]
	if !ok {
		return TurnContextProfile{}, false
	}
	switch m := raw.(type) {
	case TurnContextProfile:
		return m, true
	case *TurnContextProfile:
		if m == nil {
			return TurnContextProfile{}, false
		}
		return *m, true
	case map[string]interface{}:
		p := TurnContextProfile{}
		p.Constrained, _ = m["constrained"].(bool)
		p.NativeToolsOnly, _ = m["native_tools_only"].(bool)
		p.EagerCodebase, _ = m["eager_codebase"].(bool)
		p.EagerRepoConsult, _ = m["eager_repo_consult"].(bool)
		p.EmptyRetry, _ = m["empty_retry"].(bool)
		switch n := m["max_prompt_bytes"].(type) {
		case int:
			p.MaxPromptBytes = n
		case int64:
			p.MaxPromptBytes = int(n)
		case float64:
			p.MaxPromptBytes = int(n)
		}
		return p, true
	default:
		return TurnContextProfile{}, false
	}
}

func (a *Agent) turnContextProfile(msg *protocol.Message) TurnContextProfile {
	p := resolveTurnContextProfile(a, msg, ai.OllamaNumCtx())
	stampTurnContextProfile(msg, p)
	return p
}

func (a *Agent) constrainedIDETurn(msg *protocol.Message) bool {
	return isIDEComposerTurn(msg) && a.turnContextProfile(msg).Constrained
}

func skipUnscopedCodebaseDump(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if msg.IdeEditorModeIsPlan() {
		return true
	}
	if !isIDEComposerTurn(msg) {
		return false
	}
	p, ok := turnContextProfileFromMetadata(msg)
	return ok && p.Constrained
}
