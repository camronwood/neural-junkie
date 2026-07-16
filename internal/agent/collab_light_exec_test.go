package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollabLightFiltersUnrelatedSourcesForSchemaTasks(t *testing.T) {
	task := collaboration.CollaborationTask{
		Title:       "Schema registration",
		Description: "Write collabs/x/resource-api-schema-registration.md focusing on resource-api/json_endpoints",
	}
	sources := map[string]string{
		"core/sample/main.go":          "package main\nfunc HelloWorld()",
		"resource-api/json_endpoints/a": "endpoint schema",
	}
	got := filterLightSourcesToTaskFocus(task, sources)
	if _, ok := got["core/sample/main.go"]; ok {
		t.Fatal("sample HelloWorld must not ground schema deliverables")
	}
	if _, ok := got["resource-api/json_endpoints/a"]; !ok {
		t.Fatal("expected resource-api source kept")
	}
}

func TestAcceptCollabLightRewrite_RejectsPlaceholderAndUngrounded(t *testing.T) {
	sources := map[string]string{
		"README.md":           "# Minimal Repo\nThis is the fixture readme for findings.",
		"core/sample/main.go": "package main\n\nfunc HelloWorld() {}\n",
	}
	if acceptCollabLightRewrite("findings.md", "# App Name\n\n## Features\n\n- Feature 1\n", sources) {
		t.Fatal("placeholder rewrite must be rejected")
	}
	if acceptCollabLightRewrite("findings.md", "# Findings\n\nReact HelloWorld App.tsx dump with no repo cites.", sources) {
		t.Fatal("ungrounded fiction must be rejected when sources exist")
	}
	if !acceptCollabLightRewrite("findings.md", "# Findings\n\nFrom `README.md` and `core/sample/main.go`: fixture readme for findings.\n", sources) {
		t.Fatal("grounded rewrite citing sources must be accepted")
	}
	if acceptCollabLightRewrite("findings.md", "# findings\n\n_Initial stub created when the plan was approved. Replace with task output._\n", sources) {
		t.Fatal("stub marker body must be rejected")
	}
}

func TestCollabLightReadSources_SkipsDeliverableStub(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "collabs", "x"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Real README\nGround me.\n"), 0o644)
	stub := "# findings.md\n\n_Initial stub created when the plan was approved. Replace with task output._\n"
	_ = os.WriteFile(filepath.Join(dir, "collabs", "x", "findings.md"), []byte(stub), 0o644)

	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", protocol.AgentInfo{}, "body")
	msg.Metadata = map[string]interface{}{
		"task_title":         "Write findings",
		"task_description":   "Write collabs/x/findings.md summarizing README.md",
		"task_context_paths": []string{"README.md", "collabs/x/findings.md"},
	}
	got := collabLightReadSources(dir, msg)
	if _, ok := got["collabs/x/findings.md"]; ok {
		t.Fatal("deliverable stub must not be a light-exec source")
	}
	if _, ok := got["README.md"]; !ok {
		t.Fatal("expected README source kept")
	}
}

func TestCollabTaskPrefersLightExecution_Markdown(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", protocol.AgentInfo{}, "body")
	msg.Metadata = map[string]interface{}{
		"task_title":       "Write findings",
		"task_description": "Write collabs/x/findings.md with three bullets",
	}
	if !collabTaskPrefersLightExecution(msg) {
		t.Fatal("expected markdown deliverable to prefer light execution")
	}
}

func TestCollabTaskPrefersLightExecution_DeliverableKindMetadata(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", protocol.AgentInfo{}, "Create foo.go somehow")
	msg.Metadata = map[string]interface{}{
		"deliverable_kind": "markdown",
		"task_title":       "Write doc",
		"task_description": "Write collabs/x/notes.md",
	}
	if !collabTaskPrefersLightExecution(msg) {
		t.Fatal("deliverable_kind=markdown must prefer light exec without body scraping")
	}
	msg.Metadata["deliverable_kind"] = "file"
	if collabTaskPrefersLightExecution(msg) {
		t.Fatal("deliverable_kind=file must not prefer light exec")
	}
}

func TestCollabTaskPrefersLightExecution_Coding(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", protocol.AgentInfo{}, "body")
	msg.Metadata = map[string]interface{}{
		"task_title":       "Implement handler",
		"task_description": "Create cmd/server/foo.go with HTTP handler",
	}
	// Has write+go file — still a file deliverable; markdown-only check should be false
	// unless only .md. Mixed: TaskLooksLikeMarkdownDeliverable false if no .md
	if collabTaskPrefersLightExecution(msg) {
		t.Fatal("coding .go task should not prefer light markdown path")
	}
}

func TestCollabLightMarkdownAbortsWhenCancelled(t *testing.T) {
	dir := t.TempDir()
	hub := &lightExecCaptureHub{}
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), hub)
	a.WorkspacePath = dir
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-x", a.Info, "body")
	msg.Metadata = map[string]interface{}{
		"deliverable_kind": "markdown",
		"task_title":       "Write findings",
		"task_description": "Write collabs/x/findings.md summarizing results",
	}
	msg.SetCollaborationID("collab-1")
	msg.SetCollaborationPhase("cancelled")
	msg.SetTaskID("t1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, proposed, files, err := a.runCollabLightMarkdownExecution(ctx, msg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if proposed || len(files) > 0 {
		t.Fatalf("cancelled light exec must not propose, proposed=%v files=%v out=%q", proposed, files, out)
	}
	if hub.proposals != 0 {
		t.Fatal("expected no file change proposal after cancel")
	}
}

func TestCollabLightMarkdownEligibleAndRun(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "docs", "README.md"), []byte("# Sample fixture readme\nHelloWorld\n"), 0o644)

	hub := &lightExecCaptureHub{}
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), hub)
	a.WorkspacePath = dir
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-x", a.Info, "body")
	msg.Metadata = map[string]interface{}{
		"deliverable_kind":   "markdown",
		"task_title":         "Write findings",
		"task_description":   "Write collabs/x/findings.md summarizing results",
		"task_context_paths": []string{"docs/README.md"},
	}
	msg.SetCollaborationID("collab-1")
	msg.SetCollaborationPhase("executing")
	msg.SetTaskID("t1")
	if !collabLightMarkdownEligible(msg) {
		t.Fatal("expected light markdown eligible")
	}
	codeMsg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", a.Info, "body")
	codeMsg.Metadata = map[string]interface{}{
		"deliverable_kind": "file",
		"task_title":       "Impl",
		"task_description": "Create foo.go",
	}
	if collabLightMarkdownEligible(codeMsg) {
		t.Fatal("coding file deliverable must not use light markdown path")
	}

	out, proposed, files, err := a.runCollabLightMarkdownExecution(context.Background(), msg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !proposed {
		t.Fatal("expected proposal")
	}
	if len(files) != 1 || files[0] != "collabs/x/findings.md" {
		t.Fatalf("files=%v", files)
	}
	if !strings.Contains(out, "TASK_STATUS: completed") {
		t.Fatalf("expected TASK_STATUS in %q", out)
	}
	if hub.proposals == 0 {
		t.Fatal("expected file change proposal message")
	}
}

type lightExecCaptureHub struct {
	shouldRespondTestHub
	proposals int
}

func (h *lightExecCaptureHub) SendMessage(msg *protocol.Message) error {
	if msg != nil && msg.Type == protocol.MessageTypeFileChange {
		h.proposals++
	}
	return nil
}
