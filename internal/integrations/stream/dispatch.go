package stream

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/connectors"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// Actions is the hub surface used by the stream dispatcher.
type Actions interface {
	TriggerRunbookDefinition(defID string, version int, req hub.RunbookCreateRequest) (*hub.TriggerRunbookResult, error)
	SendMessage(msg *protocol.Message) error
}

// DispatchResult describes the outcome of processing one event.
type DispatchResult struct {
	Matched bool   `json:"matched"`
	Fired   bool   `json:"fired"`
	Skipped bool   `json:"skipped"`
	Reason  string `json:"reason,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Dispatcher matches, debounces, and fires subscription actions.
type Dispatcher struct {
	Actions Actions
	HTTP    *http.Client

	mu       sync.Mutex
	debounce map[string]time.Time
}

// NewDispatcher creates a dispatcher.
func NewDispatcher(actions Actions) *Dispatcher {
	return &Dispatcher{
		Actions:  actions,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
		debounce: map[string]time.Time{},
	}
}

// Handle processes a subscription event. sub should be a snapshot of the binding.
func (d *Dispatcher) Handle(ctx context.Context, sub Subscription, ev Event) DispatchResult {
	if !MatchEvent(sub.Match, ev.Payload) {
		return DispatchResult{Matched: false, Reason: "no match"}
	}
	if sub.DebounceMs > 0 && d.isDebounced(sub, ev) {
		return DispatchResult{Matched: true, Skipped: true, Reason: "debounced"}
	}
	if err := d.fire(ctx, sub, ev); err != nil {
		msg := err.Error()
		if isConcurrencyCapError(msg) {
			log.Printf("[stream] skip runbook for %s: %v", sub.ID, err)
			return DispatchResult{Matched: true, Skipped: true, Reason: "concurrency_cap", Error: msg}
		}
		return DispatchResult{Matched: true, Error: msg}
	}
	return DispatchResult{Matched: true, Fired: true}
}

func (d *Dispatcher) isDebounced(sub Subscription, ev Event) bool {
	sum := sha256.Sum256([]byte(ev.Payload))
	key := sub.ID + "|" + ev.Topic + "|" + hex.EncodeToString(sum[:])
	now := time.Now()
	window := time.Duration(sub.DebounceMs) * time.Millisecond
	d.mu.Lock()
	defer d.mu.Unlock()
	if last, ok := d.debounce[key]; ok && now.Sub(last) < window {
		return true
	}
	d.debounce[key] = now
	// Opportunistic prune of stale keys
	if len(d.debounce) > 5000 {
		for k, t := range d.debounce {
			if now.Sub(t) > window {
				delete(d.debounce, k)
			}
		}
	}
	return false
}

func (d *Dispatcher) fire(ctx context.Context, sub Subscription, ev Event) error {
	switch sub.Action.Type {
	case ActionRunbook:
		return d.fireRunbook(sub, ev)
	case ActionChannel:
		return d.fireChannel(sub, ev)
	case ActionWebhook:
		return d.fireWebhook(ctx, sub, ev)
	default:
		return fmt.Errorf("unknown action type %q", sub.Action.Type)
	}
}

func (d *Dispatcher) fireRunbook(sub Subscription, ev Event) error {
	if d.Actions == nil {
		return fmt.Errorf("hub actions unavailable")
	}
	inputs := BuildRunbookInputs(ev, sub.Action.InputMap)
	ch := sub.Action.Channel
	if ch == "" {
		ch = "general"
	}
	_, err := d.Actions.TriggerRunbookDefinition(sub.Action.DefinitionID, sub.Action.Version, hub.RunbookCreateRequest{
		AgentIDs:  sub.Action.AgentIDs,
		Channel:   ch,
		CreatedBy: "stream",
		RunInputs: inputs,
	})
	return err
}

func (d *Dispatcher) fireChannel(sub Subscription, ev Event) error {
	if d.Actions == nil {
		return fmt.Errorf("hub actions unavailable")
	}
	tmpl := sub.Action.MessageTemplate
	if tmpl == "" {
		tmpl = "Stream event on {{topic}}:\n{{payload}}"
	}
	content := RenderTemplate(tmpl, ev.Payload, ev.Topic, ev.Key)
	mentions := append([]string{}, sub.Action.MentionAgentIDs...)
	if len(mentions) > 0 {
		var prefix strings.Builder
		for _, id := range mentions {
			prefix.WriteString("@")
			prefix.WriteString(id)
			prefix.WriteString(" ")
		}
		content = prefix.String() + content
	}
	msg := &protocol.Message{
		ID:        uuid.New().String(),
		Type:      protocol.MessageTypeChat,
		Channel:   sub.Action.HubChannel,
		Content:   content,
		Timestamp: time.Now().UTC(),
		From: protocol.AgentInfo{
			ID:   "stream",
			Name: "Stream",
			Type: protocol.AgentTypeGeneral,
		},
		Mentions: mentions,
		Metadata: map[string]interface{}{
			"stream_subscription_id": sub.ID,
			"stream_topic":           ev.Topic,
			"stream_protocol":        string(ev.Protocol),
		},
		Tags: []string{"stream"},
	}
	return d.Actions.SendMessage(msg)
}

func (d *Dispatcher) fireWebhook(ctx context.Context, sub Subscription, ev Event) error {
	url := strings.TrimSpace(sub.Action.URLOverride)
	var prof *connectors.Profile
	if sub.Action.WebhookConnectorID != "" {
		p, err := connectors.Get(sub.Action.WebhookConnectorID)
		if err != nil {
			return err
		}
		prof = p
		if url == "" {
			url = strings.TrimSpace(p.Config["url"])
		}
	}
	if url == "" {
		return fmt.Errorf("webhook url not configured")
	}
	body := RenderTemplate(sub.Action.BodyTemplate, ev.Payload, ev.Topic, ev.Key)
	if sub.Action.BodyTemplate == "" {
		// Prefer structured JSON envelope when template empty
		env, _ := json.Marshal(map[string]string{
			"topic":   ev.Topic,
			"key":     ev.Key,
			"payload": ev.Payload,
		})
		body = string(env)
	}
	cfg := map[string]interface{}{"url": url}
	cfg = connectors.ApplyToHTTPConfig(cfg, prof)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if headers, ok := cfg["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}
	client := d.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}

func isConcurrencyCapError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "maximum concurrent") ||
		strings.Contains(lower, "max concurrent") ||
		strings.Contains(lower, "concurrent collaboration") ||
		strings.Contains(lower, "too many active")
}
