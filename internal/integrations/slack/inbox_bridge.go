package slack

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	slackapi "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const nonOwnerDMReply = "This Neural Junkie bot is linked to a personal inbox on another user's machine."

// Inbox returns the inbox store.
func (b *Bridge) Inbox() *InboxStore {
	return b.inbox
}

// ReloadInbox reapplies inbox hub state and restarts the inbox outbound listener
// only when subscription targets change (agent/channel), not on every settings save.
func (b *Bridge) ReloadInbox(ctx context.Context) error {
	if b.inbox == nil {
		return nil
	}
	if err := b.inbox.Reload(); err != nil {
		return err
	}
	ReconcileInboxAgentID(b.inbox, b.hub)
	cfg := b.inbox.Get()
	if cfg.Enabled {
		ensureKey := inboxEnsureKey(cfg)
		b.mu.Lock()
		needEnsure := ensureKey != b.lastInboxEnsureKey
		b.mu.Unlock()

		var ensure AgentEnsurer
		if needEnsure {
			ensure = b.ensure
		}
		if err := ApplyInbox(ctx, b.hub, ensure, cfg); err != nil {
			log.Printf("[slack] apply inbox: %v", err)
		} else if needEnsure {
			b.mu.Lock()
			b.lastInboxEnsureKey = ensureKey
			b.mu.Unlock()
		}
	} else {
		b.mu.Lock()
		b.lastInboxEnsureKey = ""
		b.mu.Unlock()
	}

	outboundKey := inboxOutboundKey(cfg, b.hub)
	b.mu.Lock()
	needsOutboundRestart := outboundKey != b.lastInboxOutboundKey || b.inboxOutboundCancel == nil
	if needsOutboundRestart {
		b.lastInboxOutboundKey = outboundKey
	}
	b.mu.Unlock()
	if needsOutboundRestart {
		b.refreshInboxOutboundSubscription()
	}
	_ = b.ReloadUserTokens()
	return nil
}

// ReloadUserTokens re-reads the encrypted user OAuth token from disk.
func (b *Bridge) ReloadUserTokens() error {
	if b.userTokens == nil {
		return nil
	}
	return b.userTokens.Reload()
}

func inboxEnsureKey(cfg InboxConfig) string {
	if !cfg.Enabled {
		return ""
	}
	channel := cfg.NJChannel
	if channel == "" {
		channel = NJInboxChannelName(cfg.OwnerSlackUserID)
	}
	return cfg.AgentID + "|" + channel
}

func inboxOutboundKey(cfg InboxConfig, hub HubClient) string {
	if !cfg.Enabled {
		return "disabled"
	}
	channel := cfg.NJChannel
	if channel == "" {
		channel = NJInboxChannelName(cfg.OwnerSlackUserID)
	}
	key := channel
	if hub != nil {
		key += "|peers:" + strconv.Itoa(len(inboxPeerHubChannels(cfg, hub)))
	}
	return key
}

func inboxPeerHubChannels(cfg InboxConfig, hub HubClient) []string {
	if hub == nil || cfg.OwnerSlackUserID == "" {
		return nil
	}
	prefix := NJInboxPeerChannelPrefix(cfg.OwnerSlackUserID)
	if prefix == "" {
		return nil
	}
	var out []string
	for _, ch := range hub.ListChannels() {
		if ch != nil && strings.HasPrefix(ch.Name, prefix) {
			out = append(out, ch.Name)
		}
	}
	return out
}

func inboxOutboundChannels(cfg InboxConfig, hub HubClient) []string {
	seen := map[string]struct{}{}
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
	}
	main := cfg.NJChannel
	if main == "" {
		main = NJInboxChannelName(cfg.OwnerSlackUserID)
	}
	add(main)
	for _, name := range inboxPeerHubChannels(cfg, hub) {
		add(name)
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	return out
}

func (b *Bridge) refreshInboxOutboundSubscription() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.inboxOutboundCancel != nil {
		b.inboxOutboundCancel()
		b.inboxOutboundCancel = nil
	}
	if b.inbox == nil || b.ctx == nil {
		return
	}
	cfg := b.inbox.Get()
	if !cfg.Enabled || cfg.NJChannel == "" {
		return
	}
	ctx, cancel := context.WithCancel(b.ctx)
	b.inboxOutboundCancel = cancel
	go b.runInboxOutbound(ctx)
}

