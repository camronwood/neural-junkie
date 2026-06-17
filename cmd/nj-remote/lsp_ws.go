package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	lspserver "github.com/camronwood/neural-junkie/internal/lsp/server"
	"github.com/gorilla/websocket"
)

var (
	sidecarLSPManager = lspserver.NewManager()
	lspUpgrader       = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

func handleSidecarLSPWebSocket(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if *token != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got != *token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		lang := strings.TrimSpace(r.URL.Query().Get("lang"))
		if lang == "" {
			http.Error(w, "lang required", http.StatusBadRequest)
			return
		}
		sess, err := sidecarLSPManager.GetOrStart(r.Context(), "sidecar", lang, root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		conn, err := lspUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[nj-remote lsp] upgrade: %v", err)
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
				var req lspserver.JSONRPCRequest
				if err := json.Unmarshal(data, &req); err != nil {
					continue
				}
				if req.ID == 0 {
					_ = sess.Notify(req.Method, req.Params)
					continue
				}
				result, callErr := sess.Request(req.Method, req.Params)
				resp := lspserver.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID}
				if callErr != nil {
					if je, ok := callErr.(*lspserver.JSONRPCError); ok {
						resp.Error = je
					} else {
						resp.Error = &lspserver.JSONRPCError{Code: -32603, Message: callErr.Error()}
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
}
