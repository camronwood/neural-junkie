package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/artifacts"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestWantsMermaidCanvas(t *testing.T) {
	if !wantsMermaidCanvas("Create a Neural Canvas Mermaid diagram of this architecture") {
		t.Fatal("expected mermaid canvas ask")
	}
	if wantsMermaidCanvas("Create a Neural Canvas table of coverage gaps") {
		t.Fatal("table ask is not mermaid shortcut")
	}
	if wantsMermaidCanvas("draw me a sunset") {
		t.Fatal("image ask is not mermaid canvas")
	}
	// Style revisions are not create-phrase asks; they route via ActionArtifact + open canvas.
	if wantsMermaidCanvas("lets update the canvas to be black and white") {
		t.Fatal("style update must not match create-phrase mermaid gate")
	}
	if wantsMermaidCanvas("Create a Neural Canvas markdown report summarizing this workspace.") {
		t.Fatal("markdown report must not match mermaid create")
	}
	if wantsMermaidCanvas("Render a markdown canvas with architecture notes and open questions.") {
		t.Fatal("markdown + architecture notes must not match mermaid create")
	}
	if wantsMermaidCanvas("create a new canvas") {
		t.Fatal("generic new canvas defaults to markdown, not mermaid")
	}
}

func TestWantsMarkdownCanvas(t *testing.T) {
	if !wantsMarkdownCanvas("Create a Neural Canvas markdown report summarizing this workspace.") {
		t.Fatal("expected markdown canvas ask")
	}
	if !wantsMarkdownCanvas("Render a markdown canvas with architecture notes and open questions.") {
		t.Fatal("expected markdown canvas ask")
	}
	if !wantsMarkdownCanvas("create a new canvas") {
		t.Fatal("generic new canvas should be markdown")
	}
	if wantsMarkdownCanvas("Create a Neural Canvas Mermaid diagram of this architecture") {
		t.Fatal("mermaid ask is not markdown")
	}
}

func TestMermaidShortcutSkipsMarkdownAskWithPriorMermaid(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := ai.NewMockProvider()
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)

	created, err := store.Create(artifacts.Artifact{
		Kind:  "mermaid",
		Title: "dickory-docs architecture",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.mermaid", APIVersion: "1", MediaType: "text/vnd.mermaid",
		},
		Payload: json.RawMessage(`"flowchart TD\n  User --> App"`),
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := protocol.NewMessage(protocol.MessageTypeArtifactChanged, "dm-camron-assistant", a.Info, created.Title)
	changed.SetArtifactReference(protocol.ArtifactReference{
		ID: created.ID, Title: created.Title, RendererID: "nj.mermaid",
		MediaType: "text/vnd.mermaid", Revision: int64(created.Revision), Action: "created",
	})
	a.replaceChannelHistory("dm-camron-assistant", []*protocol.Message{changed})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"Create a Neural Canvas markdown report summarizing this workspace.",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "dickory-docs",
			"file_tree":      "src\ndocs\n",
		},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		Retrieval:       []intent.RetrievalTarget{intent.RetrievalPriorReference},
		PolicyOverrides: []string{"open_canvas_artifact"},
	}); err != nil {
		t.Fatal(err)
	}

	if _, ok := a.tryNeuralCanvasMermaidShortcut(context.Background(), msg, "", provider); ok {
		t.Fatal("mermaid shortcut must not hijack markdown report ask")
	}
	resp, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider)
	if !ok {
		t.Fatal("expected markdown shortcut")
	}
	if !strings.Contains(resp, "markdown report") {
		t.Fatalf("resp=%q", resp)
	}
	items, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant"})
	if err != nil {
		t.Fatal(err)
	}
	var sawMarkdown bool
	for _, item := range items {
		if item.Renderer.ID == "nj.markdown" {
			sawMarkdown = true
		}
	}
	if !sawMarkdown {
		t.Fatalf("expected nj.markdown artifact, got %+v", items)
	}
	still, err := store.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if still.Revision != created.Revision {
		t.Fatalf("mermaid revision changed to %d; markdown ask must not update it", still.Revision)
	}
}

