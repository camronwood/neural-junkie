package hub

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Preset expert slugs for engineering specialists (used by /create-expert and DM spawn).
var presetExpertTypes = map[string]protocol.AgentType{
	"rust":      protocol.AgentTypeRust,
	"backend":   protocol.AgentTypeBackend,
	"frontend":  protocol.AgentTypeFrontend,
	"devops":    protocol.AgentTypeDevOps,
	"database":  protocol.AgentTypeDatabase,
	"security":  protocol.AgentTypeSecurity,
	"assistant": protocol.AgentTypeAssistant,
}

// ExpertResolveResult describes how to instantiate an expert from a user slug.
type ExpertResolveResult struct {
	AgentType       protocol.AgentType
	IsPreset        bool
	Label           string // human-readable domain label (for messages)
	Expertise       []string
	PersonaMarkdown string
}

var knownExpertProviders = map[string]struct{}{
	"ollama":   {},
	"claude":   {},
	"lmstudio": {},
}

// normalizeCommandArg trims whitespace and leading/trailing punctuation from a slash-command token.
func normalizeCommandArg(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ",;:.")
	return strings.TrimSpace(s)
}

// parseCreateExpertParts re-tokenizes /create-expert arguments: commas become spaces and
// each token is normalized (fixes "guitar," and "guitar, Name" style input).
func parseCreateExpertParts(parts []string) []string {
	if len(parts) < 2 {
		return parts
	}
	rest := strings.Join(parts[1:], " ")
	rest = strings.ReplaceAll(rest, ",", " ")
	tokens := strings.Fields(rest)
	out := make([]string, 0, 1+len(tokens))
	out = append(out, parts[0])
	for _, t := range tokens {
		if t = normalizeCommandArg(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func isKnownExpertProvider(providerName string) bool {
	p := strings.ToLower(normalizeCommandArg(providerName))
	if p == "" {
		return true
	}
	_, ok := knownExpertProviders[p]
	return ok
}

// splitCreateExpertArgs parses tokens after /create-expert.
// Only ollama, claude, and lmstudio are treated as providers; every other token is
// part of the display name (supports multi-word names like "Music Muisc").
func splitCreateExpertArgs(parts []string) (expertSlug, displayName, provider, model string) {
	if len(parts) < 2 {
		return "", "", "", ""
	}
	expertSlug = parts[1]
	if len(parts) < 3 {
		return expertSlug, "", "", ""
	}
	rest := parts[2:]
	providerIdx := -1
	for i, tok := range rest {
		if isKnownExpertProvider(tok) {
			providerIdx = i
			break
		}
	}
	if providerIdx < 0 {
		displayName = strings.Join(rest, " ")
		return expertSlug, displayName, "", ""
	}
	if providerIdx > 0 {
		displayName = strings.Join(rest[:providerIdx], " ")
	}
	provider = strings.ToLower(normalizeCommandArg(rest[providerIdx]))
	if providerIdx+1 < len(rest) {
		model = strings.Join(rest[providerIdx+1:], " ")
	}
	return expertSlug, displayName, provider, model
}

// buildExpertAIProvider matches /create-expert provider + model resolution.
func buildExpertAIProvider(providerName, modelOverride string) (ai.AIProvider, error) {
	p := strings.ToLower(normalizeCommandArg(providerName))
	switch p {
	case "claude":
		claudeProvider, err := ai.NewClaudeProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to create Claude provider: %w", err)
		}
		return claudeProvider, nil
	case "lmstudio":
		if modelOverride != "" {
			return ai.NewLMStudioProviderWithConfig("", modelOverride), nil
		}
		lmProvider, err := ai.NewLMStudioProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to create LM Studio provider: %w", err)
		}
		return lmProvider, nil
	case "ollama", "":
		if modelOverride != "" {
			return ai.NewOllamaProviderWithConfig("", modelOverride), nil
		}
		ollamaProvider, err := ai.NewOllamaProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama provider: %w", err)
		}
		return ollamaProvider, nil
	default:
		return nil, fmt.Errorf("unknown provider %q: use ollama, claude, or lmstudio", providerName)
	}
}

// normalizeExpertSlug trims whitespace, lowercases, and strips trailing punctuation
// (e.g. "guitar," from "/create-expert guitar, Name").
func normalizeExpertSlug(expertType string) string {
	s := strings.ToLower(strings.TrimSpace(expertType))
	s = strings.TrimRightFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == '.' || r == ':'
	})
	return strings.TrimSpace(s)
}

