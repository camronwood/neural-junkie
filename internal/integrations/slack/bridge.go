package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/gorilla/websocket"
	slackapi "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// Bridge connects Slack Socket Mode to the Neural Junkie hub.
type Bridge struct {
	cfg      *config.Config
	hub      HubClient
	ensure   AgentEnsurer
	bindings *BindingStore
	threads  *ThreadMap

	api       *slackapi.Client
	socket    *socketmode.Client
	botUserID string
	teamID    string

	mu          sync.Mutex
	outbound    map[string]context.CancelFunc
	wsURL       string
	displayName string
	iconURL     string

	ctx    context.Context
	cancel context.CancelFunc
}

// NewBridge creates a bridge (does not start Socket Mode until Start).
func NewBridge(cfg *config.Config, hub HubClient, ensure AgentEnsurer) (*Bridge, error) {
	if cfg == nil || !cfg.Slack.SlackReady() {
		return nil, fmt.Errorf("slack not configured")
	}
	bindings, err := NewBindingStore()
	if err != nil {
		return nil, err
	}
	threads, err := NewThreadMap()
	if err != nil {
		return nil, err
	}
	api := slackapi.New(cfg.Slack.BotToken, slackapi.OptionAppLevelToken(cfg.Slack.AppToken))
	socket := socketmode.New(api)
	host := cfg.Server.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Server.Port
	if port == 0 {
		port = 18765
	}
	wsScheme := "ws"
	if host != "localhost" && host != "127.0.0.1" {
		wsScheme = "ws"
	}
	wsURL := fmt.Sprintf("%s://%s:%d/ws", wsScheme, host, port)

	return &Bridge{
		cfg:         cfg,
		hub:         hub,
		ensure:      ensure,
		bindings:    bindings,
		threads:     threads,
		api:         api,
		socket:      socket,
		outbound:    make(map[string]context.CancelFunc),
		wsURL:       wsURL,
		displayName: cfg.Slack.EffectiveDisplayName(),
		iconURL:     cfg.Slack.DisplayIconURL,
	}, nil
}

// Bindings returns the binding store for API handlers.
func (b *Bridge) Bindings() *BindingStore {
	return b.bindings
}

// Threads returns the thread map.
func (b *Bridge) Threads() *ThreadMap {
	return b.threads
}

// BotUserID returns the cached Slack bot user id.
func (b *Bridge) BotUserID() string {
	return b.botUserID
}

// TeamID returns the workspace team id.
func (b *Bridge) TeamID() string {
	return b.teamID
}

// ApplyBinding applies a single binding to the hub.
func (b *Bridge) ApplyBinding(ctx context.Context, binding Binding) error {
	return ApplyBinding(ctx, b.hub, b.ensure, binding)
}

// Start runs Socket Mode and outbound hub subscriptions.
func (b *Bridge) Start(parent context.Context) error {
	if b.cancel != nil {
		return fmt.Errorf("slack bridge already started")
	}
	auth, err := b.api.AuthTest()
	if err != nil {
		return fmt.Errorf("slack auth.test: %w", err)
	}
	b.botUserID = auth.UserID
	b.teamID = auth.TeamID
	log.Printf("[slack] connected as @%s (team %s)", auth.User, auth.TeamID)

	ctx, cancel := context.WithCancel(parent)
	b.ctx = ctx
	b.cancel = cancel

	if err := ApplyAllBindings(ctx, b.hub, b.ensure, b.bindings); err != nil {
		log.Printf("[slack] apply bindings: %v", err)
	}
	b.refreshOutboundSubscriptions()

	go b.runSocketMode(ctx)
	return nil
}

// Stop shuts down Socket Mode and outbound listeners.
func (b *Bridge) Stop() {
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
	b.mu.Lock()
	for _, c := range b.outbound {
		c()
	}
	b.outbound = make(map[string]context.CancelFunc)
	b.mu.Unlock()
}

func (b *Bridge) refreshOutboundSubscriptions() {
	b.mu.Lock()
	defer b.mu.Unlock()
	active := make(map[string]bool)
	for _, binding := range b.bindings.List() {
		if !binding.Enabled {
			continue
		}
		active[binding.NJChannel] = true
		if _, ok := b.outbound[binding.NJChannel]; ok {
			continue
		}
		chName := binding.NJChannel
		ctx, cancel := context.WithCancel(b.ctx)
		b.outbound[chName] = cancel
		go b.runOutbound(ctx, chName)
	}
	for ch, cancel := range b.outbound {
		if !active[ch] {
			cancel()
			delete(b.outbound, ch)
		}
	}
}

func (b *Bridge) runSocketMode(ctx context.Context) {
	go func() {
		for evt := range b.socket.Events {
			b.handleSocketEvent(ctx, evt)
		}
	}()
	if err := b.socket.RunContext(ctx); err != nil && ctx.Err() == nil {
		log.Printf("[slack] socket mode ended: %v", err)
	}
}

func (b *Bridge) handleSocketEvent(ctx context.Context, evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		eventsAPI, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return
		}
		b.handleEventsAPI(ctx, eventsAPI)
	case socketmode.EventTypeConnecting, socketmode.EventTypeConnectionError:
		log.Printf("[slack] socket mode: %s", evt.Type)
	}
}

func (b *Bridge) handleEventsAPI(ctx context.Context, event slackevents.EventsAPIEvent) {
	switch event.InnerEvent.Type {
	case string(slackevents.Message):
		if m, ok := event.InnerEvent.Data.(*slackevents.MessageEvent); ok {
			b.handleMessage(ctx, m, false)
		}
	case string(slackevents.AppMention):
		if m, ok := event.InnerEvent.Data.(*slackevents.AppMentionEvent); ok {
			b.handleAppMention(ctx, m)
		}
	}
}

