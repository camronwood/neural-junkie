package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/camronwood/neural-junkie/internal/lsp/server"
	"github.com/gorilla/websocket"
)

func handleLSPWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireIDEPack(w) {
		return
	}
	wsID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if wsID == "" || lang == "" {
		http.Error(w, "workspace and lang required", http.StatusBadRequest)
		return
	}
	backend, err := workspaceBackendResolver.ForWorkspace(wsID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(wsID)
	if ok && isRemoteWorkspace(ws) {
		proxySidecarWebSocket(w, r, ws, "/api/lsp/ws?lang="+url.QueryEscape(lang))
		return
	}
	sess, err := lspManager.GetOrStart(r.Context(), wsID, lang, backend.Root())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[LSP] websocket upgrade: %v", err)
		return
	}
	defer conn.Close()

	notifyCh := sess.Subscribe()
	defer sess.Unsubscribe(notifyCh)

	errCh := make(chan error, 2)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			var req server.JSONRPCRequest
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}
			if req.ID == 0 {
				_ = sess.Notify(req.Method, req.Params)
				continue
			}
			result, callErr := sess.Request(req.Method, req.Params)
			resp := server.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID}
			if callErr != nil {
				if je, ok := callErr.(*server.JSONRPCError); ok {
					resp.Error = je
				} else {
					resp.Error = &server.JSONRPCError{Code: -32603, Message: callErr.Error()}
				}
			} else {
				resp.Result = result
			}
			out, _ := json.Marshal(resp)
			if err := conn.WriteMessage(websocket.TextMessage, out); err != nil {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		for payload := range notifyCh {
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				errCh <- err
				return
			}
		}
	}()
	<-errCh
}

func handleLSPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireIDEPack(w) {
		return
	}
	var body struct {
		WorkspaceID string          `json:"workspace_id"`
		Lang        string          `json:"lang"`
		Method      string          `json:"method"`
		Params      json.RawMessage `json:"params"`
		URI         string          `json:"uri"`
		Text        string          `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if ws, ok := workspaceManager.GetWorkspace(body.WorkspaceID); ok && isRemoteWorkspace(ws) {
		rewind, _ := json.Marshal(body)
		proxySidecarHTTPBytes(w, r, ws, "/api/lsp/request", rewind)
		return
	}
	backend, err := workspaceBackendResolver.ForWorkspace(body.WorkspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	sess, err := lspManager.GetOrStart(r.Context(), body.WorkspaceID, body.Lang, backend.Root())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if body.Text != "" && body.URI != "" {
		_ = sess.DidOpen(body.URI, body.Lang, body.Text)
	}
	var params interface{}
	if len(body.Params) > 0 {
		_ = json.Unmarshal(body.Params, &params)
	}
	result, err := sess.Request(body.Method, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(result)
}

func handleLSPHover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireIDEPack(w) {
		return
	}
	var body struct {
		WorkspaceID string `json:"workspace_id"`
		Lang        string `json:"lang"`
		URI         string `json:"uri"`
		Text        string `json:"text"`
		Line        int    `json:"line"`
		Character   int    `json:"character"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if ws, ok := workspaceManager.GetWorkspace(body.WorkspaceID); ok && isRemoteWorkspace(ws) {
		rewind, _ := json.Marshal(body)
		proxySidecarHTTPBytes(w, r, ws, "/api/lsp/hover", rewind)
		return
	}
	backend, err := workspaceBackendResolver.ForWorkspace(body.WorkspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	sess, err := lspManager.GetOrStart(r.Context(), body.WorkspaceID, body.Lang, backend.Root())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if body.Text != "" && body.URI != "" {
		_ = sess.DidOpen(body.URI, body.Lang, body.Text)
	}
	result, err := sess.Hover(body.URI, body.Line, body.Character)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(result)
}
