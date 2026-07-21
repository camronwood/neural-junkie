package main

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func restorePersistedDMAgents() {
	if chatHub == nil {
		return
	}
	rawHandler := chatHub.GetCommandHandler()
	ch, ok := rawHandler.(*hub.CommandHandler)
	if !ok || ch == nil {
		return
	}

	restorePersistedCLIAgents(ch)
	restorePersistedExpertAgents(ch)
}

func agentActiveByName(name string) bool {
	for _, a := range chatHub.ListAgents() {
		if a != nil && strings.EqualFold(a.Name, name) {
			return true
		}
	}
	return false
}

func restorePersistedCLIAgents(ch *hub.CommandHandler) {
	storage, err := agent.NewCLIAgentStorage()
	if err != nil {
		log.Printf("⚠️  Failed to open CLI agent storage: %v", err)
		return
	}
	records, err := storage.ListRecords()
	if err != nil {
		log.Printf("⚠️  Failed to load CLI agent records: %v", err)
		return
	}
	for _, record := range records {
		if strings.TrimSpace(record.Name) == "" || strings.TrimSpace(record.Type) == "" {
			continue
		}
		if agentActiveByName(record.Name) {
			continue
		}
		createdBy := strings.TrimSpace(record.CreatedBy)
		if createdBy == "" {
			createdBy = inferCreatedByFromDM(record.Name)
		}
		if createdBy == "" {
			log.Printf("ℹ️  Skipping CLI agent restore for %q (no created_by)", record.Name)
			continue
		}
		if _, err := ch.SpawnCLIAgentForDM(context.Background(), createdBy, record.Type, record.Name, record.WorkDir); err != nil {
			log.Printf("⚠️  Failed to restore CLI agent %q: %v", record.Name, err)
			continue
		}
		log.Printf("✅ Restored persisted CLI agent %q", record.Name)
	}
}

func restorePersistedExpertAgents(ch *hub.CommandHandler) {
	storage, err := agent.NewExpertAgentStorage()
	if err != nil {
		log.Printf("⚠️  Failed to open expert agent storage: %v", err)
		return
	}
	records, err := storage.List()
	if err != nil {
		log.Printf("⚠️  Failed to load expert agent records: %v", err)
		return
	}
	for _, record := range records {
		if strings.TrimSpace(record.Name) == "" || strings.TrimSpace(record.ExpertSlug) == "" {
			continue
		}
		if agentActiveByName(record.Name) {
			continue
		}
		createdBy := strings.TrimSpace(record.CreatedBy)
		if createdBy == "" {
			createdBy = inferCreatedByFromDM(record.Name)
		}
		if createdBy == "" {
			log.Printf("ℹ️  Skipping expert restore for %q (no created_by)", record.Name)
			continue
		}
		if _, err := ch.SpawnExpertAgentForDM(
			context.Background(),
			createdBy,
			record.ExpertSlug,
			record.Name,
			record.ProviderID,
			record.ProviderName,
			record.Model,
			record.Persona,
			record.CapabilityAllow,
			record.CapabilityDeny,
		); err != nil {
			log.Printf("⚠️  Failed to restore expert agent %q: %v", record.Name, err)
			continue
		}
		log.Printf("✅ Restored persisted expert agent %q", record.Name)
	}
}

func inferCreatedByFromDM(agentName string) string {
	slug := strings.ToLower(strings.TrimSpace(agentName))
	for _, ch := range chatHub.ListChannels() {
		if ch == nil || ch.Type != protocol.ChannelTypeDM {
			continue
		}
		if !strings.Contains(strings.ToLower(ch.Name), slug) {
			continue
		}
		if u := strings.TrimSpace(ch.CreatedBy); u != "" {
			return u
		}
		parts := strings.SplitN(ch.Name, "-", 3)
		if len(parts) >= 2 && strings.EqualFold(parts[0], "dm") {
			return parts[1]
		}
	}
	return ""
}
