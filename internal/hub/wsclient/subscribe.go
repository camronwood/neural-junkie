package wsclient

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/gorilla/websocket"
)

const (
	defaultReconnectMin = 500 * time.Millisecond
	defaultReconnectMax = 15 * time.Second
)

// Subscribe opens a WebSocket to the hub for channelName and delivers messages on the returned channel.
// serverHTTPBase is e.g. http://localhost:18765. Reconnects with exponential backoff until stop is closed.
func Subscribe(serverHTTPBase, channelName string, stop <-chan struct{}) (chan *protocol.Message, error) {
	channelName = strings.TrimSpace(channelName)
	if channelName == "" {
		return nil, fmt.Errorf("channel name required")
	}
	wsURL, err := channelWSURL(serverHTTPBase, channelName, nil, nil)
	if err != nil {
		return nil, err
	}

	out := make(chan *protocol.Message, 512)
	go runSubscribeLoop(wsURL, stop, out, nil)
	return out, nil
}

// SubscribeWithAuth opens a WebSocket to the hub using hub token and/or session query params.
// This is needed for browser-style clients (and LAN guests) that cannot set custom headers.
func SubscribeWithAuth(serverHTTPBase, channelName, hubToken, sessionToken string, stop <-chan struct{}) (chan *protocol.Message, error) {
	channelName = strings.TrimSpace(channelName)
	if channelName == "" {
		return nil, fmt.Errorf("channel name required")
	}
	auth := &wsAuth{
		hubToken:     strings.TrimSpace(hubToken),
		sessionToken: strings.TrimSpace(sessionToken),
	}
	wsURL, err := channelWSURL(serverHTTPBase, channelName, nil, auth)
	if err != nil {
		return nil, err
	}
	out := make(chan *protocol.Message, 512)
	go runSubscribeLoop(wsURL, stop, out, nil)
	return out, nil
}

// SubscribeAgent connects to /api/agents/ws for push delivery on all agent channels.
func SubscribeAgent(serverHTTPBase, agentID string, stop <-chan struct{}, onHold func(channel string, held bool)) (chan *protocol.Message, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil, fmt.Errorf("agent_id required")
	}
	wsURL, err := agentWSURL(serverHTTPBase, agentID)
	if err != nil {
		return nil, err
	}
	out := make(chan *protocol.Message, 512)
	go runSubscribeLoop(wsURL, stop, out, onHold)
	return out, nil
}

type wsAuth struct {
	hubToken     string
	sessionToken string
}

func channelWSURL(serverHTTPBase, channel string, extra []string, auth *wsAuth) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(serverHTTPBase), "/")
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		u.Scheme = "ws"
	}
	u.Path = "/ws"
	q := u.Query()
	q.Set("channel", channel)
	if len(extra) > 0 {
		q.Set("extra", strings.Join(extra, ","))
	}
	if auth != nil {
		if auth.hubToken != "" {
			q.Set("hub_token", auth.hubToken)
		}
		if auth.sessionToken != "" {
			q.Set("nj_session", auth.sessionToken)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func agentWSURL(serverHTTPBase, agentID string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(serverHTTPBase), "/")
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		u.Scheme = "ws"
	}
	u.Path = "/api/agents/ws"
	q := u.Query()
	q.Set("agent_id", agentID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func runSubscribeLoop(wsURL string, stop <-chan struct{}, out chan *protocol.Message, onHold func(channel string, held bool)) {
	defer close(out)
	backoff := defaultReconnectMin
	for {
		select {
		case <-stop:
			return
		default:
		}
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			log.Printf("[wsclient] dial %s: %v (retry in %s)", wsURL, err, backoff)
			if !sleepOrStop(stop, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}
		backoff = defaultReconnectMin
		conn.SetReadLimit(16 << 20)
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		})

		pingStop := make(chan struct{})
		var pingWG sync.WaitGroup
		pingWG.Add(1)
		go func() {
			defer pingWG.Done()
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-pingStop:
					return
				case <-stop:
					return
				case <-ticker.C:
					if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
						return
					}
				}
			}
		}()

		readDone := make(chan struct{})
		go func() {
			defer close(readDone)
			for {
				var msg protocol.Message
				if err := conn.ReadJSON(&msg); err != nil {
					return
				}
				if msg.Type == protocol.MessageTypeAgentStatus && msg.Metadata != nil && onHold != nil {
					if v, ok := msg.Metadata[protocol.MetadataChannelHold].(bool); ok {
						onHold(msg.Channel, v)
					}
				}
				select {
				case out <- &msg:
				case <-stop:
					return
				}
			}
		}()

		select {
		case <-stop:
			close(pingStop)
			conn.Close()
			pingWG.Wait()
			return
		case <-readDone:
			close(pingStop)
			conn.Close()
			pingWG.Wait()
			if !sleepOrStop(stop, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
		}
	}
}

func sleepOrStop(stop <-chan struct{}, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-stop:
		return false
	case <-t.C:
		return true
	}
}

func nextBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > defaultReconnectMax {
		return defaultReconnectMax
	}
	return next
}

// ParseWSMessage decodes a raw WS frame for tests.
func ParseWSMessage(data []byte) (*protocol.Message, error) {
	var msg protocol.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