// humanizeExpertSlug turns "legal-advice" into "Legal Advice".
func humanizeExpertSlug(slug string) string {
	slug = strings.NewReplacer("-", " ", "_", " ").Replace(slug)
	words := strings.Fields(slug)
	for i, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func buildCustomPersonaMarkdown(label, extraPersona string) string {
	var b strings.Builder
	b.WriteString("## Persona\n\n")
	b.WriteString("You are **")
	b.WriteString(label)
	b.WriteString("**, a knowledgeable expert in **")
	b.WriteString(label)
	b.WriteString("**.\n\n")
	b.WriteString("Stay in character as this specialist. Give practical, accurate, and approachable advice in this domain.\n")
	if extra := strings.TrimSpace(extraPersona); extra != "" {
		b.WriteString("\n### Additional instructions\n\n")
		b.WriteString(extra)
		b.WriteString("\n")
	}
	return b.String()
}

// ResolveExpert maps a slug to a preset specialist or a custom domain expert.
// Any slug not in the preset list becomes AgentTypeHelper with persona rules.
func ResolveExpert(expertSlug, persona string) (ExpertResolveResult, error) {
	slug := normalizeExpertSlug(expertSlug)
	if slug == "" {
		return ExpertResolveResult{}, fmt.Errorf("expert type is required")
	}
	if agentType, ok := presetExpertTypes[slug]; ok {
		return ExpertResolveResult{
			AgentType: agentType,
			IsPreset:  true,
			Label:     slug,
		}, nil
	}
	label := humanizeExpertSlug(slug)
	return ExpertResolveResult{
		AgentType:       protocol.AgentTypeHelper,
		IsPreset:        false,
		Label:           label,
		Expertise:       []string{label},
		PersonaMarkdown: buildCustomPersonaMarkdown(label, persona),
	}, nil
}

// ExpertSlugToAgentType maps preset /create-expert slugs to protocol types.
// Unknown slugs resolve to AgentTypeHelper (custom experts).
func ExpertSlugToAgentType(expertType string) (protocol.AgentType, error) {
	spec, err := ResolveExpert(expertType, "")
	if err != nil {
		return "", err
	}
	return spec.AgentType, nil
}

func (ch *CommandHandler) expertAgentNameTaken(name string) bool {
	name = protocol.NormalizeAgentName(name)
	for _, existing := range ch.hub.ListAgents() {
		if strings.EqualFold(existing.Name, name) {
			return true
		}
	}
	return false
}

// prepareExpertAgent registers a new expert-style runtime agent (no channel join / start).
func (ch *CommandHandler) prepareExpertAgent(spec ExpertResolveResult, name, providerName, modelOverride string) (*agent.Agent, error) {
	name = protocol.NormalizeAgentName(name)
	if name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if ch.expertAgentNameTaken(name) {
		return nil, fmt.Errorf("agent %q already exists; use a different name or delete the existing agent first", name)
	}

	aiProvider, err := buildExpertAIProvider(providerName, modelOverride)
	if err != nil {
		return nil, err
	}

	var agentInstance *agent.Agent
	if spec.IsPreset {
		agentInstance, err = agent.AgentFactory(spec.AgentType, name, aiProvider, ch.hub)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent: %w", err)
		}
	} else {
		agentInstance = agent.NewCustomExpertAgent(name, spec.Expertise, aiProvider, ch.hub)
		if spec.PersonaMarkdown != "" {
			agentInstance.Info.CustomRulesMarkdown = spec.PersonaMarkdown
		}
	}
	agentInstance.SetCollabClient(ch.hub.NewCollaborationClientAdapter())
	ch.runtimeAgents[agentInstance.Info.ID] = agentInstance

	if err := ch.hub.RegisterAgent(&agentInstance.Info); err != nil {
		delete(ch.runtimeAgents, agentInstance.Info.ID)
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	if !spec.IsPreset && strings.TrimSpace(spec.PersonaMarkdown) != "" {
		if err := ch.hub.SetAgentCustomRulesMarkdown(agentInstance.Info.ID, spec.PersonaMarkdown); err != nil {
			log.Printf("Failed to persist custom expert persona for %s: %v", name, err)
		}
	}

	return agentInstance, nil
}

// SpawnExpertAgentForDM creates an expert (or assistant) agent and a DM with createdBy; agent only joins the DM.
func (ch *CommandHandler) SpawnExpertAgentForDM(_ context.Context, createdBy, expertSlug, displayName, providerName, modelOverride, persona string) (*protocol.Channel, error) {
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return nil, fmt.Errorf("created_by is required")
	}
	spec, err := ResolveExpert(expertSlug, persona)
	if err != nil {
		return nil, err
	}

	name := protocol.NormalizeAgentName(displayName)
	if name == "" {
		return nil, fmt.Errorf("display_name is required")
	}

	agentInstance, err := ch.prepareExpertAgent(spec, name, providerName, modelOverride)
	if err != nil {
		return nil, err
	}
	agentInstance.DisableChannelDiscovery = true

	dmCh, err := ch.hub.CreateDMChannel(createdBy, agentInstance.Info.ID)
	if err != nil {
		_ = ch.hub.UnregisterAgent(agentInstance.Info.ID)
		delete(ch.runtimeAgents, agentInstance.Info.ID)
		return nil, fmt.Errorf("failed to create DM channel: %w", err)
	}

	// Agent listen loop must not use the HTTP request context: net/http cancels
	// r.Context() when ServeHTTP returns, which would tear down subscriptions.
	if err := agentInstance.Start(context.Background(), dmCh.Name); err != nil {
		_ = ch.hub.UnregisterAgent(agentInstance.Info.ID)
		delete(ch.runtimeAgents, agentInstance.Info.ID)
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}

	return dmCh, nil
}

