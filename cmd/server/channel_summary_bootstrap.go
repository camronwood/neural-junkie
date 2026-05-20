package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
)

const sessionSummarySystemPrompt = `You summarize a short chat channel session for an AI agent's context.
Write 4-8 bullet points covering: user goals, key facts stated, decisions, and open questions.
Be factual; do not invent details not present in the transcript.
Keep under 400 words. Plain text only.`

func initChannelSummaryGenerator(cfg *config.Config, h *hub.Hub) {
	if cfg == nil || h == nil {
		return
	}
	pcfg := cfg.GetProvider("ollama-local")
	if pcfg == nil {
		log.Printf("[Hub] session summary disabled: ollama-local provider not configured")
		return
	}
	util := *pcfg
	util.Model = config.UtilityOllamaModel
	prov, err := ai.ProviderFromConfig(&util)
	if err != nil {
		log.Printf("[Hub] session summary disabled: %v", err)
		return
	}

	gen := func(transcript string) (string, error) {
		transcript = strings.TrimSpace(transcript)
		if transcript == "" {
			return "", fmt.Errorf("empty transcript")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		prompt := sessionSummarySystemPrompt + "\n\n=== TRANSCRIPT ===\n" + transcript
		return prov.GenerateResponse(ctx, prompt, nil)
	}
	h.SetChannelSummaryGenerator(gen, config.UtilityOllamaModel)
	log.Printf("[Hub] session summary generator wired (model=%s)", config.UtilityOllamaModel)
}
