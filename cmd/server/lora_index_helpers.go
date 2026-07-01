package main

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/lora/export"
	"github.com/camronwood/neural-junkie/internal/lora/train"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

func enrichLoraTrainRequest(body *train.StartRequest) export.Request {
	req := export.Request{
		Source:            body.Source,
		SourceID:          body.SourceID,
		ThreadID:          body.ThreadID,
		AgentName:         body.AgentName,
		RowIDs:            body.RowIDs,
		ApprovedTasksOnly: body.ApprovedTasksOnly,
		ExtraRows:         export.AppendExtraRows(body.LearningRows, body.ExtraRows),
	}
	if body.Incremental && loraAdapterRegistry != nil {
		agentID := strings.TrimSpace(body.AgentID)
		if e, ok := loraAdapterRegistry.ActiveForAgent(agentID); ok {
			req.SinceJobExportedAt = e.ExportedAt
		} else if id := strings.TrimSpace(body.PriorAdapterID); id != "" {
			if e, ok := loraAdapterRegistry.Get(id); ok {
				req.SinceJobExportedAt = e.ExportedAt
			}
		}
	}
	return req
}

func attachLearningRows(body *train.StartRequest) {
	if !body.IncludeLearnings || learningStore == nil || !personalLearningActive() {
		return
	}
	agentID := strings.TrimSpace(body.AgentID)
	if agentID == "" {
		return
	}
	body.LearningRows = export.ExportLearningsRows(learningStore.List(agentID))
}

func loadRepoIndexForAgent(agentID string) (*repo.RepositoryIndex, error) {
	info, err := chatHub.GetAgent(agentID)
	if err != nil {
		return nil, err
	}
	if info.Type != protocol.AgentTypeRepo && strings.TrimSpace(info.RepositoryPath) == "" {
		return nil, fmt.Errorf("agent is not a repo expert")
	}
	if raw := chatHub.GetCommandHandler(); raw != nil {
		if ch, ok := raw.(*hub.CommandHandler); ok {
			if ra, ok := ch.RepoAgentByID(agentID); ok {
				if idx := ra.RepositoryIndex(); idx != nil {
					return idx, nil
				}
			}
		}
	}
	repoPath := strings.TrimSpace(info.RepositoryPath)
	if repoPath == "" {
		return nil, fmt.Errorf("repo expert has no repository path")
	}
	storage, err := repo.NewStorage()
	if err != nil {
		return nil, err
	}
	cacheKey, err := storage.GetCacheKeyForPath(repoPath)
	if err != nil {
		return nil, err
	}
	if !storage.IndexExists(cacheKey) {
		return nil, fmt.Errorf("repository index not ready — wait for indexing to finish")
	}
	return storage.LoadIndex(cacheKey)
}