func TestTryNeuralCanvasMarkdownShortcutCreatesArtifact(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := ai.NewMockProvider()
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"create a new canvas",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "dickory-docs",
			"file_tree":      "src\ndocs\n",
		},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}

	resp, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider)
	if !ok {
		t.Fatal("expected markdown shortcut for create a new canvas")
	}
	if !strings.Contains(resp, "blank Neural Canvas") {
		t.Fatalf("resp=%q, want blank canvas confirmation", resp)
	}
	items, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant", RendererID: "nj.markdown"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	var payload string
	if err := json.Unmarshal(items[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(payload), "# Canvas") {
		t.Fatalf("blank canvas payload=%q", payload)
	}
	if strings.Contains(payload, "Workspace-grounded") || strings.Contains(payload, "mock response") {
		t.Fatalf("generic create must not auto-generate a workspace report; got=%q", payload)
	}
}

func TestTryNeuralCanvasMarkdownShortcutUsesPriorReferenceBody(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := ai.NewMockProvider()
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)

	prior := strings.Repeat("Overall Architecture: React frontend with Tauri backend for dickory-docs. ", 12) +
		"### Code Quality\n\nClear separation of concerns between desktop UI and Rust sidecar.\n"
	priorMsg := protocol.NewMessage(protocol.MessageTypeAnswer, "dm-camron-assistant", a.Info, prior)
	a.replaceChannelHistory("dm-camron-assistant", []*protocol.Message{priorMsg})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"can you create a canvas with that summary",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "dickory-docs",
			"file_tree":      "src\ndocs\n",
		},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation:        intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		Retrieval: []intent.RetrievalTarget{intent.RetrievalPriorReference, intent.RetrievalCodebase},
	}); err != nil {
		t.Fatal(err)
	}

	resp, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider)
	if !ok {
		t.Fatal("expected markdown shortcut")
	}
	if !strings.Contains(resp, "markdown report") {
		t.Fatalf("resp=%q", resp)
	}
	items, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant", RendererID: "nj.markdown"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	var payload string
	if err := json.Unmarshal(items[0].Payload, &payload); err != nil {
		t.Fatalf("payload unmarshal: %v raw=%s", err, string(items[0].Payload))
	}
	if !strings.Contains(payload, "React frontend with Tauri backend for dickory-docs") {
		t.Fatalf("payload missing prior summary; got=%q", payload)
	}
	if strings.Contains(payload, "mock response") {
		t.Fatalf("prior_reference canvas must not regenerate via mock LLM; got=%q", payload)
	}
	if strings.Contains(strings.ToLower(payload), "neural network") {
		t.Fatalf("payload should not invent Neural Canvas product fiction; got=%q", payload)
	}
}

func TestTryNeuralCanvasMarkdownShortcutWorkspaceReport(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := ai.NewMockProvider()
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"create a canvas with a report about this project",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "dickory-docs",
			"file_tree":      "src\ndocs\n",
		},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"workspace_report"},
	}); err != nil {
		t.Fatal(err)
	}

	resp, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider)
	if !ok {
		t.Fatal("expected markdown shortcut for workspace report")
	}
	if !strings.Contains(resp, "markdown report") {
		t.Fatalf("resp=%q", resp)
	}
	items, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant", RendererID: "nj.markdown"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	var payload string
	_ = json.Unmarshal(items[0].Payload, &payload)
	if !strings.Contains(payload, "dickory-docs") && !strings.Contains(payload, "mock response") {
		t.Fatalf("workspace report payload unexpected: %q", payload)
	}
}

type revisionMarkdownProvider struct {
	ai.MockProvider
	revision string
}

func (p *revisionMarkdownProvider) GenerateResponse(ctx context.Context, prompt string, history []protocol.Message) (string, error) {
	if strings.Contains(prompt, "Revise the Markdown document") {
		if p.revision != "" {
			return p.revision, nil
		}
		return "# Trip plan\n\n## Places\n\n- Tokyo\n- Kyoto\n", nil
	}
	return p.MockProvider.GenerateResponse(ctx, prompt, history)
}