// SpawnCLIAgentForDM creates a CLI-backed agent and a DM; agent only joins the DM.
func (ch *CommandHandler) SpawnCLIAgentForDM(_ context.Context, createdBy, cliType, displayName, workDir string) (*protocol.Channel, error) {
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return nil, fmt.Errorf("created_by is required")
	}
	cliType = strings.ToLower(strings.TrimSpace(cliType))
	cfg, ok := agent.GetCLIAgentConfig(cliType)
	if !ok {
		return nil, fmt.Errorf("unknown CLI agent type %q", cliType)
	}

	if workDir == "" && cfg.WorkDirEnv != "" {
		workDir = os.Getenv(cfg.WorkDirEnv)
	}
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			workDir = "."
		}
	}

	opts := []ai.CLIAgentOption{
		ai.WithBaseArgs(cfg.BaseArgs),
		ai.WithModel(cfg.ModelName),
	}
	provider := ai.NewCLIAgentProvider(cfg.Command, workDir, cfg.ProviderName, opts...)
	for _, envKey := range cfg.EnvVars {
		if val := os.Getenv(envKey); val != "" {
			provider.Env[envKey] = val
		}
	}
	if !provider.IsCLIInstalled() {
		return nil, fmt.Errorf("CLI binary %q not found on PATH: %s", cfg.Command, cfg.InstallHint)
	}

	name := protocol.NormalizeAgentName(displayName)
	if name == "" {
		name = cfg.DefaultName
	}
	name = protocol.NormalizeAgentName(name)
	if ch.expertAgentNameTaken(name) {
		return nil, fmt.Errorf("agent %q already exists; use a different name or delete the existing agent first", name)
	}

	agentInstance := agent.NewCLIAgentFromConfig(cfg, name, provider, ch.hub)
	agentInstance.SetCollabClient(ch.hub.NewCollaborationClientAdapter())
	agentInstance.DisableChannelDiscovery = true
	if cfg.ApprovalMode != "" {
		agentInstance.Info.ApprovalMode = cfg.ApprovalMode
	}

	if err := ch.hub.RegisterAgent(&agentInstance.Info); err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	dmCh, err := ch.hub.CreateDMChannel(createdBy, agentInstance.Info.ID)
	if err != nil {
		_ = ch.hub.UnregisterAgent(agentInstance.Info.ID)
		return nil, fmt.Errorf("failed to create DM channel: %w", err)
	}

	ch.cliAgents[agentInstance.Info.ID] = agentInstance
	ch.runtimeAgents[agentInstance.Info.ID] = agentInstance
	agent.SaveCLIAgent(cliType, name, workDir)

	joinMsg := protocol.NewMessage(
		protocol.MessageTypeAgentJoin,
		dmCh.Name,
		agentInstance.Info,
		cfg.JoinMessage,
	)
	if err := ch.hub.SendMessage(joinMsg); err != nil {
		log.Printf("Failed to send CLI agent join message: %v", err)
	}

	// Same as SpawnExpertAgentForDM: never bind agent lifetime to r.Context().
	// Start synchronously so the DM subscriber exists before the API returns.
	if err := agentInstance.Start(context.Background(), dmCh.Name); err != nil {
		_ = ch.hub.UnregisterAgent(agentInstance.Info.ID)
		delete(ch.cliAgents, agentInstance.Info.ID)
		delete(ch.runtimeAgents, agentInstance.Info.ID)
		log.Printf("Failed to start CLI agent %s: %v", name, err)
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}

	return dmCh, nil
}