func (b *Bridge) handleAppMention(ctx context.Context, m *slackevents.AppMentionEvent) {
	in := InboundInput{
		WorkspaceID:  b.teamID,
		ChannelID:    m.Channel,
		UserID:       m.User,
		Text:         m.Text,
		SlackTS:      m.TimeStamp,
		ThreadTS:     m.ThreadTimeStamp,
		IsAppMention: true,
	}
	b.processInbound(ctx, in)
}

func (b *Bridge) handleMessage(ctx context.Context, m *slackevents.MessageEvent, forceMention bool) {
	in := InboundInput{
		WorkspaceID: b.teamID,
		ChannelID:   m.Channel,
		UserID:      m.User,
		Text:        m.Text,
		SlackTS:     m.TimeStamp,
		ThreadTS:    m.ThreadTimeStamp,
		BotID:       m.BotID,
		Subtype:     m.SubType,
	}
	if forceMention {
		in.IsAppMention = true
	}
	b.processInbound(ctx, in)
}

func (b *Bridge) processInbound(ctx context.Context, in InboundInput) {
	if ShouldIgnoreInbound(in, b.botUserID) {
		return
	}
	binding, ok := b.bindings.GetBySlackChannel(in.ChannelID)
	if !ok {
		return
	}
	if !ShouldTriggerAgent(in, binding, b.botUserID) {
		return
	}
	if in.UserName == "" && in.UserID != "" {
		if u, err := b.api.GetUserInfo(in.UserID); err == nil && u != nil {
			in.UserName = u.RealName
			if in.UserName == "" {
				in.UserName = u.Name
			}
		}
	}
	if in.UserName == "" {
		in.UserName = "Slack User"
	}
	msg := BuildHubMessage(in, binding, b.threads, b.botUserID)
	if err := b.hub.SendMessage(msg); err != nil {
		log.Printf("[slack] SendMessage: %v", err)
		return
	}
	if msg.IsThreadReply && msg.ThreadID == in.ThreadTS {
		_ = b.threads.RegisterInboundRoot(in.ChannelID, in.ThreadTS, msg.ThreadID)
	}
}

func (b *Bridge) runOutbound(ctx context.Context, njChannel string) {
	u := b.wsURL + "?channel=" + njChannel
	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, u, nil)
	if err != nil {
		log.Printf("[slack] outbound ws %s: %v", njChannel, err)
		return
	}
	defer conn.Close()
	log.Printf("[slack] outbound listening on %s", njChannel)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		var msg protocol.Message
		if err := conn.ReadJSON(&msg); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[slack] outbound read %s: %v", njChannel, err)
			time.Sleep(2 * time.Second)
			return
		}
		binding, ok := b.bindings.GetByNJChannel(njChannel)
		if !ok {
			continue
		}
		if !ShouldPostToSlack(&msg, binding) {
			continue
		}
		text := FormatSlackText(&msg)
		threadTS := ThreadTSForOutbound(&msg, b.threads, binding)
		b.postSlack(binding.SlackChannelID, text, threadTS, msg)
	}
}

func (b *Bridge) postSlack(channelID, text, threadTS string, hubMsg protocol.Message) {
	opts := []slackapi.MsgOption{
		slackapi.MsgOptionText(text, false),
	}
	if threadTS != "" {
		opts = append(opts, slackapi.MsgOptionTS(threadTS))
	}
	if b.displayName != "" {
		opts = append(opts, slackapi.MsgOptionUsername(b.displayName))
	}
	if b.iconURL != "" {
		opts = append(opts, slackapi.MsgOptionIconURL(b.iconURL))
	}
	_, ts, err := b.api.PostMessage(channelID, opts...)
	if err != nil {
		log.Printf("[slack] postMessage: %v", err)
		return
	}
	tid := hubMsg.GetThreadID()
	if tid == "" {
		tid = hubMsg.ThreadID
	}
	if tid != "" {
		_ = b.threads.RegisterOutbound(tid, ts)
	}
}

// PostTestMessage sends a test line to a Slack channel (API test endpoint).
func (b *Bridge) PostTestMessage(channelID, text string) error {
	if text == "" {
		text = "Neural Junkie Slack bridge test."
	}
	_, _, err := b.api.PostMessage(channelID,
		slackapi.MsgOptionText(text, false),
		slackapi.MsgOptionUsername(b.displayName),
	)
	return err
}

// ReloadBindings reapplies hub state and restarts outbound listeners.
func (b *Bridge) ReloadBindings(ctx context.Context) error {
	if err := b.bindings.Reload(); err != nil {
		return err
	}
	if err := ApplyAllBindings(ctx, b.hub, b.ensure, b.bindings); err != nil {
		return err
	}
	b.refreshOutboundSubscriptions()
	return nil
}

// Status returns connection metadata for the API.
func (b *Bridge) Status() map[string]interface{} {
	return map[string]interface{}{
		"enabled":        b.cfg.Slack.Enabled,
		"connected":      b.botUserID != "",
		"bot_user_id":    b.botUserID,
		"team_id":        b.teamID,
		"bindings_count": len(b.bindings.List()),
		"display_name":   b.displayName,
	}
}

// ParseSlackOAuthResponse extracts tokens from oauth.v2.access JSON.
func ParseSlackOAuthResponse(data []byte) (botToken, appToken string, err error) {
	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", "", err
	}
	if !resp.OK {
		return "", "", fmt.Errorf("slack oauth: %s", resp.Error)
	}
	return resp.AccessToken, "", nil
}