func TestTryNeuralCanvasMarkdownUpdateShortcutFillIn(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := &revisionMarkdownProvider{}
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)

	created, err := store.Create(artifacts.Artifact{
		Kind:  "markdown",
		Title: "Canvas",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.markdown", APIVersion: "1", MediaType: "text/markdown",
		},
		Payload: mustJSONString("# Canvas\n\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := protocol.NewMessage(protocol.MessageTypeArtifactChanged, "dm-camron-assistant", a.Info, created.Title)
	changed.SetArtifactReference(protocol.ArtifactReference{
		ID: created.ID, Title: created.Title, RendererID: "nj.markdown",
		MediaType: "text/markdown", Revision: int64(created.Revision), Action: "created",
	})
	a.replaceChannelHistory("dm-camron-assistant", []*protocol.Message{changed})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"add a list of places we are going to visit",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		Retrieval: []intent.RetrievalTarget{intent.RetrievalPriorReference},
	}); err != nil {
		t.Fatal(err)
	}

	resp, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider)
	if !ok {
		t.Fatal("expected markdown update shortcut")
	}
	if !strings.Contains(resp, "Updated the Neural Canvas") {
		t.Fatalf("resp=%q", resp)
	}
	got, err := store.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 2 {
		t.Fatalf("revision=%d, want 2", got.Revision)
	}
	var payload string
	_ = json.Unmarshal(got.Payload, &payload)
	if !strings.Contains(payload, "Tokyo") {
		t.Fatalf("payload missing places; got=%q", payload)
	}
	items, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant", RendererID: "nj.markdown"})
	if err != nil || len(items) != 1 {
		t.Fatalf("expected one markdown artifact, got %+v err=%v", items, err)
	}
}

func TestTryNeuralCanvasMarkdownUpdateEmbedsMermaidNotSeparateArtifact(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := &revisionMarkdownProvider{
		revision: "# Trip plan\n\n```mermaid\nflowchart LR\n  A[Tokyo] --> B[Kyoto]\n```\n",
	}
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)

	created, err := store.Create(artifacts.Artifact{
		Kind:  "markdown",
		Title: "Canvas",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.markdown", APIVersion: "1", MediaType: "text/markdown",
		},
		Payload: mustJSONString("# Trip plan\n\n## Places\n\n- Tokyo\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := protocol.NewMessage(protocol.MessageTypeArtifactChanged, "dm-camron-assistant", a.Info, created.Title)
	changed.SetArtifactReference(protocol.ArtifactReference{
		ID: created.ID, Title: created.Title, RendererID: "nj.markdown",
		MediaType: "text/markdown", Revision: int64(created.Revision), Action: "created",
	})
	a.replaceChannelHistory("dm-camron-assistant", []*protocol.Message{changed})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"add a Mermaid itinerary diagram",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}

	if _, ok := a.tryNeuralCanvasMermaidShortcut(context.Background(), msg, "", provider); ok {
		t.Fatal("mermaid shortcut must not spawn nj.mermaid when markdown page is open")
	}
	resp, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider)
	if !ok {
		t.Fatal("expected markdown update for diagram embed")
	}
	if !strings.Contains(resp, "Updated the Neural Canvas") {
		t.Fatalf("resp=%q", resp)
	}
	mermaidItems, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant", RendererID: "nj.mermaid"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mermaidItems) != 0 {
		t.Fatalf("expected no nj.mermaid artifacts, got %+v", mermaidItems)
	}
	got, err := store.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	var payload string
	_ = json.Unmarshal(got.Payload, &payload)
	if !strings.Contains(payload, "```mermaid") {
		t.Fatalf("expected mermaid fence on markdown page; got=%q", payload)
	}
}

func mustJSONString(s string) json.RawMessage {
	data, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return data
}

