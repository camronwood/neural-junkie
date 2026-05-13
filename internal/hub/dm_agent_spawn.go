package hub

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// buildExpertAIProvider matches /create-expert provider + model resolution.
func buildExpertAIProvider(providerName, modelOverride string) (ai.AIProvider, error) {
	p := strings.ToLower(strings.TrimSpace(providerName))
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

// ExpertSlugToAgentType maps /create-expert slugs (plus "assistant") to protocol types.
func ExpertSlugToAgentType(expertType string) (protocol.AgentType, error) {
	expertType = strings.ToLower(strings.TrimSpace(expertType))
	validTypes := map[string]protocol.AgentType{
		"rust":       protocol.AgentTypeRust,
		"backend":    protocol.AgentTypeBackend,
		"frontend":   protocol.AgentTypeFrontend,
		"devops":     protocol.AgentTypeDevOps,
		"database":   protocol.AgentTypeDatabase,
		"security":   protocol.AgentTypeSecurity,
		"assistant":  protocol.AgentTypeAssistant,
	}
	agentType, ok := validTypes[expertType]
	if !ok {
		keys := make([]string, 0, len(validTypes))
		for k := range validTypes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return "", fmt.Errorf("unknown expert type %q; valid: %s", expertType, strings.Join(keys, ", "))
	}
	return agentType, nil
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
func (ch *CommandHandler) prepareExpertAgent(agentType protocol.AgentType, name, providerName, modelOverride string) (*agent.Agent, error) {
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

	agentInstance, err := agent.AgentFactory(agentType, name, aiProvider, ch.hub)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	agentInstance.SetCollabClient(ch.hub.NewCollaborationClientAdapter())
	ch.runtimeAgents[agentInstance.Info.ID] = agentInstance

	if err := ch.hub.RegisterAgent(&agentInstance.Info); err != nil {
		delete(ch.runtimeAgents, agentInstance.Info.ID)
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	return agentInstance, nil
}

// SpawnExpertAgentForDM creates an expert (or assistant) agent and a DM with createdBy; agent only joins the DM.
func (ch *CommandHandler) SpawnExpertAgentForDM(_ context.Context, createdBy, expertSlug, displayName, providerName, modelOverride string) (*protocol.Channel, error) {
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return nil, fmt.Errorf("created_by is required")
	}
	agentType, err := ExpertSlugToAgentType(expertSlug)
	if err != nil {
		return nil, err
	}

	name := protocol.NormalizeAgentName(displayName)
	if name == "" {
		return nil, fmt.Errorf("display_name is required")
	}

	agentInstance, err := ch.prepareExpertAgent(agentType, name, providerName, modelOverride)
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
