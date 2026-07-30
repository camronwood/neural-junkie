package main

import (
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
)

// applyRuntimeConfigSideEffects hot-reloads settings that do not require a hub restart.
func applyRuntimeConfigSideEffects(before *config.Config) {
	if appConfig == nil {
		return
	}
	config.SetAppConfig(appConfig)
	if apiRateLimiter != nil {
		sec := appConfig.ResolvedSecurity()
		apiRateLimiter.Reconfigure(sec.RateLimitEnabledOrDefault(), sec.RateReadPerMinute, sec.RateMutatePerMinute)
	}
	if hubSessions != nil {
		sec := appConfig.ResolvedSecurity()
		hubSessions.SetTTLHours(sec.SessionTTLHours)
	}
	hub.InvalidateRuntimeConfigCache()
	if chatHub != nil {
		chatHub.SetSemanticTurnRouter(semanticTurnRouter(appConfig))
	}
	_ = before
}