func TestTryNeuralCanvasMarkdownShortcutDoesNotRecreateOnStatusQuestion(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := ai.NewMockProvider()
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)

	created, err := store.Create(artifacts.Artifact{
		Kind:  "markdown",
		Title: "Canvas",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.markdown", APIVersion: "1", MediaType: "text/markdown",
		},
		Payload: mustJSONString("# Canvas\n\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := protocol.NewMessage(protocol.MessageTypeArtifactChanged, "dm-camron-assistant", a.Info, created.Title)
	changed.SetArtifactReference(protocol.ArtifactReference{
		ID: created.ID, Title: created.Title, RendererID: "nj.markdown",
		MediaType: "text/markdown", Revision: int64(created.Revision), Action: "created",
	})
	a.replaceChannelHistory("dm-camron-assistant", []*protocol.Message{changed})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"did you update the canvas with the info?",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"blank_canvas"},
		Retrieval:   []intent.RetrievalTarget{intent.RetrievalPriorReference},
	}); err != nil {
		t.Fatal(err)
	}

	if _, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider); ok {
		t.Fatal("status question must not create or revise via markdown shortcut")
	}
	items, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant", RendererID: "nj.markdown"})
	if err != nil || len(items) != 1 {
		t.Fatalf("expected one markdown artifact, got %+v err=%v", items, err)
	}
	if items[0].Revision != 1 {
		t.Fatalf("revision=%d, want unchanged 1", items[0].Revision)
	}
}

func TestTryNeuralCanvasMarkdownUpdateShortcutWeatherFill(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := &revisionMarkdownProvider{
		revision: "# Weather Forecast\n\n## Today\n\n- 72°F and sunny\n",
	}
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)

	created, err := store.Create(artifacts.Artifact{
		Kind:  "markdown",
		Title: "Canvas",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.markdown", APIVersion: "1", MediaType: "text/markdown",
		},
		Payload: mustJSONString("# Canvas\n\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := protocol.NewMessage(protocol.MessageTypeArtifactChanged, "dm-camron-assistant", a.Info, created.Title)
	changed.SetArtifactReference(protocol.ArtifactReference{
		ID: created.ID, Title: created.Title, RendererID: "nj.markdown",
		MediaType: "text/markdown", Revision: int64(created.Revision), Action: "created",
	})
	a.replaceChannelHistory("dm-camron-assistant", []*protocol.Message{changed})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"Can you get todays weather for St. Louis, MO and put it in the canvas please",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}

	resp, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider)
	if !ok {
		t.Fatal("expected markdown update for weather fill-in")
	}
	if strings.Contains(resp, "Opened a blank") {
		t.Fatalf("must update existing canvas, not create blank: %q", resp)
	}
	if !strings.Contains(resp, "Updated the Neural Canvas") {
		t.Fatalf("resp=%q", resp)
	}
	got, err := store.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 2 {
		t.Fatalf("revision=%d, want 2", got.Revision)
	}
	if got.Title == "Weather Forecast" {
		t.Fatalf("title=%q must come from user ask, not invented H1", got.Title)
	}
	if !strings.Contains(strings.ToLower(got.Title), "weather") ||
		!strings.Contains(strings.ToLower(got.Title), "louis") {
		t.Fatalf("title=%q, want weather + St Louis from ask", got.Title)
	}
	items, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant", RendererID: "nj.markdown"})
	if err != nil || len(items) != 1 {
		t.Fatalf("expected one artifact, got %+v", items)
	}
}

type revisionMermaidProvider struct {
	ai.MockProvider
}

func (p *revisionMermaidProvider) GenerateResponse(ctx context.Context, prompt string, history []protocol.Message) (string, error) {
	if strings.Contains(prompt, "Revise the Mermaid diagram") {
		return "%%{init: {'theme':'base','themeVariables':{'primaryTextColor':'#000000','lineColor':'#000000','mainBkg':'#ffffff','nodeBorder':'#000000'}}}%%\nflowchart TD\n  User --> App", nil
	}
	return p.MockProvider.GenerateResponse(ctx, prompt, history)
}

