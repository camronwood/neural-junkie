package config

import (
	"log"
	"os"
	"strings"
)

// SeedFromEnv copies env-only values into empty config fields on first load (one-time migration).
func (c *Config) SeedFromEnv() {
	if c == nil {
		return
	}
	seeded := false
	seedBool := func(applied *bool, envKey string, set func(bool)) {
		if *applied {
			return
		}
		v := strings.TrimSpace(os.Getenv(envKey))
		if v == "1" || strings.EqualFold(v, "true") {
			set(true)
			*applied = true
			seeded = true
		}
	}
	seedStr := func(dst *string, envKey string) {
		if strings.TrimSpace(*dst) != "" {
			return
		}
		if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
			*dst = v
			seeded = true
		}
	}

	if !c.Security.AuthRequired && os.Getenv("NEURAL_JUNKIE_AUTH_REQUIRED") == "1" {
		c.Security.AuthRequired = true
		seeded = true
	}
	if !c.Security.RelaxedLocal && os.Getenv("NEURAL_JUNKIE_RELAXED_LOCAL") == "1" {
		c.Security.RelaxedLocal = true
		seeded = true
	}
	if !c.Security.ListenAll && !c.Server.ListenAll && os.Getenv("NEURAL_JUNKIE_LISTEN_ALL") == "1" {
		c.Security.ListenAll = true
		c.Server.ListenAll = true
		seeded = true
	}
	seedStr(&c.Security.HubToken, "NEURAL_JUNKIE_HUB_TOKEN")
	seedStr(&c.Security.FullMetadataSecret, "NEURAL_JUNKIE_FULL_METADATA_SECRET")
	if c.Security.SessionTTLHours <= 0 {
		if v, ok := envInt("NEURAL_JUNKIE_SESSION_TTL_HOURS"); ok && v > 0 {
			c.Security.SessionTTLHours = v
			seeded = true
		}
	}
	if os.Getenv("NEURAL_JUNKIE_RATE_LIMIT") == "0" {
		disabled := false
		c.Security.RateLimitEnabled = &disabled
		seeded = true
	}
	if c.Server.CorsAny == false && os.Getenv("NEURAL_JUNKIE_CORS_ANY") == "1" {
		c.Server.CorsAny = true
		seeded = true
	}
	if !c.Debug.Enabled && os.Getenv("NEURAL_JUNKIE_DEBUG") == "1" {
		c.Debug.Enabled = true
		seeded = true
	}
	seedStr(&c.Debug.PprofAddr, "NEURAL_JUNKIE_PPROF_ADDR")
	seedStr(&c.ImageGen.Provider, "NEURAL_JUNKIE_IMAGE_PROVIDER")
	seedStr(&c.ImageGen.Model, "NEURAL_JUNKIE_IMAGE_MODEL")
	seedStr(&c.ImageGen.OllamaModel, "OLLAMA_IMAGE_MODEL")
	seedStr(&c.ImageGen.OpenAIAPIKey, "OPENAI_API_KEY")
	seedStr(&c.SessionSummary.Model, "NJ_SESSION_SUMMARY_MODEL")
	seedStr(&c.Automation.ScenarioRepo, "NEURAL_JUNKIE_SCENARIO_REPO")
	seedStr(&c.Automation.HumanName, "NEURAL_JUNKIE_HUMAN_NAME")
	seedStr(&c.Storage.RepoDir, "NEURAL_JUNKIE_REPO_DIR")
	seedStr(&c.Packs.CatalogURL, "NEURAL_JUNKIE_PACKS_CATALOG_URL")
	seedStr(&c.Routing.CapabilityProfilesPath, "NEURAL_JUNKIE_CAPABILITY_PROFILES")
	if !c.Session.RestoreOnStartup && os.Getenv("NEURAL_JUNKIE_RESTORE_LAST_SESSION") == "1" {
		c.Session.RestoreOnStartup = true
		seeded = true
	}
	var slackDisabled bool
	seedBool(&slackDisabled, "NEURAL_JUNKIE_SLACK_DISABLED", func(v bool) { c.Slack.ForceDisabled = v })
	if c.Features.LegacyFileChangeParse == false && os.Getenv("NEURAL_JUNKIE_LEGACY_FILE_CHANGE_PARSE") == "1" {
		c.Features.LegacyFileChangeParse = true
		seeded = true
	}
	if c.MCPResources.Enabled == false && strings.EqualFold(os.Getenv("ENABLE_MCP_RESOURCES"), "true") {
		c.MCPResources.Enabled = true
		seeded = true
	}
	if seeded {
		log.Printf("[config] seeded settings from environment into config.json (Settings panel is now primary)")
	}
}
