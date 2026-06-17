package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/gorilla/websocket"
)

func proxySidecarWebSocket(w http.ResponseWriter, r *http.Request, ws *hub.Workspace, sidecarPath string) {
	token, _ := hub.GetRemoteToken(ws.ID)
	sidecarWS := strings.Replace(ws.SidecarURL, "http://", "ws://", 1)
	sidecarWS = strings.Replace(sidecarWS, "https://", "wss://", 1)
	u, err := url.Parse(sidecarWS + sidecarPath)
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
		http.Error(w, "sidecar ws: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer remoteConn.Close()

	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[sidecar-proxy] upgrade: %v", err)
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

func proxySidecarHTTP(w http.ResponseWriter, r *http.Request, ws *hub.Workspace, sidecarPath string) {
	proxySidecarHTTPBytes(w, r, ws, sidecarPath, nil)
}

func proxySidecarHTTPBytes(w http.ResponseWriter, r *http.Request, ws *hub.Workspace, sidecarPath string, body []byte) {
	token, _ := hub.GetRemoteToken(ws.ID)
	target := strings.TrimRight(ws.SidecarURL, "/") + sidecarPath
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	} else if r.Body != nil {
		bodyReader = r.Body
	}
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, bodyReader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header.Clone()
	if token != "" {
		proxyReq.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		http.Error(w, "sidecar: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