func TestTryNeuralCanvasMermaidUpdateShortcutStructural(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := &revisionMermaidProvider{}
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)

	created, err := store.Create(artifacts.Artifact{
		Kind:  "mermaid",
		Title: "dickory-docs architecture",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.mermaid", APIVersion: "1", MediaType: "text/vnd.mermaid",
		},
		Payload: json.RawMessage(`"flowchart TD\n  User --> App"`),
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := protocol.NewMessage(protocol.MessageTypeArtifactChanged, "dm-camron-assistant", a.Info, created.Title)
	changed.SetArtifactReference(protocol.ArtifactReference{
		ID: created.ID, Title: created.Title, RendererID: "nj.mermaid",
		MediaType: "text/vnd.mermaid", Revision: int64(created.Revision), Action: "created",
	})
	a.replaceChannelHistory("dm-camron-assistant", []*protocol.Message{changed})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"lets update the canvas to be black and white",
	)
	// Semantic/open-canvas policy stamps ActionArtifact — not phrase matching.
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		Retrieval:       []intent.RetrievalTarget{intent.RetrievalPriorReference},
		PolicyOverrides: []string{"open_canvas_artifact"},
	}); err != nil {
		t.Fatal(err)
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact {
		t.Fatalf("goal=%+v, want artifact", goal)
	}
	if userRequestsImplementationForMessage(a, msg) {
		t.Fatal("artifact turn must not force FILE_CHANGE")
	}

	ledger := &ActionEvidenceLedger{}
	ctx := contextWithActionEvidence(context.Background(), ledger)
	resp, ok := a.tryNeuralCanvasMermaidShortcut(ctx, msg, "", provider)
	if !ok {
		t.Fatal("expected mermaid update shortcut to fire on ActionArtifact + open mermaid")
	}
	if !strings.Contains(resp, "Updated the Neural Canvas Mermaid") {
		t.Fatalf("resp=%q", resp)
	}
	if !ledger.Has(EvidenceArtifactCreated) {
		t.Fatalf("missing artifact evidence: %+v", ledger.Entries())
	}
	got, err := store.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 2 {
		t.Fatalf("revision=%d, want 2", got.Revision)
	}
	payload := mermaidSourceFromPayload(got.Payload)
	if !strings.Contains(payload, "#000000") || !strings.Contains(payload, "flowchart TD") {
		t.Fatalf("payload=%q, want revised monochrome mermaid", payload)
	}
}

func TestTryNeuralCanvasMermaidShortcutCreatesArtifact(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	provider := ai.NewMockProvider()
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"Create a Neural Canvas Mermaid diagram of this architecture",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "dickory-docs",
			"file_tree":      "src\nsrc-tauri\nscripts\ndocs\nassets\npublic\n",
		},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"durable_artifact"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = deriveTurnGoal(a, msg, IntentTask)

	ledger := &ActionEvidenceLedger{}
	ctx := contextWithActionEvidence(context.Background(), ledger)
	resp, ok := a.tryNeuralCanvasMermaidShortcut(ctx, msg, "workspace context here", provider)
	if !ok {
		t.Fatal("expected mermaid shortcut to fire")
	}
	if !strings.Contains(resp, "Posted a Neural Canvas Mermaid") {
		t.Fatalf("resp=%q", resp)
	}
	if !ledger.Has(EvidenceArtifactCreated) {
		t.Fatalf("missing artifact evidence: %+v", ledger.Entries())
	}
	items, err := store.List(artifacts.Filter{ChannelID: "dm-camron-assistant"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].Renderer.ID != "nj.mermaid" {
		t.Fatalf("renderer=%+v", items[0].Renderer)
	}
	payload := mermaidSourceFromPayload(items[0].Payload)
	if !strings.Contains(payload, "flowchart") {
		t.Fatalf("payload=%s, want mermaid flowchart", payload)
	}
	if !strings.Contains(payload, "src") || !strings.Contains(payload, "src-tauri") {
		t.Fatalf("tree-first payload=%s, want workspace dirs", payload)
	}
	if looksLikeMetaCanvasProcessMermaid(payload) || strings.Contains(strings.ToLower(payload), "neural canvas") {
		t.Fatalf("shortcut shipped meta canvas process diagram: %s", payload)
	}
}

