package main

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/embed"
	"github.com/camronwood/neural-junkie/internal/memory"
)

func initConversationMemory() {
	if appConfig == nil {
		return
	}
	store, err := memory.Open("")
	if err != nil {
		log.Printf("[memory] store unavailable: %v", err)
		return
	}
	memory.SetStore(store)
	log.Println("[memory] SQLite store enabled (~/.neural-junkie/memory.db)")

	endpoint := "http://localhost:11434"
	if p := appConfig.GetProvider("ollama-local"); p != nil && strings.TrimSpace(p.Endpoint) != "" {
		endpoint = strings.TrimRight(p.Endpoint, "/")
	}
	model := appConfig.LearningEmbedModel()
	if model == "" {
		model = appConfig.CodebaseEmbedModel()
	}
	if model == "" {
		model = embed.DefaultModel
	}
	memory.SetEmbedClient(embed.NewClient(endpoint, model), model)

	memory.SetEnabledChecker(func() bool {
		return appConfig != nil && appConfig.ConversationMemoryEnabled()
	})
	memory.SetCollabResolver(func(channel string) string {
		if chatHub == nil {
			return ""
		}
		c := chatHub.GetCollaborationManager().GetByChannel(channel)
		if c == nil {
			return ""
		}
		return c.ID
	})

	var workspaceRoots []string
	if chatHub != nil {
		for _, c := range chatHub.GetCollaborationManager().Snapshot() {
			if c == nil {
				continue
			}
			if root := strings.TrimSpace(c.SourceRepoPath); root != "" {
				workspaceRoots = append(workspaceRoots, root)
			}
		}
		if fcm := chatHub.GetFileChangeManager(); fcm != nil {
			if root := strings.TrimSpace(fcm.GetExecutor().GetWorkspaceRoot()); root != "" {
				workspaceRoots = append(workspaceRoots, root)
			}
		}
	}
	memory.ScheduleBackfill(config.CollabAssetsRoot(appConfig), workspaceRoots...)
}
