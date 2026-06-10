package main

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func rebindRuntimeAgentsToRestoredDMs() {
	channels := chatHub.ListChannels()
	agents := chatHub.ListAgents()
	if len(channels) == 0 || len(agents) == 0 {
		return
	}

	agentsByName := make(map[string]*protocol.AgentInfo, len(agents))
	agentsBySlug := make(map[string]*protocol.AgentInfo, len(agents))
	for _, a := range agents {
		if a == nil {
			continue
		}
		agentsByName[strings.ToLower(strings.TrimSpace(a.Name))] = a
		agentsBySlug[slugifyName(a.Name)] = a
	}

	for _, ch := range channels {
		if ch == nil || ch.Type != protocol.ChannelTypeDM {
			continue
		}
		targetName := extractDMAgentName(ch)
		if targetName == "" {
			continue
		}

		target, ok := agentsByName[strings.ToLower(targetName)]
		if !ok {
			target, ok = agentsBySlug[slugifyName(targetName)]
		}
		if !ok || target == nil {
			log.Printf("ℹ️  DM rebind skipped for %s (agent not found: %s)", ch.Name, targetName)
			continue
		}

		if err := chatHub.JoinChannel(target.ID, ch.Name); err != nil {
			log.Printf("⚠️  DM rebind failed for %s -> %s: %v", target.Name, ch.Name, err)
			continue
		}
		if chHandler, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok && chHandler != nil {
			if err := chHandler.EnsureAgentSubscribedToChannel(context.Background(), target.ID, ch.Name); err != nil {
				log.Printf("⚠️  DM rebind subscribe failed for %s -> %s: %v", target.Name, ch.Name, err)
			}
		}
		log.Printf("✅ DM rebind: %s -> %s", target.Name, ch.Name)
	}
}

func extractDMAgentName(ch *protocol.Channel) string {
	if ch == nil {
		return ""
	}

	desc := strings.TrimSpace(ch.Description)
	lowerDesc := strings.ToLower(desc)
	const prefix = "direct message with "
	if strings.HasPrefix(lowerDesc, prefix) && len(desc) > len(prefix) {
		return strings.TrimSpace(desc[len(prefix):])
	}

	// Fallback: dm-<user>-<agent-slug> where <agent-slug> may contain hyphens
	// (e.g. dm-camron-cursor-buddy → "cursor-buddy", not "buddy").
	parts := strings.SplitN(ch.Name, "-", 3)
	if len(parts) == 3 && strings.EqualFold(parts[0], "dm") {
		return parts[2]
	}
	return ""
}

func slugifyName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
