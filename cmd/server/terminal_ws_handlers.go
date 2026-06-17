package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
	"github.com/gorilla/websocket"
)

func handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	if wsID == "" {
		http.Error(w, "workspace required", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(wsID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	if !isRemoteWorkspace(ws) {
		http.Error(w, "terminal proxy only for remote workspaces", http.StatusBadRequest)
		return
	}
	token, _ := hub.GetRemoteToken(wsID)
	sidecarWS := strings.Replace(ws.SidecarURL, "http://", "ws://", 1)
	sidecarWS = strings.Replace(sidecarWS, "https://", "wss://", 1)
	u, err := url.Parse(sidecarWS + "/api/pty/ws")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	header := http.Header{}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	remoteConn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		http.Error(w, "sidecar pty: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer remoteConn.Close()

	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[terminal] upgrade: %v", err)
		return
	}
	defer clientConn.Close()

	errCh := make(chan error, 2)
	copyWS := func(dst, src *websocket.Conn) {
		for {
			mt, data, err := src.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			if err := dst.WriteMessage(mt, data); err != nil {
				errCh <- err
				return
			}
		}
	}
	go copyWS(remoteConn, clientConn)
	go copyWS(clientConn, remoteConn)
	<-errCh
}

func handleRemoteExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		WorkspaceID string   `json:"workspace_id"`
		Command     string   `json:"command"`
		Args        []string `json:"args"`
		Cwd         string   `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b, err := workspaceBackendResolver.ForWorkspace(body.WorkspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if b.Kind() == workspacebackend.KindLocal {
		http.Error(w, "use local terminal for local workspaces", http.StatusBadRequest)
		return
	}
	res, err := b.Exec(r.Context(), workspacebackend.ExecRequest{
		Command: body.Command,
		Args:    body.Args,
		RelCwd:  body.Cwd,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
