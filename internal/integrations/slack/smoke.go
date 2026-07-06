package slack

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
	slackapi "github.com/slack-go/slack"
)

const smokeInboundMarker = "[nj-smoke] synthetic inbound"
const smokeOutboundMarker = "[nj-smoke] outbound test"

// SmokeCheck is one step in a smoke run report.
type SmokeCheck struct {
	ID     string `json:"id"`
	Status string `json:"status"` // pass | warn | fail | skip
	Detail string `json:"detail,omitempty"`
}

// SmokeResult is returned by POST /api/slack/smoke/run.
type SmokeResult struct {
	OK              bool         `json:"ok"`
	Checks          []SmokeCheck `json:"checks"`
	DurationMS      int64        `json:"duration_ms"`
	OutboundSkipped bool         `json:"outbound_skipped"`
	ChannelID       string       `json:"channel_id,omitempty"`
}

// SmokeOptions configures a smoke run.
type SmokeOptions struct {
	ChannelID     string
	SkipInbound   bool
	Outbound      bool
	AllowOutbound bool
	EnvAllow      bool // SLACK_SMOKE_ALLOW=1 on hub
}

// HubMessageReader reads recent hub channel messages (for smoke verification).
type HubMessageReader interface {
	GetMessages(channelName string, limit int) ([]*protocol.Message, error)
}

var defaultSmokeChannelNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^nj-smoke(-test)?$`),
}

// SmokeAllowOutboundEnv reports whether the hub permits real Slack posts in smoke.
func SmokeAllowOutboundEnv() bool {
	return strings.TrimSpace(os.Getenv("SLACK_SMOKE_ALLOW")) == "1"
}

func smokeChannelNamePatterns() []*regexp.Regexp {
	raw := strings.TrimSpace(os.Getenv("SLACK_SMOKE_CHANNEL_NAMES"))
	if raw == "" {
		return defaultSmokeChannelNamePatterns
	}
	var out []*regexp.Regexp
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		re, err := regexp.Compile(part)
		if err != nil {
			continue
		}
		out = append(out, re)
	}
	if len(out) == 0 {
		return defaultSmokeChannelNamePatterns
	}
	return out
}

func channelNameAllowed(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, re := range smokeChannelNamePatterns() {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// ValidateSmokeChannel ensures outbound smoke may post to channelID (private nj-smoke*, small membership).
func ValidateSmokeChannel(api *slackapi.Client, channelID string) error {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return fmt.Errorf("smoke outbound blocked: channel_id required")
	}
	if IsIMChannel(channelID) {
		return fmt.Errorf("smoke outbound blocked: IM/DM channels are not allowed")
	}
	if api == nil {
		return fmt.Errorf("smoke outbound blocked: slack api unavailable")
	}
	ch, err := api.GetConversationInfo(&slackapi.GetConversationInfoInput{ChannelID: channelID})
	if err != nil {
		return fmt.Errorf("smoke outbound blocked: %w", err)
	}
	if ch == nil {
		return fmt.Errorf("smoke outbound blocked: channel not found")
	}
	if ch.IsIM || ch.IsMpIM {
		return fmt.Errorf("smoke outbound blocked: direct message channels are not allowed")
	}
	if !ch.IsPrivate {
		return fmt.Errorf("smoke outbound blocked: channel must be private (create #nj-smoke-test)")
	}
	if !channelNameAllowed(ch.Name) {
		return fmt.Errorf("smoke outbound blocked: channel name %q must match nj-smoke-test allowlist", ch.Name)
	}
	if ch.NumMembers > 0 && ch.NumMembers > 2 {
		return fmt.Errorf("smoke outbound blocked: channel has %d members (max 2: bot + you)", ch.NumMembers)
	}
	return nil
}

// SmokeInbound ingests a synthetic Slack line into the hub without routing to an agent.
func SmokeInbound(hub HubClient, binding *Binding, threads *ThreadMap, botUserID, text string) (*protocol.Message, error) {
	if hub == nil || binding == nil {
		return nil, fmt.Errorf("hub and binding required")
	}
	if text == "" {
		text = smokeInboundMarker
	}
	in := InboundInput{
		ChannelID:    binding.SlackChannelID,
		UserID:       "USMOKE",
		UserName:     "NJ Smoke",
		Text:         text,
		SlackTS:      fmt.Sprintf("%d.000001", time.Now().Unix()),
		IsAppMention: true,
	}
	msg := BuildHubMessage(in, binding, threads, botUserID)
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	delete(msg.Metadata, protocol.SlackMetaRouteAgentID)
	delete(msg.Metadata, protocol.SlackMetaAppMention)
	msg.Mentions = nil
	msg.Metadata["nj_smoke"] = true
	if err := hub.SendMessage(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func waitForHubMessage(reader HubMessageReader, njChannel, wantText string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msgs, err := reader.GetMessages(njChannel, 20)
		if err != nil {
			return err
		}
		for _, m := range msgs {
			if m == nil {
				continue
			}
			if strings.Contains(m.Content, wantText) {
				if src, _ := m.Metadata["source"].(string); src == "slack" || m.Metadata["nj_smoke"] == true {
					return nil
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("hub message not found within %s", timeout)
}

func firstEnabledBindingChannel(store *BindingStore) string {
	if store == nil {
		return ""
	}
	for _, b := range store.List() {
		if b.Enabled && strings.TrimSpace(b.SlackChannelID) != "" {
			return b.SlackChannelID
		}
	}
	return ""
}

// RunSmoke executes diagnose + optional synthetic inbound + optional gated outbound.
func RunSmoke(ctx context.Context, cfg *config.Config, api *slackapi.Client, hub HubClient, bridge *Bridge, reader HubMessageReader, store *BindingStore, opts SmokeOptions) SmokeResult {
	start := time.Now()
	res := SmokeResult{OutboundSkipped: true}

	add := func(id, status, detail string) {
		res.Checks = append(res.Checks, SmokeCheck{ID: id, Status: status, Detail: detail})
	}

	channelID := strings.TrimSpace(opts.ChannelID)
	if channelID == "" {
		channelID = firstEnabledBindingChannel(store)
	}
	res.ChannelID = channelID

	if opts.Outbound && (!opts.AllowOutbound || !opts.EnvAllow) {
		add("outbound_post", "fail", "outbound requires allow_outbound in request and SLACK_SMOKE_ALLOW=1 on hub")
		res.OK = false
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}

	diag := Diagnose(cfg)
	if !diag.AuthTestOK {
		add("auth_test", "fail", "auth.test failed")
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}
	add("auth_test", "pass", "ok")

	if !diag.SocketOpenOK {
		detail := diag.SocketOpenError
		if detail == "" {
			detail = "socket mode open failed"
		}
		add("socket_open", "fail", detail)
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}
	add("socket_open", "pass", "ok")

	if bridge != nil && bridge.socketConnected.Load() {
		add("bridge_connected", "pass", "socket mode running")
	} else {
		add("bridge_connected", "warn", "bridge not connected (diagnose socket open ok)")
	}

	if !opts.SkipInbound {
		if channelID == "" {
			add("synthetic_inbound", "warn", "no channel_id and no enabled binding — skipped")
		} else if store == nil {
			add("synthetic_inbound", "fail", "binding store unavailable")
		} else if binding, ok := store.GetBySlackChannel(channelID); !ok {
			add("synthetic_inbound", "warn", fmt.Sprintf("no enabled binding for %s — skipped", channelID))
		} else if reader == nil {
			add("synthetic_inbound", "fail", "hub message reader unavailable")
		} else if hub == nil {
			add("synthetic_inbound", "fail", "hub unavailable")
		} else {
			var threads *ThreadMap
			botUserID := ""
			if bridge != nil {
				threads = bridge.threads
				botUserID = bridge.botUserID
			}
			if threads == nil {
				var err error
				threads, err = NewThreadMap()
				if err != nil {
					add("synthetic_inbound", "fail", err.Error())
					res.DurationMS = time.Since(start).Milliseconds()
					return res
				}
			}
			if _, err := SmokeInbound(hub, binding, threads, botUserID, smokeInboundMarker); err != nil {
				add("synthetic_inbound", "fail", err.Error())
			} else if err := waitForHubMessage(reader, binding.NJChannel, smokeInboundMarker, 5*time.Second); err != nil {
				add("synthetic_inbound", "fail", err.Error())
			} else {
				add("synthetic_inbound", "pass", "hub ingested synthetic slack message (no agent dispatch)")
			}
		}
	} else {
		add("synthetic_inbound", "skip", "skip_inbound requested")
	}

	if opts.Outbound {
		if channelID == "" {
			add("outbound_post", "fail", "channel_id required for outbound smoke")
		} else if api == nil {
			add("outbound_post", "fail", "slack api unavailable")
		} else if err := ValidateSmokeChannel(api, channelID); err != nil {
			add("outbound_post", "fail", err.Error())
		} else {
			text := smokeOutboundMarker + " " + time.Now().UTC().Format(time.RFC3339)
			ts, err := postSmokeOutbound(api, channelID, text)
			if err != nil {
				add("outbound_post", "fail", err.Error())
			} else {
				add("outbound_post", "pass", "posted")
				if err := verifySmokeHistory(api, channelID, text); err != nil {
					add("outbound_verify", "fail", err.Error())
				} else {
					add("outbound_verify", "pass", "found in history")
					if _, _, err := api.DeleteMessage(channelID, ts); err != nil {
						add("outbound_delete", "warn", err.Error())
					} else {
						add("outbound_delete", "pass", "deleted test message")
					}
				}
				res.OutboundSkipped = false
			}
		}
	} else {
		add("outbound_post", "skip", "outbound disabled (default — no Slack messages sent)")
	}

	res.OK = true
	for _, c := range res.Checks {
		if c.Status == "fail" {
			res.OK = false
			break
		}
	}
	res.DurationMS = time.Since(start).Milliseconds()
	return res
}

func postSmokeOutbound(api *slackapi.Client, channelID, text string) (string, error) {
	_, ts, err := api.PostMessage(channelID, slackapi.MsgOptionText(text, false))
	return ts, err
}

func verifySmokeHistory(api *slackapi.Client, channelID, text string) error {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		hist, err := api.GetConversationHistory(&slackapi.GetConversationHistoryParameters{
			ChannelID: channelID,
			Limit:     10,
		})
		if err != nil {
			return err
		}
		for _, m := range hist.Messages {
			if strings.Contains(m.Text, text) || strings.Contains(m.Text, smokeOutboundMarker) {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("outbound message not found in channel history")
}
