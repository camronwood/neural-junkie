package hfhub

import (
	"path/filepath"
	"regexp"
	"strings"
)

const DefaultLoRABaseTag = "qwen2.5-coder:14b"

// BiologyLoRABaseTag is the Ollama base for life-sciences LoRA compose.
const BiologyLoRABaseTag = "llama3:8b"

// BiologyLoRATag is the composed Ollama tag for biology specialists.
const BiologyLoRATag = "nj-biology:8b"

var slugSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

// BiologyLoRATag returns the canonical composed tag for BiologyExpert.
func BiologyLoRATagFn() string {
	return BiologyLoRATag
}

// SpecialistLoRATag returns the canonical composed tag for a dev specialist type.
func SpecialistLoRATag(agentType string) string {
	t := strings.ToLower(strings.TrimSpace(agentType))
	if t == "" {
		return ""
	}
	if t == "biology" {
		return BiologyLoRATag
	}
	return "nj-" + t + ":14b"
}

// RepoLoRATag returns a composed Ollama tag for a repository path.
func RepoLoRATag(repoPath string) string {
	base := filepath.Base(strings.TrimSpace(repoPath))
	slug := strings.ToLower(base)
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = slugSanitizer.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "repo"
	}
	if len(slug) > 32 {
		slug = slug[:32]
	}
	return "nj-repo-" + slug + ":14b"
}

// DefaultAdapterOllamaTag picks a composed tag from catalog metadata or heuristics.
func DefaultAdapterOllamaTag(entry *LibraryModel, filename string) string {
	if entry == nil {
		return DefaultOllamaTag("", filename)
	}
	if tag := strings.TrimSpace(entry.DefaultOllamaTag); tag != "" {
		return tag
	}
	if agentType := strings.TrimSpace(entry.AgentType); agentType != "" {
		return SpecialistLoRATag(agentType)
	}
	repoLower := strings.ToLower(entry.RepoID)
	for _, t := range []string{"biology", "bio", "medmcqa", "biomed", "openbio"} {
		if strings.Contains(repoLower, t) {
			return BiologyLoRATag
		}
	}
	for _, t := range []string{"security", "backend", "frontend", "devops", "architecture", "code-review", "rust", "database"} {
		if strings.Contains(repoLower, t) {
			return SpecialistLoRATag(t)
		}
	}
	return DefaultOllamaTag(entry.RepoID, filename)
}

// IsAdapterEntry reports whether a catalog row is a LoRA adapter (not a full GGUF).
func IsAdapterEntry(entry *LibraryModel) bool {
	if entry == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(entry.Kind))
	if kind == "adapter" {
		return true
	}
	if kind == "full" || kind == "" {
		return false
	}
	return kind == "lora"
}