func (b *Bridge) runInboxOutbound(ctx context.Context) {
	cfg := b.inbox.Get()
	if !cfg.Enabled || cfg.NJChannel == "" {
		return
	}
	channels := inboxOutboundChannels(cfg, b.hub)
	if len(channels) == 0 {
		return
	}
	subs := make(map[string]chan *protocol.Message, len(channels))
	for _, channelName := range channels {
		sub, err := b.hub.Subscribe(channelName)
		if err != nil {
			log.Printf("[slack] inbox outbound subscribe %s: %v", channelName, err)
			continue
		}
		subs[channelName] = sub
		log.Printf("[slack] inbox outbound listening on %s", channelName)
	}
	defer func() {
		for channelName, sub := range subs {
			b.hub.Unsubscribe(channelName, sub)
		}
	}()

	out := make(chan *protocol.Message, len(subs)*4)
	var wg sync.WaitGroup
	for channelName, sub := range subs {
		channelName, sub := channelName, sub
		wg.Add(1)
		go func() {
			defer wg.Done()
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
					select {
					case out <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
		_ = channelName
	}
	go func() {
		wg.Wait()
		close(out)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-out:
			if !ok {
				return
			}
			b.handleInboxOutboundMessage(msg, msg.Channel)
		}
	}
}

func (b *Bridge) handleInboxOutboundMessage(msg *protocol.Message, channelName string) {
	if msg == nil {
		return
	}
	inbox := b.inbox.Get()
	if resolved, err := b.hub.ResolveAgentID(inbox.AgentID, inbox.AgentName); err == nil {
		inbox.AgentID = resolved
	}
	if !ShouldPostInboxToSlack(msg, &inbox) {
		return
	}
	channelID, threadTS := InboxOutboundTarget(msg, &inbox, b.threads)
	if channelID == "" {
		return
	}
	if InboxOutboundHumanDM(msg, b.threads, &inbox) {
		text := FormatSlackText(msg)
		if msg.From.ID == inbox.AgentID {
			text = FormatHumanDMOutboundText(msg, &inbox)
		}
		b.postUserSlack(channelID, text)
		log.Printf("[slack] human_dm outbound → Slack channel=%s (hub %s)", channelID, channelName)
		return
	}
	text := FormatSlackText(msg)
	username := OutboundInboxSlackUsername(msg, &inbox, b.displayName)
	b.postSlack(channelID, text, threadTS, username, msg)
	if InboundDebugEnabled() {
		log.Printf("[slack] inbox outbound → Slack channel=%s thread=%s (hub %s)", channelID, threadTS, channelName)
	}
}

func (b *Bridge) inboxRequiresManualReply(inbox InboxConfig) bool {
	userTokenSet := b.userTokens != nil && b.userTokens.HasToken()
	return inbox.ForwardEnabled && !ShouldAutoReplyHumanDMs(inbox, userTokenSet, time.Now())
}

func (b *Bridge) routeInbound(ctx context.Context, in InboundInput) {
	if ShouldIgnoreInbound(in, b.botUserID) {
		if InboundDebugEnabled() {
			log.Printf("[slack] inbound ignored channel=%s (bot/subtype/empty)", in.ChannelID)
		}
		return
	}

	inbox := b.inbox.Get()

	if IsIMChannel(in.ChannelID) {
		if InboundDebugEnabled() || strings.TrimSpace(in.Text) != "" {
			log.Printf("[slack] im inbound channel=%s user=%s text_len=%d", in.ChannelID, in.UserID, len(strings.TrimSpace(in.Text)))
		}
		b.routeIMInbound(ctx, in, inbox)
		return
	}

	if binding, ok := b.bindings.GetBySlackChannel(in.ChannelID); ok {
		b.processBindingInbound(ctx, in, binding)
		return
	}

	if inbox.Enabled && inbox.ForwardEnabled && inbox.OwnerSlackUserID != "" {
		if match, ok := EvaluateForwardRules(in, inbox.ForwardRules, inbox.OwnerSlackUserID); ok {
			b.processInboxInbound(ctx, in, &inbox, match)
			return
		}
	}

	if InboundDebugEnabled() {
		log.Printf("[slack] inbound ignored channel=%s (no binding or forward rule)", in.ChannelID)
	}
}

func (b *Bridge) routeIMInbound(ctx context.Context, in InboundInput, inbox InboxConfig) {
	if !inbox.Enabled {
		if InboundDebugEnabled() {
			log.Printf("[slack] inbox disabled; ignoring DM channel=%s", in.ChannelID)
		}
		return
	}
	if !InboxOwnerAllowed(&inbox, in.UserID) {
		b.postSlack(in.ChannelID, nonOwnerDMReply, "", b.displayName, nil)
		return
	}
	_ = b.inbox.UpdateDMChannelID(in.ChannelID)
	inbox = b.inbox.Get()
	b.processInboxInbound(ctx, in, &inbox, nil)
}

func (b *Bridge) processBindingInbound(ctx context.Context, in InboundInput, binding *Binding) {
	if !ShouldTriggerAgent(in, binding, b.botUserID) {
		log.Printf("[slack] inbound ignored channel=%s (policy %q — use always, or @mention the bot / app_mention)",
			in.ChannelID, binding.Policy)
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
	b.registerBindingThreadState(in, msg)
}

func (b *Bridge) processInboxInbound(ctx context.Context, in InboundInput, inbox *InboxConfig, forward *ForwardMatch) {
	if inbox == nil || !inbox.Enabled {
		return
	}
	dedupeKey := in.ChannelID + ":" + in.SlackTS
	if in.SlackTS != "" {
		if forward != nil {
			dedupeKey = "fwd:" + dedupeKey
		}
		if _, loaded := b.seenInbound.LoadOrStore(dedupeKey, struct{}{}); loaded {
			if InboundDebugEnabled() {
				log.Printf("[slack] inbox dedupe %s", dedupeKey)
			}
			return
		}
	}

	active := *inbox
	if resolved, err := b.hub.ResolveAgentID(active.AgentID, active.AgentName); err == nil {
		active.AgentID = resolved
	} else if InboundDebugEnabled() {
		log.Printf("[slack] inbox agent resolve: %v", err)
	}

	work := in
	if forward != nil {
		b.enrichForwardMatch(forward, &work)
		if forward.StrippedText != "" {
			work.Text = forward.StrippedText
		}
	}

	b.resolveInboundUserIdentity(&work)
	msg := BuildInboxMessage(work, &active, b.threads, forward, "")
	if forward != nil && b.inboxRequiresManualReply(active) {
		ApplyInboxManualReply(msg)
	}
	if err := b.hub.SendMessage(msg); err != nil {
		log.Printf("[slack] inbox SendMessage: %v", err)
		return
	}

	if IsIMChannel(work.ChannelID) {
		_ = b.inbox.UpdateDMChannelID(work.ChannelID)
	}

	replyChannel := msg.SlackReplyChannelID()
	replyThread := msg.SlackReplyThreadTS()
	if replyChannel != "" {
		routeKey := msg.ID
		if msg.ThreadID != "" {
			routeKey = msg.ThreadID
		}
		_ = b.threads.RegisterInboxReplyRoute(routeKey, replyChannel, replyThread)
		if msg.ID != "" && routeKey != msg.ID {
			_ = b.threads.RegisterInboxReplyRoute(msg.ID, replyChannel, replyThread)
		}
	}

	b.registerBindingThreadState(work, msg)

	kind := "dm"
	if forward != nil {
		kind = "forward:" + string(forward.RuleType)
	}
	log.Printf("[slack] inbox → hub %s (%s) from %s", active.NJChannel, kind, work.UserName)
}

func (b *Bridge) enrichForwardMatch(forward *ForwardMatch, in *InboundInput) {
	if forward == nil || in == nil {
		return
	}
	if forward.SourceChannelName == "" {
		forward.SourceChannelName = ResolveChannelName(b.api, forward.SourceChannelID)
	}
	if forward.SourceAuthor == "" && in.UserID != "" {
		b.resolveInboundUserIdentity(in)
		forward.SourceAuthor = in.UserName
	}
	if forward.Permalink == "" && forward.SourceChannelID != "" && forward.SourceTS != "" {
		if link, err := b.api.GetPermalink(&slackapi.PermalinkParameters{
			Channel: forward.SourceChannelID,
			Ts:      forward.SourceTS,
		}); err == nil {
			forward.Permalink = link
		}
	}
}

func (b *Bridge) registerBindingThreadState(in InboundInput, msg *protocol.Message) {
	if msg == nil {
		return
	}
	if msg.ID != "" && in.SlackTS != "" {
		_ = b.threads.RegisterNJMessageSlackTS(msg.ID, in.SlackTS)
	}
	parentTS := in.SlackTS
	if in.ThreadTS != "" {
		parentTS = in.ThreadTS
	}
	_ = b.threads.RegisterChannelParent(in.ChannelID, parentTS)
	if msg.IsThreadReply && in.ThreadTS != "" {
		rootID := msg.ThreadID
		if rootID == "" {
			rootID = in.ThreadTS
		}
		_ = b.threads.RegisterInboundRoot(in.ChannelID, in.ThreadTS, rootID)
	}
}

func (b *Bridge) handleReactionAdded(ctx context.Context, ev *slackevents.ReactionAddedEvent) {
	if ev == nil || ev.Item.Type != "message" {
		return
	}
	inbox := b.inbox.Get()
	if !inbox.Enabled || inbox.OwnerSlackUserID == "" {
		return
	}
	if strings.TrimSpace(ev.User) != strings.TrimSpace(inbox.OwnerSlackUserID) {
		return
	}
	channelID := ev.Item.Channel
	messageTS := ev.Item.Timestamp
	if channelID == "" || messageTS == "" {
		return
	}

	var matched *ForwardRule
	for i := range inbox.ForwardRules {
		rule := inbox.ForwardRules[i]
		if !rule.Enabled || rule.Type != ForwardRuleReaction {
			continue
		}
		if !emojiMatches(ev.Reaction, rule.Emoji) {
			continue
		}
		if !channelMatchesRule(channelID, rule.SlackChannelIDs) {
			continue
		}
		matched = &inbox.ForwardRules[i]
		break
	}
	if matched == nil {
		return
	}

	dedupeKey := "react:" + channelID + ":" + messageTS + ":" + ev.Reaction
	if _, loaded := b.seenInbound.LoadOrStore(dedupeKey, struct{}{}); loaded {
		return
	}

	text, authorID := b.fetchSlackMessageText(channelID, messageTS)
	in := InboundInput{
		WorkspaceID: b.teamID,
		ChannelID:   channelID,
		UserID:      authorID,
		Text:        text,
		SlackTS:     messageTS,
	}
	b.resolveInboundUserIdentity(&in)
	channelName := ResolveChannelName(b.api, channelID)
	permalink := ""
	if link, err := b.api.GetPermalink(&slackapi.PermalinkParameters{Channel: channelID, Ts: messageTS}); err == nil {
		permalink = link
	}
	forward := ReactionForwardMatch(*matched, channelID, messageTS, "", text, in.UserName, channelName, permalink)
	b.processInboxInbound(ctx, in, &inbox, forward)
}

func emojiMatches(got, want string) bool {
	got = strings.TrimSpace(strings.TrimPrefix(got, ":"))
	want = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(want), ":"), ":"))
	if want == "" {
		want = "robot_face"
	}
	return got == want
}

func (b *Bridge) fetchSlackMessageText(channelID, ts string) (text, userID string) {
	hist, err := b.api.GetConversationHistory(&slackapi.GetConversationHistoryParameters{
		ChannelID: channelID,
		Latest:    ts,
		Inclusive: true,
		Limit:     1,
	})
	if err != nil || len(hist.Messages) == 0 {
		return "", ""
	}
	m := hist.Messages[0]
	return strings.TrimSpace(m.Text), m.User
}

// PostInboxTestDM sends a test message to the owner's DM channel.
func (b *Bridge) PostInboxTestDM(inbox InboxConfig, text string) error {
	channelID, err := b.resolveOwnerBotDMChannel(inbox.OwnerSlackUserID)
	if err != nil {
		return err
	}
	_ = b.inbox.UpdateDMChannelID(channelID)
	if text == "" {
		text = "Neural Junkie personal inbox test — reply here anytime; this is your DM with the bot (not note-to-self)."
	}
	return b.PostTestMessage(channelID, text)
}
