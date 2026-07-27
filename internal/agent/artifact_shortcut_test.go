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
		RequestedAction: intent.ActionRun, Action: intent.ActionRun,
		Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
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