func TestExtractMermaidSource(t *testing.T) {
	got := extractMermaidSource("```mermaid\nflowchart TD\n  A-->B\n```")
	if !strings.Contains(got, "flowchart TD") || strings.Contains(got, "```") {
		t.Fatalf("got=%q", got)
	}
	if sanitizeMermaidSource("not a diagram") != "" {
		t.Fatal("non-mermaid prose must be rejected")
	}
}

func TestLooksLikeMetaCanvasProcessMermaid(t *testing.T) {
	meta := `graph TD
    A[User Request] --> B{Process User Request}
    B -->|User Request: Create Mermaid Diagram| C[Neural Canvas]
    C --> D{Generate Mermaid Code}`
	if !looksLikeMetaCanvasProcessMermaid(meta) {
		t.Fatal("expected meta canvas-process diagram to be rejected")
	}
	bleed := `flowchart TD
  App[App.tsx] --> UI[Desktop UI]
  UI --> NC[Neural Canvas]
  NC --> CA[Create Artifact]
  CA --> MV[Mermaid Diagram Viewer]`
	if !looksLikeMetaCanvasProcessMermaid(bleed) {
		t.Fatal("Neural Canvas / Create Artifact bleed must be rejected")
	}
	good := `flowchart TD
  User --> Desktop[dickory-docs UI]
  Desktop --> Tauri[src-tauri]
  Desktop --> Vite[src / Vite]
  Tauri --> Backend[App backend]`
	if looksLikeMetaCanvasProcessMermaid(good) {
		t.Fatal("workspace architecture diagram must not be rejected")
	}
}

func TestMermaidMentionsWorkspaceNodes(t *testing.T) {
	src := `flowchart TD
  App --> A[Markdown Preview]
  App --> B[File Explorer]`
	if mermaidMentionsWorkspaceNodes(src, []string{"src", "src-tauri", "scripts"}) {
		t.Fatal("generic UI labels without workspace dirs must fail grounding")
	}
	grounded := `flowchart TD
  App --> A[src]
  App --> B[src-tauri]
  A --> C[components]`
	if !mermaidMentionsWorkspaceNodes(grounded, []string{"src", "src-tauri", "scripts"}) {
		t.Fatal("diagram with real dirs must pass grounding")
	}
}

func TestQuoteUnsafeMermaidPathLabels(t *testing.T) {
	raw := `flowchart TD
  A[packages/package-lock.json] --> B[public/assets]
  C[src] --> D[ok]`
	got := sanitizeMermaidSource(raw)
	if !strings.Contains(got, `A["packages/package-lock.json"]`) {
		t.Fatalf("got=%q, want quoted path label", got)
	}
	if !strings.Contains(got, `B["public/assets"]`) {
		t.Fatalf("got=%q, want quoted public path", got)
	}
	if !strings.Contains(got, `D[ok]`) {
		t.Fatalf("got=%q, want simple label left unquoted", got)
	}
}

func TestLooksLikeInvalidMermaidArchitecture(t *testing.T) {
	bad := `graph LR
    subgraph Tauri
        participant "Tauri App"
    end`
	if !looksLikeInvalidMermaidArchitecture(bad) {
		t.Fatal("flowchart+participant mix must be rejected")
	}
	good := `flowchart TD
  User --> App[dickory-docs]
  App --> Tauri[src-tauri]`
	if looksLikeInvalidMermaidArchitecture(good) {
		t.Fatal("valid flowchart must be accepted")
	}
}

func TestFallbackMermaidFromWorkspaceUsesTree(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{}, "Create a Neural Canvas Mermaid diagram of this architecture")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "dickory-docs",
			"file_tree":      "src\nsrc-tauri\ndocs\npackage.json\n",
		},
	}
	got := fallbackMermaidFromWorkspace(msg, "")
	if !strings.Contains(got, "dickory-docs") || !strings.Contains(got, "src") {
		t.Fatalf("got=%q", got)
	}
}

