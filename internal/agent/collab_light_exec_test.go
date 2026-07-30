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

func TestSynthesizeWebsiteDesignSystemMarkdown(t *testing.T) {
	task := collaboration.CollaborationTask{
		Title:       "Design system",
		Description: "Write collabs/x/design-system.md with color palette black white gray blue red, typography, spacing",
	}
	got := synthesizeWebsiteDesignSystemMarkdown(task)
	if !strings.Contains(got, "Color Palette") || !strings.Contains(got, "black") {
		t.Fatalf("expected design system seed, got %q", got)
	}
	body := buildCollabLightMarkdownBody(task, nil, "collabs/x/design-system.md")
	if strings.Contains(body, "No allowlisted") {
		t.Fatalf("design-system task should synthesize without empty stub: %q", body)
	}
	site := collaboration.CollaborationTask{
		Title:       "Site structure",
		Description: "Write collabs/x/site-structure.md with navigation and page hierarchy",
	}
	siteBody := synthesizeWebsiteMarkdownDeliverable(site)
	if !strings.Contains(siteBody, "Navigation") || !strings.Contains(siteBody, "Hierarchy") {
		t.Fatalf("expected site-structure seed, got %q", siteBody)
	}
}

func TestSynthesizeResourceAPIMarkdownDeliverable(t *testing.T) {
	scope := collaboration.CollaborationTask{
		Title:       "Define Scope",
		Description: "Write collabs/<id>/scope.md for resource API schema standardization",
	}
	got := synthesizeResourceAPIMarkdownDeliverable(scope, "collabs/x/scope.md")
	if !strings.Contains(got, "schema") || !strings.Contains(got, "resource API") {
		t.Fatalf("expected scope seed with schema keywords, got %q", got)
	}
	body := buildCollabLightMarkdownBody(scope, nil, "collabs/x/scope.md")
	if strings.Contains(body, "No allowlisted") {
		t.Fatalf("scope.md task should synthesize without empty stub: %q", body)
	}
	// Dest path alone is enough when task text is sparse.
	sparse := collaboration.CollaborationTask{Title: "Task 1"}
	byPath := synthesizeResourceAPIMarkdownDeliverable(sparse, "collabs/abc/scope.md")
	if !strings.Contains(byPath, "schema") {
		t.Fatalf("expected scope synth from dest path alone, got %q", byPath)
	}
	review := collaboration.CollaborationTask{
		Title:       "Review API docs",
		Description: "Write collabs/<id>/existing-schema.md summarizing current schemas",
	}
	rev := synthesizeResourceAPIMarkdownDeliverable(review, "collabs/x/existing-schema.md")
	if !strings.Contains(rev, "Existing Schema") || !strings.Contains(rev, "registration") {
		t.Fatalf("expected existing-schema seed, got %q", rev)
	}
}


func TestCollabLightReadSources_InfersShortCollabPrefix(t *testing.T) {
	dir := t.TempDir()
	prior := filepath.Join(dir, "collabs", "b222bffe-39e8-4b00-91ca-ee1c555b9592")
	_ = os.MkdirAll(prior, 0o755)
	_ = os.WriteFile(filepath.Join(prior, "index.html"), []byte("<html>Collaboration Station XSS</html>\n"), 0o644)
	_ = os.WriteFile(filepath.Join(prior, "style.css"), []byte("body{color:black}\n"), 0o644)

	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", protocol.AgentInfo{}, "body")
	msg.Metadata = map[string]interface{}{
		"task_title":       "Security audit",
		"task_description": "Write collabs/x/security-audit.md reviewing b222bffe HTML/CSS for XSS",
	}
	got := collabLightReadSources(dir, msg)
	if _, ok := got["collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/index.html"]; !ok {
		t.Fatalf("expected prior HTML source, got %v", got)
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
