package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func routingOptions(cfg *config.Config) routing.Options {
	if cfg == nil {
		return routing.DefaultOptions()
	}
	rc := cfg.Routing.Normalized()
	opts := routing.Options{
		Classifier:      rc.Classifier,
		ClassifierModel: rc.ClassifierModel,
		RulesFallback:   rc.RulesFallback,
		MinConfidence:   rc.MinConfidence,
	}
	if strings.EqualFold(rc.Classifier, "llm") {
		if p := routingClassifierProvider(cfg, rc.ClassifierModel); p != nil {
			opts.LLMClassifier = routing.NewLLMClassifierFromProvider(p)
		}
	}
	return opts
}

func routingClassifierProvider(cfg *config.Config, model string) ai.AIProvider {
	if cfg == nil {
		return nil
	}
	pcfg := cfg.GetProvider("ollama-local")
	if pcfg == nil {
		for i := range cfg.AI.Providers {
			if strings.EqualFold(cfg.AI.Providers[i].Type, "ollama") {
				pcfg = &cfg.AI.Providers[i]
				break
			}
		}
	}
	if pcfg == nil {
		return nil
	}
	util := *pcfg
	util.Model = strings.TrimSpace(model)
	if util.Model == "" {
		util.Model = config.UtilityOllamaModel
	}
	prov, err := ai.ProviderFromConfig(&util)
	if err != nil {
		log.Printf("[routing] classifier provider unavailable: %v", err)
		return nil
	}
	return prov
}

func semanticTurnRouter(cfg *config.Config) *intent.Router {
	if cfg == nil || cfg.Routing.SemanticRoutingLegacyRollback {
		return nil
	}
	rc := cfg.Routing.Normalized()
	if rc.SemanticTextGatesDisabled {
		intent.SetTextGatesDisabled(true)
	}
	provider := routingClassifierProvider(cfg, rc.SemanticClassifierModel)
	router := intent.NewRouter(nil, rc.MinConfidence)
	if provider != nil {
		router = intent.NewRouter(intent.NewLLMClassifier(semanticAIGenerator{provider: provider}), rc.MinConfidence)
	}
	router.Timeout = time.Duration(rc.SemanticClassifierTimeoutMS) * time.Millisecond
	return router
}

// prepareDispatchEnabled reports whether /api/turn/prepare + dispatch are active.
func prepareDispatchEnabled(cfg *config.Config) bool {
	if cfg == nil {
		return true
	}
	if cfg.Routing.SemanticRoutingLegacyRollback {
		return false
	}
	if cfg.Routing.SemanticPrepareDispatchEnabled == nil {
		return true
	}
	return *cfg.Routing.SemanticPrepareDispatchEnabled
}

type semanticAIGenerator struct {
	provider ai.AIProvider
}

func (g semanticAIGenerator) Generate(ctx context.Context, systemPrompt, userPayload string) (string, error) {
	prompt := systemPrompt + ai.SystemPromptSeparator + userPayload
	if provider, ok := g.provider.(ai.StructuredOutputProvider); ok {
		result, err := provider.GenerateStructuredResponse(ctx, ai.StructuredOutputRequest{
			Prompt:     prompt,
			SchemaName: "semantic_intent",
			JSONSchema: intent.SemanticIntentSchemaJSON(),
		})
		if err == nil && strings.TrimSpace(result.Content) != "" {
			return result.Content, nil
		}
	}
	return g.provider.GenerateResponse(ctx, prompt, nil)
}

func (g semanticAIGenerator) Model() string {
	return g.provider.GetModel()
}

func classifyTask(ctx context.Context, cfg *config.Config, text, agentType, agentModel string, hasImages bool, loraTags map[string]struct{}) routing.RoutingDecision {
	in := routing.Input{
		Text:          text,
		AgentType:     agentType,
		AgentModel:    agentModel,
		HasUserImages: hasImages,
		InstalledTags: loraTags,
	}
	return routing.Classify(ctx, in, routingOptions(cfg))
}
