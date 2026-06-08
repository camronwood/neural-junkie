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
Write 3-6 bullet points covering:
- The user's current goal (latest unanswered ask)
- Key facts still needed for that goal (only if truly missing from the transcript)
- Open questions only (unresolved)
If the assistant already named specific file paths, loaded workspace files, or proposed edits, record those paths as KNOWN context — do NOT list them under "still needed".
If the user asked to save or export content to a file and a file_change was approved, record the target path and that export is in progress or done — do NOT ask what sections to include unless the user explicitly asked.
Do NOT restate answered questions, copy assistant reply text, or list code the assistant already gave.
Omit topics the user has not mentioned in the last two user messages unless still explicitly open.
Be factual; do not invent details not present in the transcript.
Keep under 250 words. Plain text only.`

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
