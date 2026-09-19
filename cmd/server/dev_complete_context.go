package main

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

const (
	devCompleteMaxContextChars  = 24000
	devCompleteMaxNeighborChars = 6000
	devCompleteMaxPrefixChars   = 8000
	devCompleteMaxSuffixChars   = 4000
	devCompletePrimaryPredict   = 128
	devCompleteAlternatePredict = 96
	devCompletePrimaryTemp      = 0.2
	devCompleteAlternateTemp    = 0.55
)

// NeighborSnippet is an optional open-tab / recent-edit locality snippet from the client.
type NeighborSnippet struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Source  string `json:"source,omitempty"` // "open_tab" | "recent_edit" | "codeindex" | etc.
}

// DevCompleteRequest is the shared body for /api/dev/complete and /api/dev/complete/stream.
type DevCompleteRequest struct {
	Prefix           string            `json:"prefix"`
	Suffix           string            `json:"suffix"`
	Language         string            `json:"language"`
	Path             string            `json:"path"`
	Context          string            `json:"context"`
	Model            string            `json:"model"`
	NeighborSnippets []NeighborSnippet `json:"neighbor_snippets"`
	// NRequests asks for multiple generations (1–2). Default 2 for non-stream.
	NRequests int `json:"n,omitempty"`
}

func resolveDevCompleteModel(reqModel string) string {
	model := strings.TrimSpace(reqModel)
	if model != "" {
		return model
	}
	if appConfig != nil {
		return config.DevOllamaCodeModel
	}
	return config.DevOllamaCodeModel
}

func resolveOllamaEndpoint() string {
	endpoint := "http://localhost:11434"
	if appConfig != nil {
		for _, p := range appConfig.AI.Providers {
			if p.Type == "ollama" && strings.TrimSpace(p.Endpoint) != "" {
				return strings.TrimRight(p.Endpoint, "/")
			}
		}
	}
	return endpoint
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "\n…"
}

func truncatePrefix(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return "…\n" + string(r[len(r)-max:])
}

func truncateSuffix(s string, max int) string {
	return truncateRunes(s, max)
}

func formatNeighborSnippets(neighbors []NeighborSnippet, budget int) string {
	if len(neighbors) == 0 || budget <= 0 {
		return ""
	}
	var b strings.Builder
	remaining := budget
	for i, n := range neighbors {
		if remaining <= 0 {
			break
		}
		path := strings.TrimSpace(n.Path)
		if path == "" {
			path = "neighbor"
		}
		src := strings.TrimSpace(n.Source)
		header := "// neighbor: " + path
		if src != "" {
			header += " (" + src + ")"
		}
		header += "\n"
		bodyBudget := remaining - len(header) - 8
		if bodyBudget < 80 {
			break
		}
		body := truncateRunes(strings.TrimSpace(n.Content), bodyBudget)
		if body == "" {
			continue
		}
		chunk := header + body + "\n\n"
		if len(chunk) > remaining {
			chunk = truncateRunes(chunk, remaining)
		}
		b.WriteString(chunk)
		remaining -= len(chunk)
		_ = i
	}
	return strings.TrimSpace(b.String())
}

// AssembleDevCompletePrompt builds the Ollama prompt from prefix/suffix/context/neighbors.
func AssembleDevCompletePrompt(req DevCompleteRequest) string {
	prefix := truncatePrefix(req.Prefix, devCompleteMaxPrefixChars)
	suffix := truncateSuffix(req.Suffix, devCompleteMaxSuffixChars)
	ctx := strings.TrimSpace(req.Context)
	if ctx != "" {
		ctx = truncateRunes(ctx, devCompleteMaxContextChars)
	}
	neighbors := formatNeighborSnippets(req.NeighborSnippets, devCompleteMaxNeighborChars)

	lang := strings.TrimSpace(req.Language)
	if lang == "" {
		lang = "text"
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		path = "file"
	}

	var b strings.Builder
	if neighbors != "" {
		b.WriteString("Related open files / recent edits:\n")
		b.WriteString(neighbors)
		b.WriteString("\n")
	}
	if ctx != "" {
		b.WriteString("File: ")
		b.WriteString(path)
		b.WriteString("\n```")
		b.WriteString(lang)
		b.WriteString("\n")
		b.WriteString(ctx)
		b.WriteString("\n```\n\n")
	}
	// Prefer FIM markers when we have suffix; otherwise fall back to prefix-only complete-at-cursor.
	if suffix != "" {
		b.WriteString("<|fim_prefix|>")
		b.WriteString(prefix)
		b.WriteString("<|fim_suffix|>")
		b.WriteString(suffix)
		b.WriteString("<|fim_middle|>")
	} else if ctx != "" || neighbors != "" {
		b.WriteString("Complete at cursor:\n")
		b.WriteString(prefix)
	} else {
		b.WriteString(prefix)
	}
	return b.String()
}

func clampDevCompleteN(n int, defaultN int) int {
	if n <= 0 {
		return defaultN
	}
	if n > 2 {
		return 2
	}
	return n
}

func ollamaGenerateBody(model, prompt string, stream bool, numPredict int, temperature float64) map[string]interface{} {
	return map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": stream,
		"options": map[string]interface{}{
			"num_predict": numPredict,
			"temperature": temperature,
		},
	}
}

func dedupeCompletions(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
