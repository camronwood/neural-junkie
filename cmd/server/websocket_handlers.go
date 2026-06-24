package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func wsEnsureChannelAccess(w http.ResponseWriter, r *http.Request, channels ...string) bool {
	if !hub.RequireHubAccess(w, r) {
		return false
	}
	seen := make(map[string]struct{}, len(channels))
	for _, ch := range channels {
		ch = strings.TrimSpace(ch)
		if ch == "" {
			continue
		}
		if _, ok := seen[ch]; ok {
			continue
		}
		seen[ch] = struct{}{}
		if !ensureChannelReadAccess(w, r, ch) {
			return false
		}
	}
	return true
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	threadID := r.URL.Query().Get("thread")
	extraParam := r.URL.Query().Get("extra")

	if channel == "" {
		channel = "general"
	}

	if threadID != "" {
		ch := threadChannelForACL(threadID, r)
		if ch == "" {
			ch = channel
		}
		if !wsEnsureChannelAccess(w, r, ch) {
			return
		}
	} else {
		watch := []string{channel}
		if extraParam != "" {
			for _, name := range strings.Split(extraParam, ",") {
				name = strings.TrimSpace(name)
				if name != "" {
					watch = append(watch, name)
				}
			}
		}
		if !wsEnsureChannelAccess(w, r, watch...) {
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Subscribe to thread or channel
	if threadID != "" {
		msgCh, err := chatHub.SubscribeToThread(threadID)
		if err != nil {
			log.Println("Thread subscribe error:", err)
			return
		}
		defer chatHub.UnsubscribeFromThread(threadID, msgCh)
		for msg := range msgCh {
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("Write error:", err)
				break
			}
		}
		return
	}

	watch := []string{channel}
	seen := map[string]struct{}{channel: {}}
	if extraParam != "" {
		for _, name := range strings.Split(extraParam, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			watch = append(watch, name)
		}
	}

	type wsSub struct {
		name string
		ch   chan *protocol.Message
	}
	subs := make([]wsSub, 0, len(watch))
	defer func() {
		for _, s := range subs {
			chatHub.Unsubscribe(s.name, s.ch)
		}
	}()
	for _, name := range watch {
		msgCh, err := chatHub.Subscribe(name)
		if err != nil {
			log.Printf("Subscribe %q: %v", name, err)
			continue
		}
		subs = append(subs, wsSub{name: name, ch: msgCh})
	}
	if len(subs) == 0 {
		return
	}

	out := make(chan *protocol.Message, 256)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var wg sync.WaitGroup
	for _, sub := range subs {
		sub := sub
		wg.Add(1)
		go func() {
			defer wg.Done()
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
	go func() {
		wg.Wait()
		close(out)
	}()

	for msg := range out {
		if err := conn.WriteJSON(msg); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}
