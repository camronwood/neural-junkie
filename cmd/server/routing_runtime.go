package main

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
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
