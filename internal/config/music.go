package config

import (
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/music"
)

// MusicMCPConfig holds ACE-Step generation settings edited in Domain packs → Tools.
type MusicMCPConfig struct {
	ModelVariant   string  `json:"ace_step_model_variant,omitempty"`
	InferenceSteps int     `json:"inference_steps,omitempty"`
	GuidanceScale  float64 `json:"guidance_scale,omitempty"`
	InferMethod    string  `json:"infer_method,omitempty"`
	DefaultSeed    int     `json:"default_seed,omitempty"`
}

func (m MusicMCPConfig) Normalized() MusicMCPConfig {
	out := m
	out.ModelVariant = music.NormalizeModelVariant(out.ModelVariant)
	if out.ModelVariant == "" {
		out.ModelVariant = "sft"
	}
	if out.InferenceSteps <= 0 {
		out.InferenceSteps = music.DefaultInferenceSteps(out.ModelVariant)
	}
	if out.GuidanceScale <= 0 {
		out.GuidanceScale = music.DefaultGuidanceScale(out.ModelVariant)
	}
	out.InferMethod = strings.ToLower(strings.TrimSpace(out.InferMethod))
	if out.InferMethod != "sde" {
		out.InferMethod = "ode"
	}
	return out
}

// MusicMCPSettings returns a copy of music MCP settings (thread-safe).
func (c *Config) MusicMCPSettings() MusicMCPConfig {
	if c == nil {
		return MusicMCPConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.MCP.Music
}

// MusicSidecarSettings returns overlay key/value pairs merged into the music sidecar env.
func (c *Config) MusicSidecarSettings() map[string]string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.musicSidecarSettingsLocked()
}

func (c *Config) musicSidecarSettingsLocked() map[string]string {
	n := c.MCP.Music.Normalized()
	checkpoint, err := music.CheckpointDirForVariant(n.ModelVariant)
	if err != nil {
		checkpoint = ""
	}
	out := map[string]string{
		"ace_step_model_variant":   n.ModelVariant,
		"ace_step_inference_steps": strconv.Itoa(n.InferenceSteps),
		"ace_step_guidance_scale":  strconv.FormatFloat(n.GuidanceScale, 'f', -1, 64),
		"ace_step_infer_method":    n.InferMethod,
		"ace_step_default_seed":    strconv.Itoa(n.DefaultSeed),
	}
	if checkpoint != "" {
		out["ace_step_checkpoint"] = checkpoint
	}
	return out
}
