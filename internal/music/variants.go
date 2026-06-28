package music

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModelVariant describes an ACE-Step checkpoint family.
type ModelVariant struct {
	ID             string
	CheckpointDir  string
	HFRepo         string
	DefaultSteps   int
	DefaultGuidance float64
}

var modelVariants = map[string]ModelVariant{
	"sft": {
		ID: "sft", CheckpointDir: "acestep-v15-sft", HFRepo: "ACE-Step/acestep-v15-sft",
		DefaultSteps: 50, DefaultGuidance: 7.0,
	},
	"turbo": {
		ID: "turbo", CheckpointDir: "acestep-v15-turbo", HFRepo: "ACE-Step/Ace-Step1.5",
		DefaultSteps: 8, DefaultGuidance: 1.0,
	},
	"xl-sft": {
		ID: "xl-sft", CheckpointDir: "acestep-v15-xl-sft", HFRepo: "ACE-Step/acestep-v15-xl-sft",
		DefaultSteps: 50, DefaultGuidance: 7.0,
	},
	"xl-turbo": {
		ID: "xl-turbo", CheckpointDir: "acestep-v15-xl-turbo", HFRepo: "ACE-Step/acestep-v15-xl-turbo",
		DefaultSteps: 8, DefaultGuidance: 1.0,
	},
}

// NormalizeModelVariant returns a supported variant id or sft.
func NormalizeModelVariant(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if _, ok := modelVariants[raw]; ok {
		return raw
	}
	return "sft"
}

// ModelVariantInfo returns metadata for a variant id.
func ModelVariantInfo(variantID string) ModelVariant {
	v, ok := modelVariants[NormalizeModelVariant(variantID)]
	if !ok {
		return modelVariants["sft"]
	}
	return v
}

// ListModelVariants returns supported variant ids in display order.
func ListModelVariants() []string {
	return []string{"sft", "turbo", "xl-sft", "xl-turbo"}
}

// CheckpointDirForVariant returns the absolute checkpoint path for a variant.
func CheckpointDirForVariant(variantID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	v := ModelVariantInfo(variantID)
	return filepath.Join(home, ".neural-junkie", "music", "checkpoints", v.CheckpointDir), nil
}

// CheckpointNameForVariant returns the ACE-Step config_path directory name.
func CheckpointNameForVariant(variantID string) string {
	return ModelVariantInfo(variantID).CheckpointDir
}

// HFRepoForVariant returns the HuggingFace repo to download for a variant.
func HFRepoForVariant(variantID string) string {
	return ModelVariantInfo(variantID).HFRepo
}

// DefaultInferenceSteps returns recommended steps for a variant.
func DefaultInferenceSteps(variantID string) int {
	return ModelVariantInfo(variantID).DefaultSteps
}

// DefaultGuidanceScale returns recommended guidance for a variant.
func DefaultGuidanceScale(variantID string) float64 {
	return ModelVariantInfo(variantID).DefaultGuidance
}

// ResolveVariantFromCheckpoint infers variant from a checkpoint path basename.
func ResolveVariantFromCheckpoint(checkpoint string) string {
	base := strings.TrimSpace(filepath.Base(checkpoint))
	for id, v := range modelVariants {
		if v.CheckpointDir == base {
			return id
		}
	}
	return "sft"
}

// VariantLabel returns a short human label.
func VariantLabel(variantID string) string {
	switch NormalizeModelVariant(variantID) {
	case "turbo":
		return "Turbo (fast, 8 steps)"
	case "xl-sft":
		return "XL SFT (higher quality, ~50 steps)"
	case "xl-turbo":
		return "XL Turbo (fast XL, 8 steps)"
	default:
		return "SFT (balanced, ~50 steps)"
	}
}

// ValidateModelVariant returns an error for unknown variants.
func ValidateModelVariant(raw string) error {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return nil
	}
	if _, ok := modelVariants[raw]; ok {
		return nil
	}
	return fmt.Errorf("unsupported ace_step_model_variant %q (use sft, turbo, xl-sft, xl-turbo)", raw)
}
