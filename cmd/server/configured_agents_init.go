package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/contextcompress"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func syncMCPFromConfig() {
	if appConfig != nil {
		mcp.SetAppConfig(appConfig)
		ai.SetHubRuntimeOptions(appConfig.Performance, appConfig.Ollama)
		initContextCompressStore()
	}
}

func initContextCompressStore() {
	if appConfig == nil {
		return
	}
	p := appConfig.Performance
	maxEntries := p.ContextCacheMaxEntries
	if maxEntries <= 0 {
		maxEntries = 500
	}
	ttl := p.ContextCacheTTLMinutes
	if ttl <= 0 {
		ttl = 60
	}
	contextcompress.SetDefaultStore(contextcompress.NewStore(maxEntries, ttl, contextcompress.DefaultCacheDir()))
}

// reconcileConfiguredSpecialists stops hub specialists whose type is disabled in config
// (e.g. after turning off a domain pack).

func reconcileConfiguredSpecialists() {
	if appConfig == nil || chatHub == nil {
		return
	}
	specTypes := config.ConfigurableSpecialistTypes()
	var cmdHandler *hub.CommandHandler
	if h := chatHub.GetCommandHandler(); h != nil {
		cmdHandler, _ = h.(*hub.CommandHandler)
	}
	for _, ag := range chatHub.ListAgents() {
		t := strings.ToLower(string(ag.Type))
		if !specTypes[t] {
			continue
		}
		if appConfig.SpecialistShouldBeRunning(t) {
			continue
		}
		// Leave channels first so clients get agent_leave and channel membership updates.
		for _, channel := range chatHub.ListChannels() {
			for _, member := range channel.Agents {
				if member.ID == ag.ID {
					if err := chatHub.LeaveChannel(ag.ID, channel.Name); err != nil {
						log.Printf("Pack reconcile: leave %s from %s: %v", ag.Name, channel.Name, err)
					}
					break
				}
			}
		}
		if cmdHandler != nil {
			cmdHandler.StopAndUnregisterRuntimeAgent(ag.ID)
		}
		if err := chatHub.UnregisterAgent(ag.ID); err != nil {
			log.Printf("Pack reconcile: unregister %s: %v", ag.Name, err)
		} else {
			log.Printf("Pack reconcile: stopped specialist %s (type=%s, pack off or disabled)", ag.Name, t)
		}
	}
}

// initializeConfiguredAgents starts specialist agents defined in the config
// file. Each enabled agent runs in-process using the hub's push-based
// message delivery (same as moderator/assistant).

func initializeConfiguredAgents() {
	if appConfig == nil {
		return
	}

	enabled := appConfig.EnabledAgents()
	if len(enabled) == 0 {
		log.Println("ℹ️  No specialist agents configured")
		return
	}

	log.Printf("🤖 Starting %d configured specialist agent(s)...", len(enabled))

	for _, acfg := range enabled {
		pcfg := appConfig.ProviderForAgent(acfg)
		if pcfg == nil {
			log.Printf("⚠️  No provider found for agent %s (provider_id=%q, default=%q) — skipping",
				acfg.Name, acfg.ProviderID, appConfig.AI.DefaultProviderID)
			continue
		}

		aiProvider, err := globalProviderCache.GetForAgent(appConfig, acfg)
		if err != nil {
			log.Printf("⚠️  Failed to build provider for agent %s: %v — skipping", acfg.Name, err)
			continue
		}

		agentType := protocol.AgentType(acfg.Type)
		if builtinType, ok := packs.ParseBuiltinImplementation(acfg.Implementation); ok {
			agentType = protocol.AgentType(builtinType)
		}
		agentObj, err := agent.AgentFactory(agentType, acfg.Name, aiProvider, chatHub)
		if err != nil {
			log.Printf("❌ Failed to create agent %s (type=%s): %v", acfg.Name, acfg.Type, err)
			continue
		}
		agentObj.SetCollabClient(chatHub.NewCollaborationClientAdapter())

		if err := chatHub.RegisterAgent(&agentObj.Info); err != nil {
			log.Printf("❌ Failed to register agent %s: %v", acfg.Name, err)
			continue
		}
		if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
			if ch, ok := commandHandler.(*hub.CommandHandler); ok {
				ch.RegisterRuntimeAgent(agentObj)
				wireAgentWorkspaceBackend(agentObj)
			}
		}

		greeting := fmt.Sprintf("👋 %s online! Ready to help with %s questions.", acfg.Name, acfg.Type)
		if err := chatHub.JoinChannel(agentObj.Info.ID, "general", greeting); err != nil {
			log.Printf("❌ Failed to join agent %s to general channel: %v", acfg.Name, err)
			continue
		}

		ctx := context.Background()
		go func(name string) {
			if err := agentObj.Start(ctx, "general"); err != nil {
				log.Printf("❌ Failed to start agent %s: %v", name, err)
			}
		}(acfg.Name)

		log.Printf("✅ Agent %s started (type=%s, provider=%s, model=%s)",
			acfg.Name, acfg.Type, pcfg.Name, aiProvider.GetModel())
	}
}

// ── Health, Settings, Provider, Agent Config endpoints ───────────────

// handleDebugHubMemory returns hub message counts and Go runtime memory stats.
// Enabled only when NEURAL_JUNKIE_DEBUG=1 (localhost tooling; do not expose publicly).