func TestFindRecentMarkdownArtifactIgnoresStoreWithoutHistory(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), hub)

	if _, err := store.Create(artifacts.Artifact{
		Kind:  "markdown",
		Title: "Weather Forecast",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.markdown", APIVersion: "1", MediaType: "text/markdown",
		},
		Payload: mustJSONString("# Weather Forecast\n\n- sunny\n"),
	}); err != nil {
		t.Fatal(err)
	}
	// Cleared conversation: no channel history, no open_artifact metadata.
	a.replaceChannelHistory("dm-camron-assistant", nil)

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"create a new canvas",
	)
	if got := a.findRecentMarkdownArtifact(msg); got != nil {
		t.Fatalf("after clear-history must not revive store canvas %q", got.Title)
	}
}

func TestFindRecentMarkdownArtifactUsesOpenArtifactMetadata(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), hub)
	a.replaceChannelHistory("dm-camron-assistant", nil)

	created, err := store.Create(artifacts.Artifact{
		Kind:  "markdown",
		Title: "Weather Forecast",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.markdown", APIVersion: "1", MediaType: "text/markdown",
		},
		Payload: mustJSONString("# Weather Forecast\n\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"why did you name it weather forcast?",
	)
	msg.Metadata = map[string]interface{}{
		"open_artifact": map[string]interface{}{
			"id": created.ID, "title": created.Title, "renderer_id": "nj.markdown",
		},
	}
	got := a.findRecentMarkdownArtifact(msg)
	if got == nil || got.ID != created.ID {
		t.Fatalf("expected open_artifact metadata canvas, got %+v", got)
	}
	resp, ok := a.tryOpenCanvasMetaAnswer(msg)
	if !ok {
		t.Fatal("expected meta title answer")
	}
	if !strings.Contains(resp, "Weather Forecast") || !strings.Contains(resp, "clearing chat history") {
		t.Fatalf("resp=%q", resp)
	}
	if strings.Contains(strings.ToLower(resp), "national weather service") {
		t.Fatalf("must not digress into meteorology lecture: %q", resp)
	}
}

func TestResolveMarkdownCanvasUpdateTitleFromUserAsk(t *testing.T) {
	got := resolveMarkdownCanvasUpdateTitle("Canvas",
		"Can you get todays weather for St. Louis, MO and put it in the canvas please",
		"# Weather Forecast\n\n- sunny\n")
	if got == "Weather Forecast" {
		t.Fatal("must not adopt invented H1 absent from user ask")
	}
	if !strings.Contains(strings.ToLower(got), "weather") || !strings.Contains(strings.ToLower(got), "louis") {
		t.Fatalf("title=%q, want weather + St Louis from ask", got)
	}
	if !titleGroundedInUserAsk("St. Louis weather", "weather for St. Louis, MO") {
		t.Fatal("expected grounded title")
	}
	if titleGroundedInUserAsk("Weather Forecast", "add a grocery list to the canvas") {
		t.Fatal("weather forecast must not ground in grocery ask")
	}
}

func TestMarkdownCanvasCreateKindPrefersPriorContentOverBlankReason(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"add that information to a neural canvas",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"blank_canvas"},
	}); err != nil {
		t.Fatal(err)
	}
	if got := markdownCanvasCreateKind(msg); got != "prior_reference" {
		t.Fatalf("kind=%q, want prior_reference despite blank_canvas reason", got)
	}
}

func TestMarkdownBodyFromPriorAssistantContentStripsCanvasChatter(t *testing.T) {
	prior := "Certainly! Below is a canvas with the information from the most recent Phoenix Team Meeting:\n\n```\n===========================================\nPhoenix Team Meeting - Jul 21, 2026\n===========================================\n\n**Attendees:**\n- Camron Wood\n```\n\nFeel free to use this canvas as a reference."
	body := markdownBodyFromPriorAssistantContent(prior)
	if !strings.Contains(body, "Phoenix Team Meeting") {
		t.Fatalf("missing meeting title: %q", body)
	}
	if strings.Contains(strings.ToLower(body), "certainly") || strings.Contains(strings.ToLower(body), "feel free") {
		t.Fatalf("chatter not stripped: %q", body)
	}
	if looksLikeSpuriousCanvasJSONPayload(body) {
		t.Fatal("meeting markdown must not look like JSON metadata")
	}
}

