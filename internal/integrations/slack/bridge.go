package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
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
	displayName string
	iconURL     string

	socketConnected atomic.Bool
	seenInbound     sync.Map // channelID:slackTS → struct{}, dedupe duplicate Socket Mode events
	userNames       sync.Map // slack user id → cachedSlackUser (display + handle)

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

	return &Bridge{
		cfg:         cfg,
		hub:         hub,
		ensure:      ensure,
		bindings:    bindings,
		threads:     threads,
		api:         api,
		socket:      socket,
		outbound:    make(map[string]context.CancelFunc),
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

// API returns the Slack Web API client.
func (b *Bridge) API() *slackapi.Client {
	return b.api
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
	case socketmode.EventTypeConnected:
		b.socketConnected.Store(true)
		log.Printf("[slack] socket mode: connected")
	case socketmode.EventTypeConnecting:
		b.socketConnected.Store(false)
		log.Printf("[slack] socket mode: connecting")
	case socketmode.EventTypeConnectionError:
		b.socketConnected.Store(false)
		if ce, ok := evt.Data.(*slackapi.ConnectionErrorEvent); ok && ce != nil {
			log.Printf("[slack] socket mode: connection_error (attempt %d, retry in %s): %v",
				ce.Attempt, ce.Backoff, ce.ErrorObj)
		} else {
			log.Printf("[slack] socket mode: connection_error")
		}
	case socketmode.EventTypeDisconnect:
		b.socketConnected.Store(false)
		log.Printf("[slack] socket mode: disconnected")
	}
}

func (b *Bridge) handleEventsAPI(ctx context.Context, event slackevents.EventsAPIEvent) {
	innerType := string(event.InnerEvent.Type)
	switch innerType {
	case string(slackevents.Message):
		if m, ok := event.InnerEvent.Data.(*slackevents.MessageEvent); ok {
			if InboundDebugEnabled() {
				log.Printf("[slack] event message channel=%s user=%s subtype=%q text_len=%d",
					m.Channel, m.User, m.SubType, len(strings.TrimSpace(m.Text)))
			}
			b.handleMessage(ctx, m, false)
			return
		}
		if InboundDebugEnabled() {
			log.Printf("[slack] event message: inner data type %T", event.InnerEvent.Data)
		}
	case string(slackevents.AppMention):
		if m, ok := event.InnerEvent.Data.(*slackevents.AppMentionEvent); ok {
			if InboundDebugEnabled() {
				log.Printf("[slack] event app_mention channel=%s", m.Channel)
			}
			b.handleAppMention(ctx, m)
			return
		}
	default:
		if InboundDebugEnabled() && innerType != "" {
			log.Printf("[slack] unhandled event type %q (subscribe in Slack → Event Subscriptions?)", innerType)
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
		if InboundDebugEnabled() {
			log.Printf("[slack] inbound ignored channel=%s (bot/subtype/empty)", in.ChannelID)
		}
		return
	}
	binding, ok := b.bindings.GetBySlackChannel(in.ChannelID)
	if !ok {
		if InboundDebugEnabled() {
			log.Printf("[slack] inbound ignored channel=%s (no binding; bound ids: %s)",
				in.ChannelID, b.boundSlackChannelIDs())
		}
		return
	}
	if !ShouldTriggerAgent(in, binding, b.botUserID) {
		if InboundDebugEnabled() {
			log.Printf("[slack] inbound ignored channel=%s (policy %s)", in.ChannelID, binding.Policy)
		}
		return
	}
	if in.SlackTS != "" {
		dedupeKey := in.ChannelID + ":" + in.SlackTS
		if _, loaded := b.seenInbound.LoadOrStore(dedupeKey, struct{}{}); loaded {
			if InboundDebugEnabled() {
				log.Printf("[slack] inbound dedupe channel=%s ts=%s", in.ChannelID, in.SlackTS)
			}
			return
		}
	}
	active := *binding
	if resolved, err := b.hub.ResolveAgentID(active.AgentID, active.AgentName); err == nil {
		active.AgentID = resolved
	} else if InboundDebugEnabled() {
		log.Printf("[slack] inbound agent resolve: %v", err)
	}
	b.resolveInboundUserIdentity(&in)
	msg := BuildHubMessage(in, &active, b.threads, b.botUserID)
	if err := b.hub.SendMessage(msg); err != nil {
		log.Printf("[slack] SendMessage: %v", err)
		return
	}
	agentShort := active.AgentID
	if len(agentShort) > 8 {
		agentShort = agentShort[:8]
	}
	log.Printf("[slack] inbound → hub %s from %s (agent %s)", active.NJChannel, in.UserName, agentShort)
	if msg.ID != "" && in.SlackTS != "" {
		_ = b.threads.RegisterNJMessageSlackTS(msg.ID, in.SlackTS)
	}
	parentTS := in.SlackTS
	if in.ThreadTS != "" {
		parentTS = in.ThreadTS
	}
	_ = b.threads.RegisterChannelParent(in.ChannelID, parentTS)
	if msg.IsThreadReply && msg.ThreadID == in.ThreadTS {
		_ = b.threads.RegisterInboundRoot(in.ChannelID, in.ThreadTS, msg.ThreadID)
	}
}

func (b *Bridge) runOutbound(ctx context.Context, njChannel string) {
	sub, err := b.hub.Subscribe(njChannel)
	if err != nil {
		log.Printf("[slack] outbound subscribe %s: %v", njChannel, err)
		return
	}
	defer b.hub.Unsubscribe(njChannel, sub)
	log.Printf("[slack] outbound listening on %s (hub subscribe)", njChannel)

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			binding, ok := b.bindings.GetByNJChannel(njChannel)
			if !ok {
				continue
			}
			if !ShouldPostToSlack(msg, binding) {
				continue
			}
			text := FormatSlackText(msg)
			threadTS := ThreadTSForOutbound(msg, b.threads, binding)
			username := OutboundSlackUsername(msg, binding, b.displayName)
			b.postSlack(binding.SlackChannelID, text, threadTS, username, *msg)
		}
	}
}

func (b *Bridge) postSlack(channelID, text, threadTS, username string, hubMsg protocol.Message) {
	opts := []slackapi.MsgOption{
		slackapi.MsgOptionText(text, false),
	}
	if threadTS != "" {
		opts = append(opts, slackapi.MsgOptionTS(threadTS))
	}
	if username == "" {
		username = b.displayName
	}
	if username != "" {
		opts = append(opts, slackapi.MsgOptionUsername(username))
	}
	if b.iconURL != "" {
		opts = append(opts, slackapi.MsgOptionIconURL(b.iconURL))
	}
	_, ts, err := b.api.PostMessage(channelID, opts...)
	if err != nil {
		log.Printf("[slack] postMessage: %v", err)
		return
	}
	if hubMsg.ID != "" && ts != "" {
		_ = b.threads.RegisterNJMessageSlackTS(hubMsg.ID, ts)
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

func (b *Bridge) boundSlackChannelIDs() string {
	var ids []string
	for _, binding := range b.bindings.List() {
		if binding.Enabled {
			ids = append(ids, binding.SlackChannelID)
		}
	}
	return strings.Join(ids, ", ")
}

// Status returns connection metadata for the API.
func (b *Bridge) Status() map[string]interface{} {
	return map[string]interface{}{
		"enabled":          b.cfg.Slack.Enabled,
		"connected":      b.botUserID != "" && b.socketConnected.Load(),
		"socket_connected": b.socketConnected.Load(),
		"bot_user_id":      b.botUserID,
		"team_id":          b.teamID,
		"bindings_count":   len(b.bindings.List()),
		"display_name":     b.displayName,
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
