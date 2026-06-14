package hardware

import (
	"fmt"

	"github.com/camronwood/neural-junkie/internal/config"
)

// Tier identifies a hardware class for model recommendations.
type Tier string

const (
	TierMinimal     Tier = "minimal"
	TierLight       Tier = "light"
	TierRecommended Tier = "recommended"
	TierHeavy       Tier = "heavy"
)

// TrackRecommendation is the suggested models for one wizard track at a tier.
type TrackRecommendation struct {
	PrimaryModel string `json:"primary_model"`
	UtilityModel string `json:"utility_model,omitempty"`
	Message      string `json:"message"`
}

// RecommendedStack is a full inference + optional LoRA composition for a RAM tier.
type RecommendedStack struct {
	Tier            Tier     `json:"tier"`
	InferenceModels []string `json:"inference_models"`
	LoRABases       []string `json:"lora_bases,omitempty"`
	ComposedTags    []string `json:"composed_tags,omitempty"`
	DiskEstimateGB  int      `json:"disk_estimate_gb"`
	Notes           string   `json:"notes,omitempty"`
}

// Snapshot is the full hardware assessment returned by the hub API.
type Snapshot struct {
	TotalMemoryBytes  uint64                         `json:"total_memory_bytes"`
	TotalMemoryGB     int                            `json:"total_memory_gb"`
	Tier              Tier                           `json:"tier"`
	Recommendations   map[string]TrackRecommendation `json:"recommendations"`
	RecommendedStacks []RecommendedStack             `json:"recommended_stacks,omitempty"`
}

// TierForMemoryGB maps installed RAM to a tier (whole GB, floor from bytes).
func TierForMemoryGB(gb int) Tier {
	switch {
	case gb < 8:
		return TierMinimal
	case gb < 16:
		return TierLight
	case gb < 32:
		return TierRecommended
	default:
		return TierHeavy
	}
}

// BuildSnapshot probes memory and builds per-track recommendations.
func BuildSnapshot() (Snapshot, error) {
	totalBytes, err := TotalMemoryBytes()
	if err != nil {
		return Snapshot{}, err
	}
	gb := MemoryGB(totalBytes)
	tier := TierForMemoryGB(gb)
	return Snapshot{
		TotalMemoryBytes:  totalBytes,
		TotalMemoryGB:     gb,
		Tier:              tier,
		Recommendations:   RecommendationsForTier(tier, gb),
		RecommendedStacks: RecommendedStacksForTier(tier),
	}, nil
}

// RecommendedStacksForTier returns modular-AI stack compositions per RAM tier.
func RecommendedStacksForTier(tier Tier) []RecommendedStack {
	switch tier {
	case TierMinimal:
		return []RecommendedStack{{
			Tier:            TierMinimal,
			InferenceModels: []string{"qwen3.5:9b"},
			DiskEstimateGB:  10,
			Notes:           "Cloud hybrid recommended for collab and repo-heavy work.",
		}}
	case TierLight:
		return []RecommendedStack{{
			Tier:            TierLight,
			InferenceModels: []string{"qwen3.5:9b"},
			LoRABases:       []string{"llama3:8b"},
			ComposedTags:    []string{"nj-security:14b"},
			DiskEstimateGB:  15,
			Notes:           "Single utility model for chat + tools; one LoRA base optional.",
		}}
	case TierHeavy:
		return []RecommendedStack{{
			Tier:            TierHeavy,
			InferenceModels: []string{"qwen3.5:27b", "qwen3.5:9b", config.BioOllamaChatModel},
			LoRABases:       []string{"llama3.1:8b", "llama3:8b", "llama3.2:3b", "mistral:7b"},
			ComposedTags:    []string{"nj-security:14b", "nj-code-review:14b", "nj-backend:14b", "nj-biology:8b"},
			DiskEstimateGB:  50,
			Notes:           "Full specialist-tuning bootstrap; optional nj-bio:8b and nj-cad:27b HF imports.",
		}}
	default:
		return []RecommendedStack{{
			Tier:            TierRecommended,
			InferenceModels: []string{"qwen3.5:27b", "qwen3.5:9b"},
			LoRABases:       []string{"llama3.1:8b", "llama3:8b", "mistral:7b"},
			ComposedTags:    []string{"nj-security:14b", "nj-code-review:14b", "nj-backend:14b", "nj-biology:8b"},
			DiskEstimateGB:  35,
			Notes:           "Qwen inference tier + Llama/Mistral LoRA compose bases.",
		}}
	}
}

