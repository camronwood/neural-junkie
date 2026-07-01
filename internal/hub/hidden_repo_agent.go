package hub

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/codeindex"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// EnsureHiddenRepoAgentOptions tunes hidden repo agent creation.
type EnsureHiddenRepoAgentOptions struct {
	Backend     workspacebackend.Backend
	IsRemote    bool
	WorkspaceID string
}

// EnsureHiddenRepoAgent creates or returns a consult-only repo agent for a workspace path.
func (ch *CommandHandler) EnsureHiddenRepoAgent(ctx context.Context, repoPath string, opts EnsureHiddenRepoAgentOptions) (*agent.RepoAgent, error) {
	if ch == nil || ch.hub == nil {
		return nil, fmt.Errorf("command handler not configured")
	}
	if ch.appConfig != nil && !ch.appConfig.WorkspaceIndex.AutoIndexOnAddEnabled() {
		return nil, nil
	}

	norm := normalizeRepoAgentPath(repoPath, opts.IsRemote)
	if norm == "" {
		return nil, fmt.Errorf("empty repository path")
	}

	if existing := ch.findRepoAgentByPath(norm); existing != nil {
		return existing, nil
	}

	agentName := hiddenRepoAgentName(norm)
	repoAgent, err := agent.NewRepoAgentWithOptions(agentName, norm, ch.aiProvider, ch.hub, agent.RepoAgentOptions{
		SkipPathCheck: opts.IsRemote,
		ConsultOnly:   true,
	})
	if err != nil {
		return nil, err
	}
	if opts.Backend != nil {
		repoAgent.SetIndexBackend(opts.Backend)
	}

	if err := ch.hub.RegisterAgent(&repoAgent.Info); err != nil {
		return nil, fmt.Errorf("register hidden repo agent: %w", err)
	}

	ch.agentsMu.Lock()
	ch.repoAgents[repoAgent.Info.ID] = repoAgent
	ch.agentsMu.Unlock()

	if err := repoAgent.StartIndexingOnly(ctx); err != nil {
		return nil, err
	}

	if opts.Backend != nil {
		go func() {
			c, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			_ = codeindex.BuildIndexViaBackend(c, norm, opts.Backend)
		}()
	} else {
		codeindex.BuildIndexAsync(norm)
	}

	log.Printf("[hidden-repo] ensured consult-only repo agent %q for %s", agentName, norm)
	return repoAgent, nil
}

// RepoAgentForPath returns the repo agent bound to a normalized repository path.
func (ch *CommandHandler) RepoAgentForPath(repoPath string) (*agent.RepoAgent, bool) {
	if ch == nil {
		return nil, false
	}
	norm := normalizeRepoAgentPath(repoPath, false)
	if norm == "" {
		return nil, false
	}
	ra := ch.findRepoAgentByPath(norm)
	return ra, ra != nil
}

// RepoAgentByID returns an in-process repo agent by hub agent ID.
func (ch *CommandHandler) RepoAgentByID(agentID string) (*agent.RepoAgent, bool) {
	if ch == nil || strings.TrimSpace(agentID) == "" {
		return nil, false
	}
	ch.agentsMu.RLock()
	defer ch.agentsMu.RUnlock()
	ra, ok := ch.repoAgents[agentID]
	return ra, ok && ra != nil
}

// ConsultRepoForPath runs an internal repo consult for a workspace path.
func (ch *CommandHandler) ConsultRepoForPath(ctx context.Context, repoPath, subQuestion, channel string) (text, agentName string, err error) {
	ra, ok := ch.RepoAgentForPath(repoPath)
	if !ok || ra == nil {
		return "", "", fmt.Errorf("no repo agent for path %q", repoPath)
	}
	text, err = ra.GenerateConsultResponse(ctx, subQuestion, channel)
	return text, ra.Info.Name, err
}

// StopHiddenRepoAgent stops an in-memory consult-only repo agent for a path (cache retained).
func (ch *CommandHandler) StopHiddenRepoAgent(repoPath string) {
	if ch == nil {
		return
	}
	norm := normalizeRepoAgentPath(repoPath, false)
	ch.agentsMu.Lock()
	defer ch.agentsMu.Unlock()
	for id, ra := range ch.repoAgents {
		if ra == nil || !ra.Info.ConsultOnly {
			continue
		}
		if normalizeRepoAgentPath(ra.Info.RepositoryPath, false) != norm {
			continue
		}
		ra.Stop()
		delete(ch.repoAgents, id)
		_ = ch.hub.UnregisterAgent(id)
		log.Printf("[hidden-repo] stopped consult-only agent for %s", norm)
	}
}

// ReconcileHiddenRepoAgentsForWorkspaces ensures hidden agents exist for all registered workspaces.
func (ch *CommandHandler) ReconcileHiddenRepoAgentsForWorkspaces(ctx context.Context, workspaces []*Workspace, backendFn func(ws *Workspace) workspacebackend.Backend) {
	if ch == nil || ch.appConfig == nil || !ch.appConfig.WorkspaceIndex.AutoIndexOnAddEnabled() {
		return
	}
	for _, ws := range workspaces {
		if ws == nil || strings.TrimSpace(ws.Path) == "" {
			continue
		}
		isRemote := ws.Kind == workspacebackend.KindSSH || ws.Kind == workspacebackend.KindDevcontainer
		opts := EnsureHiddenRepoAgentOptions{IsRemote: isRemote, WorkspaceID: ws.ID}
		if backendFn != nil {
			opts.Backend = backendFn(ws)
		}
		if _, err := ch.EnsureHiddenRepoAgent(ctx, ws.Path, opts); err != nil {
			log.Printf("[hidden-repo] reconcile %q: %v", ws.Path, err)
		}
	}
}

func (ch *CommandHandler) findRepoAgentByPath(norm string) *agent.RepoAgent {
	ch.agentsMu.RLock()
	defer ch.agentsMu.RUnlock()
	for _, ra := range ch.repoAgents {
		if ra == nil {
			continue
		}
		if normalizeRepoAgentPath(ra.Info.RepositoryPath, false) == norm {
			return ra
		}
	}
	return nil
}

func hiddenRepoAgentName(repoPath string) string {
	base := filepath.Base(filepath.Clean(repoPath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "workspace"
	}
	return protocol.NormalizeAgentName("__index:" + base)
}

func normalizeRepoAgentPath(p string, isRemote bool) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if isRemote {
		return filepath.ToSlash(filepath.Clean(p))
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}

// AutoIndexWorkspacesEnabled reads config when appConfig is attached later.
func (ch *CommandHandler) AutoIndexWorkspacesEnabled() bool {
	if ch == nil || ch.appConfig == nil {
		return config.DefaultWorkspaceIndexConfig().AutoIndexOnAddEnabled()
	}
	return ch.appConfig.WorkspaceIndex.AutoIndexOnAddEnabled()
}
