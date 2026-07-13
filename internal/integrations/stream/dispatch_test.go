package stream

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMatchEventEmpty(t *testing.T) {
	if !MatchEvent(nil, `{"a":1}`) {
		t.Fatal("nil match should always match")
	}
	if !MatchEvent(&MatchSpec{}, `{"a":1}`) {
		t.Fatal("empty path should match")
	}
}

func TestMatchEventEqualsAndContains(t *testing.T) {
	payload := `{"status":"alert","nested":{"code":"E1"}}`
	if !MatchEvent(&MatchSpec{JSONPath: "status", Op: MatchEquals, Value: "alert"}, payload) {
		t.Fatal("expected equals match")
	}
	if MatchEvent(&MatchSpec{JSONPath: "status", Op: MatchEquals, Value: "ok"}, payload) {
		t.Fatal("expected equals miss")
	}
	if !MatchEvent(&MatchSpec{JSONPath: "nested.code", Op: MatchContains, Value: "E"}, payload) {
		t.Fatal("expected contains match")
	}
	if MatchEvent(&MatchSpec{JSONPath: "missing", Op: MatchEquals, Value: "x"}, payload) {
		t.Fatal("missing path should not match")
	}
}

func TestBuildRunbookInputs(t *testing.T) {
	ev := Event{Topic: "devices/1", Key: "k1", Payload: `{"temp":22,"unit":"C"}`}
	in := BuildRunbookInputs(ev, map[string]string{"temp": "temperature"})
	if in["topic"] != "devices/1" || in["payload"] == "" || in["key"] != "k1" {
		t.Fatalf("base inputs: %#v", in)
	}
	if in["temperature"] != "22" {
		t.Fatalf("mapped temp: %#v", in)
	}
}

func TestRenderTemplate(t *testing.T) {
	got := RenderTemplate("t={{topic}} p={{payload}} k={{key}}", "body", "top", "ky")
	if got != "t=top p=body k=ky" {
		t.Fatalf("got %q", got)
	}
}

type fakeActions struct {
	triggerErr error
	triggered  int
	messages   []*protocol.Message
}

func (f *fakeActions) TriggerRunbookDefinition(defID string, version int, req hub.RunbookCreateRequest) (*hub.TriggerRunbookResult, error) {
	f.triggered++
	if f.triggerErr != nil {
		return nil, f.triggerErr
	}
	return &hub.TriggerRunbookResult{CollaborationID: "c1", CollaborationChannel: "collab-c1"}, nil
}

func (f *fakeActions) SendMessage(msg *protocol.Message) error {
	f.messages = append(f.messages, msg)
	return nil
}

func TestDispatcherChannelAction(t *testing.T) {
	fa := &fakeActions{}
	d := NewDispatcher(fa)
	sub := Subscription{
		ID: "s1",
		Action: ActionSpec{
			Type:            ActionChannel,
			HubChannel:      "general",
			MessageTemplate: "got {{payload}}",
			MentionAgentIDs: []string{"agent-1"},
		},
	}
	res := d.Handle(context.Background(), sub, Event{Topic: "t", Payload: "hello"})
	if !res.Fired || len(fa.messages) != 1 {
		t.Fatalf("res=%#v messages=%d", res, len(fa.messages))
	}
	if fa.messages[0].Channel != "general" {
		t.Fatalf("channel %q", fa.messages[0].Channel)
	}
	if fa.messages[0].Content != "@agent-1 got hello" {
		t.Fatalf("content %q", fa.messages[0].Content)
	}
}

func TestDispatcherRunbookConcurrencySkip(t *testing.T) {
	fa := &fakeActions{triggerErr: errString("maximum concurrent collaborations (3) reached")}
	d := NewDispatcher(fa)
	sub := Subscription{
		ID: "s1",
		Action: ActionSpec{
			Type:         ActionRunbook,
			DefinitionID: "health",
			AgentIDs:     []string{"a1"},
		},
	}
	res := d.Handle(context.Background(), sub, Event{Topic: "t", Payload: "{}"})
	if !res.Skipped || res.Reason != "concurrency_cap" {
		t.Fatalf("expected concurrency skip, got %#v", res)
	}
}

func TestDispatcherDebounce(t *testing.T) {
	fa := &fakeActions{}
	d := NewDispatcher(fa)
	sub := Subscription{
		ID:         "s1",
		DebounceMs: 5000,
		Action: ActionSpec{
			Type:            ActionChannel,
			HubChannel:      "general",
			MessageTemplate: "{{payload}}",
		},
	}
	ev := Event{Topic: "t", Payload: "same"}
	r1 := d.Handle(context.Background(), sub, ev)
	r2 := d.Handle(context.Background(), sub, ev)
	if !r1.Fired || !r2.Skipped || r2.Reason != "debounced" {
		t.Fatalf("r1=%#v r2=%#v", r1, r2)
	}
}

func TestDispatcherWebhook(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	fa := &fakeActions{}
	d := NewDispatcher(fa)
	d.HTTP = srv.Client()
	sub := Subscription{
		ID: "s1",
		Action: ActionSpec{
			Type:         ActionWebhook,
			URLOverride:  srv.URL,
			BodyTemplate: `{"x":"{{payload}}"}`,
		},
	}
	res := d.Handle(context.Background(), sub, Event{Topic: "t", Payload: "hi"})
	if !res.Fired {
		t.Fatalf("res=%#v", res)
	}
	if gotBody != `{"x":"hi"}` {
		t.Fatalf("body %q", gotBody)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}
	saved, err := store.Upsert(Subscription{
		Label:       "test",
		Enabled:     true,
		Protocol:    ProtocolMQTT,
		ConnectorID: "c1",
		Topic:       "devices/#",
		Action: ActionSpec{
			Type:       ActionChannel,
			HubChannel: "general",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" || saved.CreatedAt.IsZero() {
		t.Fatalf("saved %#v", saved)
	}
	store2, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := store2.Get(saved.ID)
	if !ok || got.Topic != "devices/#" {
		t.Fatalf("reload %#v ok=%v", got, ok)
	}
	if err := store2.Delete(saved.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store2.Get(saved.ID); ok {
		t.Fatal("expected deleted")
	}
}

func TestValidateSubscriptionRejectsBadAction(t *testing.T) {
	err := validateSubscription(Subscription{
		Protocol:    ProtocolMQTT,
		ConnectorID: "c",
		Topic:       "t",
		Action:      ActionSpec{Type: ActionRunbook},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
