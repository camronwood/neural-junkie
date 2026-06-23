package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// handleAgentWebSocket pushes channel messages to a standalone agent over one connection.
// GET /api/agents/ws?agent_id=<uuid>
func handleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}

	channels := chatHub.GetAgentChannels(agentID)
	if len(channels) == 0 {
		http.Error(w, "agent not in any channel", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[agent-ws] upgrade: %v", err)
		return
	}
	defer conn.Close()

	type wsSub struct {
		name string
		ch   chan *protocol.Message
	}

	var subsMu sync.Mutex
	subs := make([]wsSub, 0, len(channels))
	defer func() {
		subsMu.Lock()
		defer subsMu.Unlock()
		for _, s := range subs {
			chatHub.Unsubscribe(s.name, s.ch)
		}
	}()

	subscribeAll := func(names []string) {
		subsMu.Lock()
		defer subsMu.Unlock()
		existing := make(map[string]struct{}, len(subs))
		for _, s := range subs {
			existing[s.name] = struct{}{}
		}
		for _, name := range names {
			if _, ok := existing[name]; ok {
				continue
			}
			msgCh, err := chatHub.Subscribe(name)
			if err != nil {
				log.Printf("[agent-ws] subscribe %q: %v", name, err)
				continue
			}
			subs = append(subs, wsSub{name: name, ch: msgCh})
			existing[name] = struct{}{}
		}
	}

	subscribeAll(channels)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	out := make(chan *protocol.Message, 512)
	var forwardWG sync.WaitGroup

	startForwarder := func(sub wsSub) {
		forwardWG.Add(1)
		go func() {
			defer forwardWG.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-sub.ch:
					if !ok {
						return
					}
					select {
					case out <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	subsMu.Lock()
	for _, sub := range subs {
		startForwarder(sub)
	}
	subsMu.Unlock()

	go func() {
		forwardWG.Wait()
		close(out)
	}()

	// Refresh subscriptions when agent joins new channels.
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				names := chatHub.GetAgentChannels(agentID)
				if len(names) == 0 {
					continue
				}
				subsMu.Lock()
				existing := make(map[string]struct{}, len(subs))
				for _, s := range subs {
					existing[s.name] = struct{}{}
				}
				for _, name := range names {
					if _, ok := existing[name]; ok {
						continue
					}
					msgCh, err := chatHub.Subscribe(name)
					if err != nil {
						continue
					}
					sub := wsSub{name: name, ch: msgCh}
					subs = append(subs, sub)
					startForwarder(sub)
				}
				subsMu.Unlock()
			}
		}
	}()

	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	})

	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				cancel()
				return
			}
			var ctrl struct {
				Type    string `json:"type"`
				Channel string `json:"channel"`
			}
			if json.Unmarshal(data, &ctrl) != nil {
				continue
			}
			switch ctrl.Type {
			case "ping":
				_ = conn.WriteJSON(map[string]string{"type": "pong"})
			case "refresh":
				names := chatHub.GetAgentChannels(agentID)
				subsMu.Lock()
				existing := make(map[string]struct{}, len(subs))
				for _, s := range subs {
					existing[s.name] = struct{}{}
				}
				for _, name := range names {
					if _, ok := existing[name]; ok {
						continue
					}
					msgCh, err := chatHub.Subscribe(name)
					if err != nil {
						continue
					}
					sub := wsSub{name: name, ch: msgCh}
					subs = append(subs, sub)
					startForwarder(sub)
				}
				subsMu.Unlock()
			}
		}
	}()

	for msg := range out {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[agent-ws] write: %v", err)
			break
		}
	}
}