func TestLooksLikeSpuriousCanvasJSONPayload(t *testing.T) {
	junk := `{
"description": "Neural Canvas for Phoenix Team Meeting Notes",
"fallback": "Markdown summary",
"kind": "report",
"media_type": "text/markdown",
"renderer_id": "nj.markdown",
"title": "Phoenix Team Meeting Notes",
"workspace_id": "your_workspace_id"
}`
	if !looksLikeSpuriousCanvasJSONPayload(junk) {
		t.Fatal("expected JSON metadata payload to be rejected")
	}
	if looksLikeSpuriousCanvasJSONPayload("# Phoenix Team Meeting\n\n- notes\n") {
		t.Fatal("real markdown must not be rejected")
	}
}

func TestTryNeuralCanvasMarkdownUpdateUsesPriorContentNotJSON(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	// Model that returns the live-failure JSON metadata blob.
	provider := &jsonJunkMarkdownProvider{}
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, provider, hub)

	created, err := store.Create(artifacts.Artifact{
		Kind:  "markdown",
		Title: "Fill Summary Of My Most Recent",
		Links: artifacts.ArtifactLinks{ChannelID: "dm-camron-assistant"},
		Renderer: artifacts.Renderer{
			ID: "nj.markdown", APIVersion: "1", MediaType: "text/markdown",
		},
		Payload: mustJSONString("# Fill Summary Of My Most Recent\n\n## Agenda\n- Project X\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := protocol.NewMessage(protocol.MessageTypeArtifactChanged, "dm-camron-assistant", a.Info, created.Title)
	changed.SetArtifactReference(protocol.ArtifactReference{
		ID: created.ID, Title: created.Title, RendererID: "nj.markdown",
		MediaType: "text/markdown", Revision: int64(created.Revision), Action: "created",
	})

	prior := protocol.NewMessage(protocol.MessageTypeChat, "dm-camron-assistant", a.Info,
		"Certainly! Below is a canvas with the information:\n\n```\nPhoenix Team Meeting - Jul 21, 2026\n\n**Attendees:**\n- Camron Wood\n- Josh Peck\n\n**Summary:**\n- QC reader deployment\n```\n")
	a.replaceChannelHistory("dm-camron-assistant", []*protocol.Message{prior, changed})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"add that information to a neural canvas",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"blank_canvas"},
		Retrieval:   []intent.RetrievalTarget{intent.RetrievalPriorReference},
	}); err != nil {
		t.Fatal(err)
	}

	resp, ok := a.tryNeuralCanvasMarkdownShortcut(context.Background(), msg, "", provider)
	if !ok {
		t.Fatal("expected markdown update shortcut")
	}
	if !strings.Contains(resp, "Updated the Neural Canvas") {
		t.Fatalf("resp=%q", resp)
	}
	got, err := store.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	var payload string
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("payload must be markdown string: %v raw=%s", err, string(got.Payload))
	}
	if looksLikeSpuriousCanvasJSONPayload(payload) {
		t.Fatalf("payload still JSON metadata: %q", payload)
	}
	if !strings.Contains(payload, "Phoenix Team Meeting") || !strings.Contains(payload, "Camron Wood") {
		t.Fatalf("payload missing prior meeting content: %q", payload)
	}
	if strings.Contains(payload, "Project X") {
		t.Fatalf("must not keep invented Project X body: %q", payload)
	}
	if got.Title == "Fill Summary Of My Most Recent" {
		t.Fatalf("title should come from prior meeting content, got %q", got.Title)
	}
}

type jsonJunkMarkdownProvider struct {
	ai.MockProvider
}

func (p *jsonJunkMarkdownProvider) GenerateResponse(ctx context.Context, prompt string, history []protocol.Message) (string, error) {
	return `{
"description": "Neural Canvas for Phoenix Team Meeting Notes",
"kind": "report",
"media_type": "text/markdown",
"renderer_id": "nj.markdown",
"title": "Phoenix Team Meeting Notes",
"workspace_id": "your_workspace_id"
}`, nil
}
