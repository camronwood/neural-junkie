package main

import (
	"context"
	"log"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func ensureHiddenRepoAgentForWorkspace(ws *hub.Workspace) {
	if ws == nil || chatHub == nil {
		return
	}
	ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler)
	if !ok || ch == nil {
		return
	}
	isRemote := ws.Kind == workspacebackend.KindSSH || ws.Kind == workspacebackend.KindDevcontainer
	opts := hub.EnsureHiddenRepoAgentOptions{
		IsRemote:    isRemote,
		WorkspaceID: ws.ID,
	}
	if isRemote {
		if b, err := workspaceBackendResolver.ForWorkspace(ws.ID); err == nil {
			opts.Backend = b
		}
	}
	if _, err := ch.EnsureHiddenRepoAgent(context.Background(), ws.Path, opts); err != nil {
		log.Printf("[hidden-repo] ensure workspace %q: %v", ws.Name, err)
	}
}

func stopHiddenRepoAgentForWorkspace(ws *hub.Workspace) {
	if ws == nil || chatHub == nil {
		return
	}
	ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler)
	if !ok || ch == nil {
		return
	}
	ch.StopHiddenRepoAgent(ws.Path)
}

func reconcileHiddenRepoAgentsOnStartup() {
	if chatHub == nil || workspaceManager == nil {
		return
	}
	ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler)
	if !ok || ch == nil {
		return
	}
	workspaces := workspaceManager.ListWorkspaces()
	ch.ReconcileHiddenRepoAgentsForWorkspaces(context.Background(), workspaces, func(ws *hub.Workspace) workspacebackend.Backend {
		if ws == nil {
			return nil
		}
		isRemote := ws.Kind == workspacebackend.KindSSH || ws.Kind == workspacebackend.KindDevcontainer
		if !isRemote {
			return nil
		}
		b, err := workspaceBackendResolver.ForWorkspace(ws.ID)
		if err != nil {
			return nil
		}
		return b
	})
}
