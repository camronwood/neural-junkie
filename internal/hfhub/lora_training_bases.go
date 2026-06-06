package hfhub

import (
	"fmt"
	"strings"
)

// LoRATrainingBase is an Ollama base tag that supports safetensors LoRA compose after training.
type LoRATrainingBase struct {
	OllamaTag    string `json:"ollama_tag"`
	HFModel      string `json:"hf_model"`
	Label        string `json:"label"`
	Description  string `json:"description"`
	CodeFocused  bool   `json:"code_focused"`
	Recommended  bool   `json:"recommended,omitempty"`
	SizeHint     string `json:"size_hint,omitempty"`
}

// DefaultLoRATrainingCodeBase is the suggested code-training base (LoRA-compatible; local-friendly size).
const DefaultLoRATrainingCodeBase = "llama3.1:8b"

var loRATrainingBases = []LoRATrainingBase{
	{
		OllamaTag:   "llama3.1:8b",
		HFModel:     "meta-llama/Meta-Llama-3.1-8B-Instruct",
		Label:       "Llama 3.1 8B Instruct",
		Description: "Recommended default for local LoRA training — strong general + code, Ollama compose works.",
		CodeFocused: true,
		Recommended: true,
		SizeHint:    "~4.9 GB",
	},
	{
		OllamaTag:   "codestral:latest",
		HFModel:     "mistralai/Codestral-22B-v0.1",
		Label:       "Codestral 22B",
		Description: "Best code-focused alternative to Qwen among LoRA-compatible bases (Mistral arch). Large — needs more VRAM to train.",
		CodeFocused: true,
		SizeHint:    "~12 GB",
	},
	{
		OllamaTag:   "llama3:8b",
		HFModel:     "meta-llama/Meta-Llama-3-8B-Instruct",
		Label:       "Llama 3 8B Instruct",
		Description: "Life-sciences and lighter workloads; biology pack default.",
		CodeFocused: false,
		SizeHint:    "~4.7 GB",
	},
	{
		OllamaTag:   "mistral:7b",
		HFModel:     "mistralai/Mistral-7B-Instruct-v0.3",
		Label:       "Mistral 7B Instruct",
		Description: "Lighter LoRA-compatible base when VRAM is limited.",
		CodeFocused: false,
		SizeHint:    "~4.4 GB",
	},
}

// LoRATrainingBases returns Ollama bases valid for train → compose workflows.
func LoRATrainingBases() []LoRATrainingBase {
	out := make([]LoRATrainingBase, len(loRATrainingBases))
	copy(out, loRATrainingBases)
	return out
}

// DefaultLoRATrainingBaseForAgent picks a suggested training base from agent type.
func DefaultLoRATrainingBaseForAgent(agentType string) string {
	t := strings.ToLower(strings.TrimSpace(agentType))
	if t == "biology" {
		return BiologyLoRABaseTag
	}
	return DefaultLoRATrainingCodeBase
}

// ValidateLoRATrainingBase rejects bases Ollama cannot compose (e.g. Qwen safetensors LoRA).
func ValidateLoRATrainingBase(baseTag string) error {
	baseTag = strings.TrimSpace(baseTag)
	if baseTag == "" {
		return fmt.Errorf("base_ollama_tag is required")
	}
	if !OllamaSafetensorLoRABaseSupported(baseTag) {
		return fmt.Errorf(
			"base %q cannot be used for LoRA training yet: Ollama only composes safetensors LoRA for Llama, Mistral, and Gemma bases (not Qwen). Use one of: %s",
			baseTag,
			strings.Join(loRATrainingBaseTags(), ", "),
		)
	}
	if MapLoRABaseToHF(baseTag) == "" {
		return fmt.Errorf("base %q is not a configured LoRA training base; use one of: %s", baseTag, strings.Join(loRATrainingBaseTags(), ", "))
	}
	return nil
}

// MapLoRABaseToHF maps an Ollama tag to the Hugging Face model id for Unsloth training.
func MapLoRABaseToHF(ollamaTag string) string {
	tag := normalizeOllamaTag(ollamaTag)
	for _, b := range loRATrainingBases {
		if normalizeOllamaTag(b.OllamaTag) == tag {
			return b.HFModel
		}
	}
	// Allow aliases for bases Ollama supports even if not in curated training list.
	switch tag {
	case "llama3.1:latest", "llama3.1":
		return "meta-llama/Meta-Llama-3.1-8B-Instruct"
	case "codestral":
		return "mistralai/Codestral-22B-v0.1"
	case "gemma2:9b", "gemma2:2b", "gemma2:latest":
		return "" // supported by Ollama compose but not curated for NJ training yet
	}
	return ""
}

func loRATrainingBaseTags() []string {
	tags := make([]string, 0, len(loRATrainingBases))
	for _, b := range loRATrainingBases {
		tags = append(tags, b.OllamaTag)
	}
	return tags
}

func normalizeOllamaTag(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	tag = strings.TrimPrefix(tag, "ollama/")
	if i := strings.Index(tag, "@"); i >= 0 {
		tag = tag[:i]
	}
	return tag
}
