package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

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

	// Room presence: mark the current session as connected while this WS is open.
	sess := hub.SessionFromRequest(r, hubSessions)
	var roomPresence []struct {
		roomID   string
		channel  string
		username string
	}

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
			chatHub.UnsubscribeUI(s.name, s.ch)
		}
	}()
	for _, name := range watch {
		if err := ensureChannelExistsForWS(name, sess); err != nil {
			log.Printf("Ensure channel %q for WS: %v", name, err)
		}
		msgCh, err := chatHub.SubscribeUI(name)
		if err != nil {
			log.Printf("Subscribe %q: %v", name, err)
			continue
		}
		if sess != nil {
			if ch, cerr := chatHub.GetChannel(name); cerr == nil && ch != nil && ch.Type == protocol.ChannelTypeRoom && ch.RoomID != "" {
				roomPresence = append(roomPresence, struct {
					roomID   string
					channel  string
					username string
				}{roomID: ch.RoomID, channel: name, username: sess.Username})
			}
		}
		subs = append(subs, wsSub{name: name, ch: msgCh})
	}
	if len(subs) == 0 {
		return
	}

	// Apply presence connect and broadcast for each unique room.
	roomSeen := make(map[string]struct{}, len(roomPresence))
	for _, p := range roomPresence {
		if _, ok := roomSeen[p.roomID]; ok {
			continue
		}
		roomSeen[p.roomID] = struct{}{}
		if chatHub.SetRoomMemberConnected(p.roomID, p.username, true) {
			broadcastRoomPresence(p.channel, p.roomID, p.username, true)
		}
	}
	defer func() {
		for _, p := range roomPresence {
			if chatHub.SetRoomMemberConnected(p.roomID, p.username, false) {
				broadcastRoomPresence(p.channel, p.roomID, p.username, false)
			}
		}
	}()

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

func broadcastRoomPresence(channel, roomID, username string, connected bool) {
	if chatHub == nil {
		return
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"",
	)
	msg.Timestamp = time.Now()
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["presence"] = true
	msg.Metadata["room_id"] = roomID
	msg.Metadata["username"] = username
	msg.Metadata["connected"] = connected
	chatHub.BroadcastDirect(channel, msg)
}

// ensureChannelExistsForWS recreates a missing owned DM shell so SubscribeUI succeeds
// after hub restart when the desktop still points at a saved DM channel name.
func ensureChannelExistsForWS(name string, sess *hub.HubSession) error {
	if chatHub == nil || strings.TrimSpace(name) == "" {
		return nil
	}
	if _, err := chatHub.GetChannel(name); err == nil {
		return nil
	}
	username := ""
	if sess != nil {
		username = sess.Username
	}
	if username == "" || !chatHub.CanUserAccessChannel(username, name) {
		return nil
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), "dm-") {
		return nil
	}
	_ = chatHub.CreateChannelWithType(
		name,
		"Direct message",
		"",
		protocol.ChannelTypeDM,
		username,
	)
	// Best-effort: re-join assistant/agent inferred from dm-{user}-{agentSlug}.
	parts := strings.Split(strings.ToLower(strings.TrimSpace(name)), "-")
	if len(parts) >= 3 {
		agentSlug := strings.Join(parts[2:], "-")
		for _, a := range chatHub.ListAgents() {
			if strings.ToLower(a.Name) == agentSlug || strings.EqualFold(string(a.Type), agentSlug) {
				_ = chatHub.JoinChannel(a.ID, name)
				break
			}
		}
	}
	return nil
}