// RecommendationsForTier returns wizard-track model picks for a tier.
func RecommendationsForTier(tier Tier, memoryGB int) map[string]TrackRecommendation {
	devPrimary, devMsg := developerPrimary(tier, memoryGB)
	cadPrimary, cadMsg := cadPrimary(tier, memoryGB)
	bioPrimary, bioMsg := lifeSciencesPrimary(tier, memoryGB)

	return map[string]TrackRecommendation{
		"developer": {
			PrimaryModel: devPrimary,
			UtilityModel: config.UtilityOllamaModel,
			Message:      devMsg,
		},
		"cad": {
			PrimaryModel: cadPrimary,
			UtilityModel: config.CadOllamaToolModel,
			Message:      cadMsg,
		},
		"lifeSciences": {
			PrimaryModel: bioPrimary,
			UtilityModel: config.BioOllamaToolModel,
			Message:      bioMsg,
		},
		"general": {
			PrimaryModel: config.UtilityOllamaModel,
			Message:      fmt.Sprintf("Team chat and productivity work well with %s on your hardware.", config.UtilityOllamaModel),
		},
	}
}

func developerPrimary(tier Tier, memoryGB int) (string, string) {
	switch tier {
	case TierMinimal:
		return "llama3.2:3b", fmt.Sprintf(
			"Your machine has about %d GB RAM. We recommend llama3.2:3b for coding specialists and %s for Assistant. Collaboration and repo agents may struggle locally — consider a cloud API key for harder tasks.",
			memoryGB, config.UtilityOllamaModel,
		)
	case TierLight:
		return "qwen3.5:9b", fmt.Sprintf(
			"Your machine has about %d GB RAM. We recommend qwen3.5:9b instead of the 27B default for coding specialists, plus %s for Assistant.",
			memoryGB, config.UtilityOllamaModel,
		)
	case TierHeavy:
		return config.DevOllamaCodeModel, fmt.Sprintf(
			"Your machine has about %d GB RAM. The full %s + %s stack is a good fit. You can also pull LoRA bases or larger models from the Model library.",
			memoryGB, config.DevOllamaCodeModel, config.UtilityOllamaModel,
		)
	default:
		return config.DevOllamaCodeModel, fmt.Sprintf(
			"Your machine has about %d GB RAM. We recommend %s for coding specialists and %s for Assistant.",
			memoryGB, config.DevOllamaCodeModel, config.UtilityOllamaModel,
		)
	}
}

func cadPrimary(tier Tier, memoryGB int) (string, string) {
	switch tier {
	case TierMinimal:
		return "llama3.2:3b", fmt.Sprintf(
			"Your machine has about %d GB RAM. CADExpert works best with more memory; try llama3.2:3b locally or a cloud key for OpenSCAD authoring.",
			memoryGB,
		)
	case TierLight:
		return config.CadOllamaChatModelLight, fmt.Sprintf(
			"Your machine has about %d GB RAM. We recommend %s for CADExpert instead of the 14B default.",
			memoryGB, config.CadOllamaChatModelLight,
		)
	case TierHeavy:
		return config.CadOllamaChatModel, fmt.Sprintf(
			"Your machine has about %d GB RAM. %s is the balanced default for CADExpert; optional nj-cad:27b is available from the Model library.",
			memoryGB, config.CadOllamaChatModel,
		)
	default:
		return config.CadOllamaChatModel, fmt.Sprintf(
			"Your machine has about %d GB RAM. We recommend %s for CADExpert and %s for tool calls.",
			memoryGB, config.CadOllamaChatModel, config.CadOllamaToolModel,
		)
	}
}

func lifeSciencesPrimary(tier Tier, memoryGB int) (string, string) {
	if tier == TierMinimal {
		return config.BioOllamaChatModel, fmt.Sprintf(
			"Your machine has about %d GB RAM. Local Bio 8B may be tight — consider the cloud Hugging Face path in the wizard, or import nj-bio:8b from the Model library when you have more headroom.",
			memoryGB,
		)
	}
	return config.BioOllamaChatModel, fmt.Sprintf(
		"Your machine has about %d GB RAM. We recommend %s for BiologyExpert and %s for biology MCP tools.",
		memoryGB, config.BioOllamaChatModel, config.BioOllamaToolModel,
	)
}
