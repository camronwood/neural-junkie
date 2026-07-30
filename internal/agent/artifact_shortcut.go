package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/artifacts"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

var (
	mermaidFenceRE = regexp.MustCompile("(?is)```(?:mermaid)?\\s*([\\s\\S]*?)```")
	mermaidStartRE = regexp.MustCompile(`(?i)^(graph|flowchart|sequenceDiagram|classDiagram|stateDiagram|erDiagram|journey|gantt|pie|mindmap|timeline|quadrantChart|sankey|xychart|block-beta)\b`)
	mermaidInitRE  = regexp.MustCompile(`(?is)^%%\{init:.*?\}%%\s*`)
)

// wantsMarkdownCanvas classifies the artifact *kind* for a create-style ask —
// nj.markdown (reports, notes, generic "new canvas") vs Mermaid/maps/charts.
// Callers must independently confirm the turn is a stamped artifact turn
// (neuralCanvasDeliverableTurn) before using this to pick a renderer.
func wantsMarkdownCanvas(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	c = strings.ReplaceAll(c, "canvans", "canvas")
	c = strings.ReplaceAll(c, "canvass", "canvas")
	c = strings.ReplaceAll(c, "canvus", "canvas")
	if c == "" {
		return false
	}
	if wantsMermaidCanvas(c) {
		return false
	}
	if strings.Contains(c, "mermaid") || strings.Contains(c, "diagram") {
		return false
	}
	if strings.Contains(c, "chart") || strings.Contains(c, "timeline") ||
		strings.Contains(c, "table") || strings.Contains(c, "nj.map") {
		return false
	}
	if strings.Contains(c, "markdown") || strings.Contains(c, "report") ||
		strings.Contains(c, "brief") || strings.Contains(c, "writeup") ||
		strings.Contains(c, "write-up") {
		return true
	}
	if strings.Contains(c, "notes") && (strings.Contains(c, "canvas") || strings.Contains(c, "artifact")) {
		return true
	}
	// Generic "create a new canvas" / "make a canvas" → markdown by default.
	// Do not treat bare show/open as a create (those often mean the open canvas).
	if strings.Contains(c, "new canvas") || strings.Contains(c, "new neural canvas") ||
		strings.Contains(c, "new artifact") {
		return true
	}
	for _, verb := range []string{"create", "make", "generate", "build", "produce", "render"} {
		if strings.Contains(c, verb) {
			return true
		}
	}
	return false
}

// canvasAskPrefersNonMermaid reports canvas creates that must not revise an open Mermaid.
// Callers must already know the turn is a stamped artifact turn.
func canvasAskPrefersNonMermaid(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	if wantsMarkdownCanvas(c) {
		return true
	}
	if strings.Contains(c, "mermaid") || strings.Contains(c, "diagram") {
		return false
	}
	return strings.Contains(c, "chart") || strings.Contains(c, "timeline") ||
		strings.Contains(c, "table") || strings.Contains(c, "map ") ||
		strings.HasPrefix(c, "map") || strings.Contains(c, "nj.map")
}

// wantsMermaidCanvas classifies the artifact *kind* for a create-style ask as
// nj.mermaid. Revisions are selected structurally (ActionArtifact + open mermaid
// in channel). Callers must independently confirm the turn is a stamped
// artifact turn (neuralCanvasDeliverableTurn) before using this classification.
func wantsMermaidCanvas(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	// Explicit markdown/report wins over loose "architecture" wording.
	if strings.Contains(c, "markdown") || strings.Contains(c, "report") ||
		strings.Contains(c, "brief") || strings.Contains(c, "writeup") ||
		strings.Contains(c, "write-up") {
		return false
	}
	if strings.Contains(c, "mermaid") || strings.Contains(c, "diagram") {
		return true
	}
	return false
}

// tryNeuralCanvasMarkdownShortcut creates or revises a markdown Neural Canvas without
// relying on the model to call create_artifact / update_artifact.
func (a *Agent) tryNeuralCanvasMarkdownShortcut(
	ctx context.Context,
	msg *protocol.Message,
	prompt string,
	eff ai.AIProvider,
) (string, bool) {
	if a == nil || msg == nil || eff == nil {
		return "", false
	}
	if !neuralCanvasDeliverableTurn(msg) {
		return "", false
	}
	if !artifactToolsEnabledForMessage(msg) {
		return "", false
	}

	priorMD := a.findRecentMarkdownArtifact(msg)
	explicitCreate := wantsExplicitNewMarkdownCanvas(msg)

	// Status / meta questions about the open page stay in chat.
	if priorMD != nil && looksLikeCanvasStatusQuestion(msg.Content) {
		return "", false
	}

	// Fill-in / revise open markdown page (including embed mermaid or image asks).
	// Prefer update whenever a page is open unless the user explicitly asked for a new one.
	if priorMD != nil && !explicitCreate {
		if wantsCanvasPageImageEmbed(msg.Content) {
			if resp, ok := a.tryNeuralCanvasMarkdownImageEmbedShortcut(ctx, msg, priorMD, eff); ok {
				return resp, true
			}
		}
		if resp, ok := a.tryNeuralCanvasMarkdownUpdateShortcut(ctx, msg, prompt, eff, priorMD); ok {
			return resp, true
		}
		return "", false
	}
	if !explicitCreate {
		return "", false
	}
	// Explicit mermaid-only creates must not become a markdown page.
	if wantsMermaidCanvas(msg.Content) && !decisionHasReasonCode(msg, "blank_canvas") &&
		!decisionHasReasonCode(msg, "workspace_report") {
		return "", false
	}

	streamMsgID := StreamMessageIDFromContext(ctx)
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "start", Name: createArtifactToolName, Preview: "Creating Neural Canvas Markdown…",
	})

	kind := markdownCanvasCreateKind(msg)
	title := markdownCanvasTitleForKind(msg, kind)
	body := ""

	switch kind {
	case "prior_reference":
		if prior := a.priorAssistantContentForCanvas(msg); prior != "" {
			body = markdownBodyFromPriorAssistantContent(prior)
			a.recordKnowledgeExecutedFor(msg.ID, "prior_reference")
			if h1 := firstMarkdownH1(body); h1 != "" && h1 != "Canvas" {
				title = h1
			}
		}
		if body == "" {
			body = blankMarkdownScaffold(title)
			kind = "blank_canvas"
		}
	case "workspace_report":
		wsContext := a.workspaceArchitectureContext(msg, prompt)
		body = strings.TrimSpace(fallbackMarkdownFromWorkspace(msg, wsContext))
		if generated, err := a.generateMarkdownForCanvas(ctx, msg, wsContext, "", eff); err == nil {
			if cleaned := strings.TrimSpace(stripMarkdownFence(generated)); cleaned != "" &&
				!looksLikeSpuriousCanvasJSONPayload(cleaned) {
				body = cleaned
			}
		}
	default: // blank_canvas
		body = blankMarkdownScaffold(title)
	}

	if looksLikeSpuriousCanvasJSONPayload(body) {
		body = blankMarkdownScaffold(title)
		kind = "blank_canvas"
	}

	if strings.TrimSpace(body) == "" {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: createArtifactToolName, Preview: "empty markdown payload",
		})
		return "I couldn't create the requested Neural Canvas artifact in this turn.", true
	}

	data, err := json.Marshal(body)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: createArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't create the Neural Canvas: %v", err), true
	}
	input, err := json.Marshal(map[string]any{
		"title":       title,
		"renderer_id": "nj.markdown",
		"media_type":  "text/markdown",
		"kind":        "markdown",
		"data":        json.RawMessage(data),
		"fallback":    body,
	})
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: createArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't create the Neural Canvas: %v", err), true
	}

	result, err := a.executeArtifactTool(ctx, msg, createArtifactToolName, input)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: createArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't create the Neural Canvas: %v", err), true
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "result", Name: createArtifactToolName, Preview: result,
	})
	if kind == "blank_canvas" {
		return fmt.Sprintf("Opened a blank Neural Canvas (**%s**). Keep chatting to fill it in. %s", title, result), true
	}
	project := workspaceDisplayName(msg)
	if project == "" {
		project = "the shared workspace"
	}
	return fmt.Sprintf("Posted a Neural Canvas markdown report for **%s**. %s", project, result), true
}

func decisionHasReasonCode(msg *protocol.Message, code string) bool {
	decision, ok := stampedDecision(msg)
	if !ok {
		return false
	}
	return decisionHasReason(decision, code)
}

// wantsExplicitNewMarkdownCanvas reports a create of a *new* markdown page
// (blank, report, or prior-summary canvas) — not a fill-in of an open page.
func wantsExplicitNewMarkdownCanvas(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	c := strings.ToLower(strings.TrimSpace(msg.Content))
	if c == "" {
		return false
	}
	if looksLikeCanvasStatusQuestion(c) || looksLikeOpenCanvasFillAsk(c) {
		return false
	}
	if decisionHasReasonCode(msg, "workspace_report") || looksLikeWorkspaceReportAsk(c) {
		return true
	}
	if looksLikeGenericBlankCanvasCreate(c) {
		return true
	}
	if decisionHasReasonCode(msg, "blank_canvas") {
		return true
	}
	// "create a canvas with that summary" / generic markdown create verbs.
	return wantsMarkdownCanvas(c)
}

func looksLikeCanvasStatusQuestion(content string) bool {
	return intent.LooksLikeCanvasStatusQuestion(content)
}

func looksLikeOpenCanvasFillAsk(content string) bool {
	return intent.LooksLikeOpenCanvasFillAsk(content)
}

func canvasFillNeedsLiveLookup(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	cues := []string{
		"weather", "forecast", "temperature", "humidity",
		"stock price", "share price", "news", "headline",
		"today's", "todays", "current time", "exchange rate",
		"look up", "lookup", "search for", "google ",
		"real-time", "realtime", "live data",
	}
	for _, cue := range cues {
		if strings.Contains(c, cue) {
			return true
		}
	}
	return false
}

// markdownCanvasCreateKind picks blank vs workspace report vs prior body.
// Default for generic create is blank_canvas (not an auto workspace report).
func markdownCanvasCreateKind(msg *protocol.Message) string {
	if msg == nil {
		return "blank_canvas"
	}
	// "with that information" / "add that to a canvas" must beat blank_canvas reason
	// codes that tiny classifiers spray on every canvas turn.
	if looksLikePriorContentCanvasAsk(msg.Content) {
		return "prior_reference"
	}
	if decisionHasReasonCode(msg, "workspace_report") || looksLikeWorkspaceReportAsk(msg.Content) {
		return "workspace_report"
	}
	if decisionHasReasonCode(msg, "blank_canvas") || looksLikeGenericBlankCanvasCreate(msg.Content) {
		return "blank_canvas"
	}
	decision, ok := stampedDecision(msg)
	if ok && ShouldRunPriorReference(routing.PlanKnowledgeRouteForDecision(decision)) {
		return "prior_reference"
	}
	return "blank_canvas"
}

func looksLikePriorContentCanvasAsk(content string) bool {
	return intent.LooksLikePriorContentCanvasAsk(content)
}

func looksLikeGenericBlankCanvasCreate(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" || looksLikeWorkspaceReportAsk(c) {
		return false
	}
	if strings.Contains(c, "blank") || strings.Contains(c, "empty") ||
		strings.Contains(c, "fill in") || strings.Contains(c, "fill-in") {
		return true
	}
	return strings.Contains(c, "new canvas") || strings.Contains(c, "new neural canvas") ||
		strings.Contains(c, "new artifact")
}

func looksLikeWorkspaceReportAsk(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	hasReport := strings.Contains(c, "report") || strings.Contains(c, "summary") ||
		strings.Contains(c, "summariz") || strings.Contains(c, "writeup") ||
		strings.Contains(c, "write-up") || strings.Contains(c, "brief") ||
		strings.Contains(c, "overview")
	if !hasReport {
		return false
	}
	return strings.Contains(c, "project") || strings.Contains(c, "workspace") ||
		strings.Contains(c, "repo") || strings.Contains(c, "codebase") ||
		strings.Contains(c, "this app") || strings.Contains(c, "architecture")
}

func blankMarkdownScaffold(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Canvas"
	}
	return "# " + title + "\n\n"
}

func markdownCanvasTitleForKind(msg *protocol.Message, kind string) string {
	if kind == "workspace_report" {
		return markdownCanvasTitle(msg)
	}
	return "Canvas"
}

// priorAssistantContentForCanvas returns earlier assistant text when the turn's
// semantic knowledge plan includes prior_reference, or when the user asks to put
// "that information" onto a Neural Canvas.
func (a *Agent) priorAssistantContentForCanvas(msg *protocol.Message) string {
	if a == nil || msg == nil {
		return ""
	}
	history := a.historyForPriorReference(msg.Channel)
	content := ""
	if ShouldRunPriorReference(a.effectiveKnowledgePlanFromMessage(msg)) {
		content = strings.TrimSpace(findPriorAssistantContent(history, msg.ID, a.Info.ID, priorReferenceMinChars))
	}
	if content == "" && looksLikePriorContentCanvasAsk(msg.Content) {
		// Meeting dumps / ASCII "canvas" replies are often under the 400-char prior
		// threshold but are still the body the user wants posted.
		content = strings.TrimSpace(findRecentAssistantChatContent(history, msg.ID, a.Info.ID, 80))
	}
	if content == "" {
		return ""
	}
	if len(content) > priorReferenceInjectMaxBytes {
		content = content[:priorReferenceInjectMaxBytes] + "\n…(prior content truncated)\n"
	}
	return content
}

// findRecentAssistantChatContent is a lenient prior-body finder for canvas fills.
func findRecentAssistantChatContent(history []*protocol.Message, skipMsgID, agentID string, minChars int) string {
	if minChars <= 0 {
		minChars = 80
	}
	best := ""
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID || assistantMessageSkippableForPriorReference(m) {
			continue
		}
		if agentID != "" && m.From.ID != agentID {
			continue
		}
		if m.Type != protocol.MessageTypeChat && m.Type != protocol.MessageTypeAnswer {
			continue
		}
		body := strings.TrimSpace(m.Content)
		if len(body) < minChars {
			continue
		}
		lower := strings.ToLower(body)
		// Skip empty acknowledgements / tool status lines.
		if strings.HasPrefix(lower, "updated neural canvas") ||
			strings.HasPrefix(lower, "opened a blank neural canvas") ||
			strings.HasPrefix(lower, "posted a neural canvas") {
			continue
		}
		if len(body) > len(best) {
			best = body
		}
	}
	return best
}

func markdownBodyFromPriorAssistantContent(prior string) string {
	prior = strings.TrimSpace(prior)
	if prior == "" {
		return ""
	}
	// Strip leading chatter / "Certainly! Below is a canvas…" wrappers and fences.
	prior = stripMarkdownFence(prior)
	lines := strings.Split(prior, "\n")
	start := 0
	for start < len(lines) {
		trim := strings.TrimSpace(lines[start])
		lower := strings.ToLower(trim)
		if trim == "" ||
			strings.HasPrefix(lower, "certainly") ||
			strings.HasPrefix(lower, "sure") ||
			strings.HasPrefix(lower, "here") ||
			strings.Contains(lower, "below is a canvas") ||
			strings.HasPrefix(trim, "===") ||
			trim == "```" || strings.HasPrefix(trim, "```") {
			start++
			continue
		}
		break
	}
	end := len(lines)
	for end > start {
		trim := strings.TrimSpace(lines[end-1])
		lower := strings.ToLower(trim)
		if trim == "" || trim == "```" ||
			strings.HasPrefix(lower, "feel free") ||
			strings.HasPrefix(lower, "let me know") {
			end--
			continue
		}
		break
	}
	body := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	if body == "" {
		return prior
	}
	if !strings.HasPrefix(strings.TrimSpace(body), "#") {
		// Promote first non-empty line to H1 when it looks like a title line.
		first := strings.TrimSpace(strings.Split(body, "\n")[0])
		if first != "" && !strings.HasPrefix(first, "-") && !strings.HasPrefix(first, "*") &&
			!strings.HasPrefix(first, "**") && len(first) < 120 {
			rest := strings.TrimSpace(strings.TrimPrefix(body, first))
			body = "# " + strings.Trim(first, "= ") + "\n\n" + rest
		}
	}
	return body
}

func looksLikeSpuriousCanvasJSONPayload(body string) bool {
	t := strings.TrimSpace(body)
	if t == "" || t[0] != '{' {
		return false
	}
	lower := strings.ToLower(t)
	return strings.Contains(lower, "renderer_id") ||
		strings.Contains(lower, "media_type") ||
		strings.Contains(lower, "workspace_id") ||
		strings.Contains(lower, `"kind"`) ||
		strings.Contains(lower, "fallback")
}

func (a *Agent) generateMarkdownForCanvas(
	ctx context.Context,
	msg *protocol.Message,
	wsContext string,
	priorContent string,
	eff ai.AIProvider,
) (string, error) {
	var b strings.Builder
	b.WriteString("Write a concise Markdown report for Neural Canvas. Output Markdown only — no fences, no FILE_CHANGE.\n")
	b.WriteString("Ground claims in the provided context. Include headings, short bullets, and an Open questions section when useful.\n")
	b.WriteString("Do NOT invent Neural Canvas product plumbing or invent files that are not in context.\n")
	b.WriteString("The subject is the user's shared workspace / prior assistant content — never treat Neural Canvas itself as the project being documented.\n")
	b.WriteString("\nUSER REQUEST:\n")
	b.WriteString(strings.TrimSpace(msg.Content))
	b.WriteString("\n")
	if strings.TrimSpace(priorContent) != "" {
		b.WriteString("\n=== PRIOR ASSISTANT CONTENT (referenced) ===\n")
		b.WriteString("Use this as the primary source for the report. Preserve its claims; format as Markdown if needed. Do not replace it with a generic template.\n\n")
		b.WriteString(priorContent)
		b.WriteString("\n")
	}
	if strings.TrimSpace(wsContext) != "" {
		b.WriteString("\nWORKSPACE CONTEXT:\n")
		b.WriteString(wsContext)
		b.WriteString("\n")
	}
	return eff.GenerateResponse(ctx, b.String(), nil)
}

func fallbackMarkdownFromWorkspace(msg *protocol.Message, wsContext string) string {
	project := workspaceDisplayName(msg)
	if project == "" {
		project = "Workspace"
	}
	nodes := topLevelArchitectureNodes(msg, wsContext)
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(project)
	b.WriteString("\n\n")
	b.WriteString("## Summary\n\n")
	b.WriteString("Workspace-grounded overview for Neural Canvas.\n\n")
	if len(nodes) > 0 {
		b.WriteString("## Structure\n\n")
		for _, n := range nodes {
			b.WriteString("- `")
			b.WriteString(n)
			b.WriteString("`\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Open questions\n\n")
	b.WriteString("- What should we deepen next?\n")
	return b.String()
}

func markdownCanvasTitle(msg *protocol.Message) string {
	if msg == nil {
		return "Markdown report"
	}
	project := workspaceDisplayName(msg)
	if project != "" {
		return project + " report"
	}
	return "Markdown report"
}

func stripMarkdownFence(raw string) string {
	s := strings.TrimSpace(raw)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		lang := strings.TrimSpace(s[:nl])
		if strings.EqualFold(lang, "markdown") || strings.EqualFold(lang, "md") || lang == "" {
			s = s[nl+1:]
		}
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func markdownSourceFromPayload(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(payload, &asString); err == nil {
		return asString // preserve intentional leading/trailing whitespace for blank scaffolds
	}
	var asObj map[string]any
	if err := json.Unmarshal(payload, &asObj); err == nil {
		for _, key := range []string{"markdown", "content", "text", "body"} {
			if v, ok := asObj[key].(string); ok {
				return v
			}
		}
	}
	return string(payload)
}

// openArtifactFromMessageMetadata returns the canvas the client reports as currently open.
// Survives clear-history when the desktop stamps open_artifact_* (flat or nested) on metadata.
func openArtifactFromMessageMetadata(msg *protocol.Message) (id, renderer, title string) {
	if msg == nil || msg.Metadata == nil {
		return "", "", ""
	}
	meta := msg.Metadata
	id = metadataString(meta, "open_artifact_id")
	renderer = metadataString(meta, "open_artifact_renderer")
	title = metadataString(meta, "open_artifact_title")
	if id == "" {
		if nested, ok := meta["open_artifact"].(map[string]interface{}); ok {
			id = metadataString(nested, "id", "artifact_id")
			if renderer == "" {
				renderer = metadataString(nested, "renderer_id", "rendererId", "renderer")
			}
			if title == "" {
				title = metadataString(nested, "title")
			}
		} else if nestedStr, ok := meta["open_artifact"].(map[string]string); ok {
			id = strings.TrimSpace(nestedStr["id"])
			renderer = strings.TrimSpace(nestedStr["renderer_id"])
			if renderer == "" {
				renderer = strings.TrimSpace(nestedStr["rendererId"])
			}
			title = strings.TrimSpace(nestedStr["title"])
		}
	}
	if id == "" {
		if ref, ok := messageArtifactReference(msg); ok {
			id = strings.TrimSpace(ref.ID)
			if renderer == "" {
				renderer = strings.TrimSpace(ref.RendererID)
			}
			if title == "" {
				title = strings.TrimSpace(ref.Title)
			}
		}
	}
	id = strings.TrimSpace(id)
	if id == "" || id == "__library__" {
		return "", "", ""
	}
	return id, strings.TrimSpace(renderer), strings.TrimSpace(title)
}

func metadataString(meta map[string]interface{}, keys ...string) string {
	if meta == nil {
		return ""
	}
	for _, key := range keys {
		if v, ok := meta[key].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func (a *Agent) findRecentMarkdownArtifact(msg *protocol.Message) *artifacts.Artifact {
	if a == nil || msg == nil {
		return nil
	}
	store, err := getAgentArtifactStore()
	if err != nil || store == nil {
		return nil
	}
	// Prefer the canvas the client reports as open (survives clear-history).
	if id, _, _ := openArtifactFromMessageMetadata(msg); id != "" {
		if art, err := store.Get(id); err == nil && art != nil && isMarkdownArtifact(art) {
			return art
		}
	}
	if id := recentMarkdownArtifactID(a.channelHistory(msg.Channel), msg.ID); id != "" {
		if art, err := store.Get(id); err == nil && art != nil && isMarkdownArtifact(art) {
			return art
		}
	}
	// Do not fall back to store.List: after clear-history chat is empty but
	// channel-linked canvases remain; silently revising them reintroduces prior topics.
	return nil
}

func isMarkdownArtifact(art *artifacts.Artifact) bool {
	if art == nil {
		return false
	}
	if art.Renderer.ID == "nj.markdown" {
		return true
	}
	return strings.Contains(strings.ToLower(art.Renderer.MediaType), "markdown")
}

func recentMarkdownArtifactID(history []*protocol.Message, skipMsgID string) string {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 40; i-- {
		msg := history[i]
		if msg == nil || msg.ID == skipMsgID {
			continue
		}
		seen++
		ref, ok := messageArtifactReference(msg)
		if !ok || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		rid := strings.ToLower(ref.RendererID)
		media := strings.ToLower(ref.MediaType)
		if rid == "nj.markdown" || strings.Contains(media, "markdown") {
			return ref.ID
		}
	}
	return ""
}

func wantsCanvasPageImageEmbed(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	if !(strings.Contains(c, "image") || strings.Contains(c, "picture") ||
		strings.Contains(c, "photo") || strings.Contains(c, "illustration") ||
		strings.Contains(c, "skyline") || strings.Contains(c, "draw") ||
		strings.Contains(c, "generate")) {
		return false
	}
	return strings.Contains(c, "add") || strings.Contains(c, "put") ||
		strings.Contains(c, "insert") || strings.Contains(c, "embed") ||
		strings.Contains(c, "include") || strings.Contains(c, "on the canvas") ||
		strings.Contains(c, "to the canvas") || strings.Contains(c, "on this")
}

func (a *Agent) tryNeuralCanvasMarkdownUpdateShortcut(
	ctx context.Context,
	msg *protocol.Message,
	prompt string,
	eff ai.AIProvider,
	current *artifacts.Artifact,
) (string, bool) {
	if current == nil {
		current = a.findRecentMarkdownArtifact(msg)
	}
	if current == nil || !isMarkdownArtifact(current) {
		return "", false
	}
	existing := markdownSourceFromPayload(current.Payload)

	streamMsgID := StreamMessageIDFromContext(ctx)
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "start", Name: updateArtifactToolName, Preview: "Updating Neural Canvas Markdown…",
	})

	liveCtx := ""
	if canvasFillNeedsLiveLookup(msg.Content) {
		liveCtx = a.lookupLiveContextForCanvas(ctx, msg, streamMsgID)
	}

	body := existing
	usedPrior := false
	renameOnly := canvasRenameOnlyAsk(msg.Content)
	if !renameOnly {
		if looksLikePriorContentCanvasAsk(msg.Content) {
			if prior := a.priorAssistantContentForCanvas(msg); prior != "" {
				body = markdownBodyFromPriorAssistantContent(prior)
				usedPrior = true
				a.recordKnowledgeExecutedFor(msg.ID, "prior_reference")
			}
		}
		if !usedPrior {
			revised, err := a.generateMarkdownRevisionForCanvas(ctx, msg, existing, liveCtx, eff)
			if err == nil {
				if cleaned := strings.TrimSpace(stripMarkdownFence(revised)); cleaned != "" &&
					!looksLikeSpuriousCanvasJSONPayload(cleaned) {
					body = cleaned
				}
			}
		}
		if looksLikeSpuriousCanvasJSONPayload(body) {
			if prior := a.priorAssistantContentForCanvas(msg); prior != "" {
				body = markdownBodyFromPriorAssistantContent(prior)
			} else {
				body = existing
			}
		}
	}
	if strings.TrimSpace(body) == "" {
		body = existing
	}
	if strings.TrimSpace(body) == "" {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: updateArtifactToolName, Preview: "empty markdown payload",
		})
		return "I couldn't update the Neural Canvas in this turn.", true
	}

	data, err := json.Marshal(body)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: updateArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}
	title := resolveMarkdownCanvasUpdateTitle(current.Title, msg.Content, body)
	if usedPrior {
		if h1 := firstMarkdownH1(body); h1 != "" && h1 != "Canvas" {
			title = h1
		}
	}
	if title != "" {
		body = ensureMarkdownH1(body, title)
		data, err = json.Marshal(body)
		if err != nil {
			a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
				Kind: "error", Name: updateArtifactToolName, Preview: err.Error(),
			})
			return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
		}
	}
	input := map[string]any{
		"artifact_id":       current.ID,
		"expected_revision": current.Revision,
		"data":              json.RawMessage(data),
		"fallback":          body,
	}
	if title != "" {
		input["title"] = title
	}
	rawInput, err := json.Marshal(input)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: updateArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}

	result, err := a.executeArtifactTool(ctx, msg, updateArtifactToolName, rawInput)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: updateArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "result", Name: updateArtifactToolName, Preview: result,
	})
	return fmt.Sprintf("Updated the Neural Canvas to revision %d. %s", current.Revision+1, result), true
}

func firstMarkdownH1(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// resolveMarkdownCanvasUpdateTitle picks a title for a markdown canvas update.
// Explicit rename asks always win (even when the page already has a non-default
// title). Otherwise the title is frozen once it leaves the default "Canvas".
func resolveMarkdownCanvasUpdateTitle(currentTitle, userAsk, body string) string {
	if renamed := titleFromCanvasRenameAsk(userAsk); renamed != "" {
		return renamed
	}
	title := strings.TrimSpace(currentTitle)
	if title != "" && title != "Canvas" {
		return title
	}
	if derived := titleFromCanvasFillRequest(userAsk); derived != "" {
		return derived
	}
	if h1 := firstMarkdownH1(body); h1 != "" && h1 != "Canvas" && titleGroundedInUserAsk(h1, userAsk) {
		return h1
	}
	if title == "" {
		return "Canvas"
	}
	return title
}

// canvasRenameOnlyAsk reports a turn that only renames the open canvas (no
// content fill/revise). Those turns skip the revision LLM and only retitle.
func canvasRenameOnlyAsk(content string) bool {
	if !intent.LooksLikeCanvasRenameAsk(content) {
		return false
	}
	if titleFromCanvasRenameAsk(content) == "" {
		return false
	}
	c := strings.ToLower(strings.TrimSpace(content))
	for _, cue := range []string{
		"add ", "put ", "fill", "include ", "insert ", "embed ",
		"write ", "append ", "update the canvas", "update this",
	} {
		if strings.Contains(c, cue) {
			return false
		}
	}
	return true
}

// titleFromCanvasRenameAsk extracts the intended canvas title from a rename ask.
// Returns "" when the text is not a rename or no target title can be recovered.
func titleFromCanvasRenameAsk(content string) string {
	c := strings.TrimSpace(content)
	if c == "" || !intent.LooksLikeCanvasRenameAsk(c) {
		return ""
	}
	lower := strings.ToLower(c)
	markers := []string{
		"the title of the document should be ",
		"the title of the page should be ",
		"the title of the canvas should be ",
		"title of the document should be ",
		"title of the page should be ",
		"title of the canvas should be ",
		"the title should be ",
		"title should be ",
		"change the title to ",
		"change its title to ",
		"set the title to ",
		"rename it to ",
		"rename to ",
		"retitle it to ",
		"retitle to ",
		"call it ",
		"title it ",
		"name it ",
		"name this ",
		"call this ",
		"rename it ",
		"rename ",
		"retitle ",
	}
	rest := ""
	for _, m := range markers {
		if idx := strings.Index(lower, m); idx >= 0 {
			rest = strings.TrimSpace(c[idx+len(m):])
			break
		}
	}
	if rest == "" {
		return ""
	}
	if q := firstQuotedTitleSpan(rest); q != "" {
		return cleanCanvasRenameTitle(q)
	}
	return cleanCanvasRenameTitle(rest)
}

func firstQuotedTitleSpan(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	for i, r := range runes {
		var closer rune
		switch r {
		case '"':
			closer = '"'
		case '\'':
			closer = '\''
		case '\u201c':
			closer = '\u201d'
		case '\u2018':
			closer = '\u2019'
		default:
			continue
		}
		for j := i + 1; j < len(runes); j++ {
			if runes[j] == closer {
				return string(runes[i+1 : j])
			}
		}
		return strings.TrimSpace(string(runes[i+1:]))
	}
	return ""
}

func cleanCanvasRenameTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"'` \t\n\r\u201c\u201d\u2018\u2019")
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "!.?;,")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Keep St. / U.S. style abbreviations; only drop a trailing sentence period
	// when the title has multiple words and does not look like an abbreviation.
	if strings.HasSuffix(s, ".") && strings.Contains(s, " ") {
		trimmed := strings.TrimSuffix(s, ".")
		if !strings.HasSuffix(trimmed, ".") {
			s = strings.TrimSpace(trimmed)
		}
	}
	if len([]rune(s)) > 120 {
		s = string([]rune(s)[:120])
	}
	return s
}

func titleFromCanvasFillRequest(content string) string {
	c := strings.TrimSpace(content)
	if c == "" {
		return ""
	}
	// Rename asks have a dedicated extractor; never Title-Case the utterance.
	if intent.LooksLikeCanvasRenameAsk(c) {
		if renamed := titleFromCanvasRenameAsk(c); renamed != "" {
			return renamed
		}
	}
	lower := strings.ToLower(c)
	for _, phrase := range []string{
		"can you", "could you", "would you", "please",
		"put it in the canvas", "put it on the canvas", "put in the canvas",
		"in the canvas", "on the canvas", "to the canvas", "into the canvas",
		"fill in the canvas", "fill the canvas", "update the canvas",
		"get me", "get ", "todays ", "today's ", "today ",
		"and put", "for me",
		"lets name it", "let's name it", "name it", "call it", "title it",
		"rename it to", "rename to", "retitle it to", "change the title to",
		"set the title to", "the title should be", "title should be",
		"lets ", "let's ",
	} {
		lower = strings.ReplaceAll(lower, phrase, " ")
	}
	lower = strings.Map(func(r rune) rune {
		switch r {
		case ',', '.', '!', '?', ';', ':', '"', '\'':
			return ' '
		default:
			return r
		}
	}, lower)
	words := strings.Fields(lower)
	filtered := make([]string, 0, len(words))
	skip := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true,
		"it": true, "this": true, "that": true, "with": true, "from": true,
		"into": true, "onto": true, "there": true, "here": true,
		"name": true, "rename": true, "retitle": true, "call": true,
		"lets": true, "let": true, "title": true, "set": true,
	}
	for _, w := range words {
		if skip[w] || w == "canvas" || w == "page" || w == "document" {
			continue
		}
		filtered = append(filtered, w)
	}
	if len(filtered) == 0 {
		return ""
	}
	if len(filtered) > 6 {
		filtered = filtered[:6]
	}
	for i, w := range filtered {
		if len(w) == 0 {
			continue
		}
		filtered[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(filtered, " ")
}

func titleGroundedInUserAsk(title, userAsk string) bool {
	title = strings.TrimSpace(title)
	ask := strings.ToLower(strings.TrimSpace(userAsk))
	if title == "" || ask == "" {
		return false
	}
	tokens := strings.Fields(strings.ToLower(strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return ' '
	}, title)))
	meaningful := 0
	matched := 0
	for _, tok := range tokens {
		if len(tok) < 3 {
			continue
		}
		meaningful++
		if strings.Contains(ask, tok) {
			matched++
		}
	}
	if meaningful == 0 {
		return false
	}
	return matched*2 >= meaningful // at least half of meaningful tokens appear in the ask
}

func ensureMarkdownH1(body, title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			lines[i] = "# " + title
			return strings.Join(lines, "\n")
		}
	}
	if strings.TrimSpace(body) == "" {
		return "# " + title + "\n\n"
	}
	return "# " + title + "\n\n" + body
}

// tryOpenCanvasMetaAnswer answers title/name questions about the open canvas without
// tools or meteorology digressions. Returns ("", false) when not applicable.
func (a *Agent) tryOpenCanvasMetaAnswer(msg *protocol.Message) (string, bool) {
	if a == nil || msg == nil {
		return "", false
	}
	if !intent.LooksLikeCanvasTitleQuestion(msg.Content) {
		return "", false
	}
	art := a.findRecentMarkdownArtifact(msg)
	if art == nil {
		art = a.findRecentMermaidArtifact(msg)
	}
	if art == nil {
		return "I don't have an open Neural Canvas tied to this chat turn (cleared history drops the prior page link). Open the canvas tab you mean, or say what to title a new one.", true
	}
	title := strings.TrimSpace(art.Title)
	if title == "" {
		title = firstMarkdownH1(markdownSourceFromPayload(art.Payload))
	}
	if title == "" {
		title = "Canvas"
	}
	return fmt.Sprintf(
		"The open Neural Canvas is titled **%s**. That comes from this artifact's title/heading — clearing chat history does not erase the page, and I should not invent a meteorology explanation. Tell me what to rename it to.",
		title,
	), true
}

func (a *Agent) generateMarkdownRevisionForCanvas(
	ctx context.Context,
	msg *protocol.Message,
	existing string,
	liveContext string,
	eff ai.AIProvider,
) (string, error) {
	var b strings.Builder
	b.WriteString("Revise the Markdown document below per the user request. Output Markdown only — no fences, no FILE_CHANGE.\n")
	b.WriteString("Preserve existing sections, lists, mermaid fenced blocks, and image links unless the user asks to change or remove them.\n")
	b.WriteString("Add or fill content as requested (new headings, lists, sections).\n")
	b.WriteString("When the user asks for a diagram, insert or update a ```mermaid fenced block in this document with valid Mermaid source.\n")
	b.WriteString("If the H1 is still \"Canvas\", retitle it using ONLY words/topics from USER REQUEST (e.g. location + topic).\n")
	b.WriteString("If the user asks to rename/retitle the page, set the H1 to their exact requested title.\n")
	b.WriteString("Do NOT invent generic labels (e.g. \"Weather Forecast\") that are not in USER REQUEST.\n")
	b.WriteString("Do NOT reuse topics from CURRENT DOCUMENT that are absent from USER REQUEST (cleared chat must not leak).\n")
	b.WriteString("Do NOT invent Neural Canvas product plumbing. Do NOT create a separate artifact — revise this page only.\n")
	b.WriteString("When LIVE LOOKUP RESULTS are provided, ground weather/facts in that data — do not refuse or invent.\n")
	b.WriteString("\nUSER REQUEST:\n")
	b.WriteString(strings.TrimSpace(msg.Content))
	b.WriteString("\n")
	if strings.TrimSpace(liveContext) != "" {
		b.WriteString("\n=== LIVE LOOKUP RESULTS ===\n")
		b.WriteString(liveContext)
		b.WriteString("\n")
	}
	b.WriteString("\nCURRENT DOCUMENT:\n")
	if strings.TrimSpace(existing) == "" {
		b.WriteString("(empty)\n")
	} else {
		b.WriteString(existing)
		b.WriteString("\n")
	}
	return eff.GenerateResponse(ctx, b.String(), nil)
}

// lookupLiveContextForCanvas runs web_search when available so canvas fill-ins
// (weather, news, etc.) are grounded before revising the page.
func (a *Agent) lookupLiveContextForCanvas(ctx context.Context, msg *protocol.Message, streamMsgID string) string {
	if a == nil || msg == nil {
		return ""
	}
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil || mcpServer.GetTool("web_search") == nil {
		return ""
	}
	query := strings.TrimSpace(msg.Content)
	if query == "" {
		return ""
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "start", Name: "web_search", Preview: "Looking up live facts for the canvas…",
	})
	input, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return ""
	}
	toolCtx := withWebSearchGuard(ctx)
	result, err := executeMCPTool(toolCtx, mcpServer, "web_search", input)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: "web_search", Preview: err.Error(),
		})
		return ""
	}
	result = guardWebSearchToolResult(toolCtx, "web_search", result)
	preview := result
	if len(preview) > 160 {
		preview = preview[:160] + "…"
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "result", Name: "web_search", Preview: preview,
	})
	const max = 8000
	if len(result) > max {
		result = result[:max] + "\n…(truncated)"
	}
	return result
}

func (a *Agent) tryNeuralCanvasMarkdownImageEmbedShortcut(
	ctx context.Context,
	msg *protocol.Message,
	current *artifacts.Artifact,
	eff ai.AIProvider,
) (string, bool) {
	if a == nil || msg == nil || current == nil || !isMarkdownArtifact(current) {
		return "", false
	}
	gen := ai.ImageGeneratorFromEnv()
	if gen == nil {
		// Fall through to text revision (may describe the image ask).
		return "", false
	}

	streamMsgID := StreamMessageIDFromContext(ctx)
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "start", Name: generateImageToolName, Preview: "Generating image for canvas…",
	})

	prompt := ImagePromptFromMessage(msg.Content)
	if prompt == "" {
		prompt = strings.TrimSpace(msg.Content)
	}
	mime, b64, err := gen.GenerateImage(ctx, prompt, "")
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: generateImageToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't generate an image for the canvas: %v", err), true
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) == 0 {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: generateImageToolName, Preview: "invalid image payload",
		})
		return "I couldn't decode the generated image for the canvas.", true
	}
	ext := "png"
	switch {
	case strings.Contains(mime, "jpeg"), strings.Contains(mime, "jpg"):
		ext = "jpg"
	case strings.Contains(mime, "webp"):
		ext = "webp"
	case strings.Contains(mime, "gif"):
		ext = "gif"
	}
	assetName := fmt.Sprintf("embed-%d.%s", time.Now().UnixNano(), ext)
	store, err := getAgentArtifactStore()
	if err != nil || store == nil {
		return "I couldn't store the canvas image asset.", true
	}
	if err := store.PutAsset(current.ID, assetName, raw); err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: generateImageToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't store the canvas image: %v", err), true
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "result", Name: generateImageToolName, Preview: assetName,
	})

	existing := markdownSourceFromPayload(current.Payload)
	alt := "Generated image"
	if h := firstMarkdownH1(existing); h != "" {
		alt = h
	}
	assetURL := fmt.Sprintf("/api/artifacts/%s/assets/%s", current.ID, assetName)
	imageBlock := fmt.Sprintf("\n\n![%s](%s)\n", alt, assetURL)
	body := strings.TrimRight(existing, "\n") + imageBlock
	if revised, err := a.generateMarkdownRevisionForCanvas(ctx, msg, body, "", eff); err == nil {
		if cleaned := strings.TrimSpace(stripMarkdownFence(revised)); cleaned != "" &&
			strings.Contains(cleaned, assetURL) {
			body = cleaned
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}
	input, err := json.Marshal(map[string]any{
		"artifact_id":       current.ID,
		"expected_revision": current.Revision,
		"data":              json.RawMessage(data),
		"fallback":          body,
	})
	if err != nil {
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "start", Name: updateArtifactToolName, Preview: "Embedding image on canvas…",
	})
	result, err := a.executeArtifactTool(ctx, msg, updateArtifactToolName, input)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: updateArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "result", Name: updateArtifactToolName, Preview: result,
	})
	return fmt.Sprintf("Added an image to the Neural Canvas (revision %d). %s", current.Revision+1, result), true
}

// tryNeuralCanvasMermaidShortcut builds or revises a Mermaid Neural Canvas without relying
// on the model to call create_artifact / update_artifact (local models often skip tools).
func (a *Agent) tryNeuralCanvasMermaidShortcut(
	ctx context.Context,
	msg *protocol.Message,
	prompt string,
	eff ai.AIProvider,
) (string, bool) {
	if a == nil || msg == nil || eff == nil {
		return "", false
	}
	// Stamp-first: only run this shortcut for turns the classifier already
	// authorized as a Neural Canvas artifact deliverable.
	if !neuralCanvasDeliverableTurn(msg) {
		return "", false
	}
	// Open collaborative markdown page owns diagram-add asks — do not spawn nj.mermaid.
	if md := a.findRecentMarkdownArtifact(msg); md != nil && !wantsMarkdownCanvas(msg.Content) {
		if wantsMermaidCanvas(msg.Content) || strings.Contains(strings.ToLower(msg.Content), "diagram") {
			return "", false
		}
	}
	createAsk := wantsMermaidCanvas(msg.Content)
	prior := a.findRecentMermaidArtifact(msg)
	if !createAsk && prior == nil {
		return "", false
	}
	// Markdown / other non-mermaid canvas creates must not revise the open Mermaid.
	if !createAsk && canvasAskPrefersNonMermaid(msg.Content) {
		return "", false
	}
	if !artifactToolsEnabledForMessage(msg) {
		return "", false
	}

	// Revisions: open mermaid + artifact turn without an explicit create ask.
	if prior != nil && !createAsk {
		if resp, ok := a.tryNeuralCanvasMermaidUpdateShortcut(ctx, msg, prompt, eff, prior); ok {
			return resp, true
		}
		return "", false
	}
	if !createAsk {
		return "", false
	}

	streamMsgID := StreamMessageIDFromContext(ctx)
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "start", Name: createArtifactToolName, Preview: "Creating Neural Canvas Mermaid…",
	})

	wsContext := a.workspaceArchitectureContext(msg, prompt)
	// Architecture creates are tree-first: build from the shared workspace so local
	// models cannot invent Neural Canvas / Create Artifact product plumbing.
	mermaid := sanitizeMermaidSource(fallbackMermaidFromWorkspace(msg, wsContext))
	if mermaid == "" {
		// No usable tree — last resort LLM, then sanitize/reject meta bleed.
		generated, genErr := a.generateMermaidForCanvas(ctx, msg, wsContext, eff)
		if genErr != nil {
			generated = ""
		}
		mermaid = sanitizeMermaidSource(generated)
		nodes := topLevelArchitectureNodes(msg, wsContext)
		if mermaid == "" ||
			looksLikeMetaCanvasProcessMermaid(mermaid) ||
			looksLikeInvalidMermaidArchitecture(mermaid) ||
			looksLikeBrokenMermaidBrackets(mermaid) ||
			!mermaidMentionsWorkspaceNodes(mermaid, nodes) {
			mermaid = ""
		}
	}
	if mermaid == "" {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: createArtifactToolName, Preview: "empty mermaid payload",
		})
		return "I couldn't create the requested Neural Canvas artifact in this turn.", true
	}

	title := mermaidCanvasTitle(msg)
	data, err := json.Marshal(mermaid)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: createArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't create the Neural Canvas: %v", err), true
	}
	input, err := json.Marshal(map[string]any{
		"title":       title,
		"renderer_id": "nj.mermaid",
		"media_type":  "text/vnd.mermaid",
		"kind":        "mermaid",
		"data":        json.RawMessage(data),
		"fallback":    "```mermaid\n" + mermaid + "\n```",
	})
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: createArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't create the Neural Canvas: %v", err), true
	}

	result, err := a.executeArtifactTool(ctx, msg, createArtifactToolName, input)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: createArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't create the Neural Canvas: %v", err), true
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "result", Name: createArtifactToolName, Preview: result,
	})
	project := workspaceDisplayName(msg)
	if project == "" {
		project = "the shared workspace"
	}
	return fmt.Sprintf("Posted a Neural Canvas Mermaid diagram of **%s**. %s", project, result), true
}

func (a *Agent) tryNeuralCanvasMermaidUpdateShortcut(
	ctx context.Context,
	msg *protocol.Message,
	prompt string,
	eff ai.AIProvider,
	current *artifacts.Artifact,
) (string, bool) {
	if current == nil {
		current = a.findRecentMermaidArtifact(msg)
	}
	if current == nil {
		return "", false
	}
	existing := sanitizeMermaidSource(mermaidSourceFromPayload(current.Payload))
	if existing == "" {
		return "", false
	}

	streamMsgID := StreamMessageIDFromContext(ctx)
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "start", Name: updateArtifactToolName, Preview: "Updating Neural Canvas Mermaid…",
	})

	wsContext := a.workspaceArchitectureContext(msg, prompt)
	mermaid := existing
	revised, err := a.generateMermaidRevisionForCanvas(ctx, msg, existing, wsContext, eff)
	if err == nil {
		revised = sanitizeMermaidSource(revised)
	} else {
		revised = ""
	}
	nodes := topLevelArchitectureNodes(msg, wsContext)
	if revised != "" &&
		!looksLikeMetaCanvasProcessMermaid(revised) &&
		!looksLikeInvalidMermaidArchitecture(revised) &&
		!looksLikeBrokenMermaidBrackets(revised) &&
		(len(nodes) == 0 || mermaidMentionsWorkspaceNodes(revised, nodes)) {
		mermaid = revised
	} else if looksLikeMetaCanvasProcessMermaid(existing) ||
		(len(nodes) > 0 && !mermaidMentionsWorkspaceNodes(existing, nodes)) {
		// Open diagram was ungrounded/meta — rebuild from the workspace tree.
		if tree := sanitizeMermaidSource(fallbackMermaidFromWorkspace(msg, wsContext)); tree != "" {
			mermaid = tree
		}
	}

	data, err := json.Marshal(mermaid)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: updateArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}
	input, err := json.Marshal(map[string]any{
		"artifact_id":       current.ID,
		"expected_revision": current.Revision,
		"data":              json.RawMessage(data),
		"fallback":          "```mermaid\n" + mermaid + "\n```",
	})
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: updateArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}

	result, err := a.executeArtifactTool(ctx, msg, updateArtifactToolName, input)
	if err != nil {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind: "error", Name: updateArtifactToolName, Preview: err.Error(),
		})
		return fmt.Sprintf("I couldn't update the Neural Canvas: %v", err), true
	}
	a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
		Kind: "result", Name: updateArtifactToolName, Preview: result,
	})
	return fmt.Sprintf("Updated the Neural Canvas Mermaid diagram to revision %d. %s", current.Revision+1, result), true
}

func (a *Agent) generateMermaidRevisionForCanvas(
	ctx context.Context,
	msg *protocol.Message,
	existing string,
	wsContext string,
	eff ai.AIProvider,
) (string, error) {
	var b strings.Builder
	b.WriteString("Revise the Mermaid diagram below per the user request. Output Mermaid source only.\n")
	b.WriteString("Keep the same architecture/nodes unless the user asks to change content.\n")
	b.WriteString("Apply style requests in Mermaid (for example monochrome via %%{init: {'theme':'base', themeVariables: {...}}}%% with black/white colors).\n")
	b.WriteString("Do NOT diagram Neural Canvas process flows.\n")
	b.WriteString("\nUSER REQUEST:\n")
	b.WriteString(strings.TrimSpace(msg.Content))
	b.WriteString("\n\nCURRENT DIAGRAM:\n")
	b.WriteString(existing)
	b.WriteString("\n")
	if strings.TrimSpace(wsContext) != "" {
		b.WriteString("\nWORKSPACE CONTEXT:\n")
		b.WriteString(wsContext)
		b.WriteString("\n")
	}
	raw, err := eff.GenerateResponse(ctx, b.String(), nil)
	if err != nil {
		return "", err
	}
	return extractMermaidSource(raw), nil
}

func mermaidSourceFromPayload(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(payload, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	var asObj map[string]any
	if err := json.Unmarshal(payload, &asObj); err == nil {
		for _, key := range []string{"source", "mermaid", "diagram", "content", "text"} {
			if v, ok := asObj[key].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return strings.TrimSpace(string(payload))
}

func (a *Agent) findRecentMermaidArtifact(msg *protocol.Message) *artifacts.Artifact {
	if a == nil || msg == nil {
		return nil
	}
	store, err := getAgentArtifactStore()
	if err != nil || store == nil {
		return nil
	}
	if id, _, _ := openArtifactFromMessageMetadata(msg); id != "" {
		if art, err := store.Get(id); err == nil && art != nil && isMermaidArtifact(art) {
			return art
		}
	}
	if id := recentMermaidArtifactID(a.channelHistory(msg.Channel), msg.ID); id != "" {
		if art, err := store.Get(id); err == nil && art != nil && isMermaidArtifact(art) {
			return art
		}
	}
	// No store.List / global fallback — same clear-history leak as markdown.
	return nil
}

func isMermaidArtifact(art *artifacts.Artifact) bool {
	if art == nil {
		return false
	}
	if art.Renderer.ID == "nj.mermaid" {
		return true
	}
	return strings.Contains(strings.ToLower(art.Renderer.MediaType), "mermaid")
}

func recentMermaidArtifactID(history []*protocol.Message, skipMsgID string) string {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 40; i-- {
		msg := history[i]
		if msg == nil || msg.ID == skipMsgID {
			continue
		}
		seen++
		ref, ok := messageArtifactReference(msg)
		if !ok || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		rid := strings.ToLower(ref.RendererID)
		media := strings.ToLower(ref.MediaType)
		if rid == "nj.mermaid" || strings.Contains(media, "mermaid") {
			return ref.ID
		}
	}
	return ""
}

func messageArtifactReference(msg *protocol.Message) (protocol.ArtifactReference, bool) {
	if msg == nil || msg.Metadata == nil {
		return protocol.ArtifactReference{}, false
	}
	raw, ok := msg.Metadata["artifact_ref"]
	if !ok || raw == nil {
		return protocol.ArtifactReference{}, false
	}
	switch v := raw.(type) {
	case protocol.ArtifactReference:
		return v, strings.TrimSpace(v.ID) != ""
	case *protocol.ArtifactReference:
		if v == nil {
			return protocol.ArtifactReference{}, false
		}
		return *v, strings.TrimSpace(v.ID) != ""
	case map[string]interface{}:
		ref := protocol.ArtifactReference{}
		if id, _ := v["id"].(string); id != "" {
			ref.ID = id
		}
		if title, _ := v["title"].(string); title != "" {
			ref.Title = title
		}
		if rid, _ := v["renderer_id"].(string); rid != "" {
			ref.RendererID = rid
		}
		if media, _ := v["media_type"].(string); media != "" {
			ref.MediaType = media
		}
		switch rev := v["revision"].(type) {
		case float64:
			ref.Revision = int64(rev)
		case int64:
			ref.Revision = rev
		case int:
			ref.Revision = int64(rev)
		}
		return ref, strings.TrimSpace(ref.ID) != ""
	default:
		data, err := json.Marshal(raw)
		if err != nil {
			return protocol.ArtifactReference{}, false
		}
		var ref protocol.ArtifactReference
		if err := json.Unmarshal(data, &ref); err != nil || strings.TrimSpace(ref.ID) == "" {
			return protocol.ArtifactReference{}, false
		}
		return ref, true
	}
}

func (a *Agent) generateMermaidForCanvas(
	ctx context.Context,
	msg *protocol.Message,
	wsContext string,
	eff ai.AIProvider,
) (string, error) {
	project := workspaceDisplayName(msg)
	var b strings.Builder
	b.WriteString("Produce ONLY a Mermaid architecture diagram for the shared application workspace.\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Output Mermaid source only — no markdown fences, no prose, no FILE_CHANGE.\n")
	b.WriteString("- Diagram the APPLICATION in the shared workspace (packages, UI, Tauri/Rust, scripts, docs).\n")
	b.WriteString("- Do NOT include Neural Junkie product concepts: Neural Canvas, Create Artifact, update_artifact, chat turns, or 'generate diagram' flows.\n")
	b.WriteString("- Do NOT invent Mermaid API / Data Store / Database layers unless those names appear as real packages in the tree.\n")
	b.WriteString("- Prefer flowchart TD. Use ONLY names from the workspace file tree / manifests below.\n")
	b.WriteString("- Use flowchart/graph node syntax only (A[Label] --> B[Label]). Never mix sequenceDiagram keywords (participant, ->>) into a flowchart.\n")
	b.WriteString("- Use top-level dirs (src, src-tauri, scripts, docs, …) and key subdirs (components, stores, hooks). Never file paths like package-lock.json.\n")
	b.WriteString("- If a label has spaces, slashes, dots, or punctuation, quote it: A[\"User / Client\"].\n")
	b.WriteString("- Keep it readable (roughly 6–20 nodes).\n")
	if project != "" {
		b.WriteString("- Project name: ")
		b.WriteString(project)
		b.WriteString("\n")
	}
	if nodes := topLevelArchitectureNodes(msg, wsContext); len(nodes) > 0 {
		b.WriteString("- Required nodes to include (use these exact directory names): ")
		b.WriteString(strings.Join(nodes, ", "))
		b.WriteString("\n")
	}
	b.WriteString("\nUSER REQUEST:\n")
	b.WriteString(strings.TrimSpace(msg.Content))
	b.WriteString("\n\n")
	if strings.TrimSpace(wsContext) != "" {
		b.WriteString("WORKSPACE ARCHITECTURE CONTEXT:\n")
		b.WriteString(wsContext)
		b.WriteString("\n")
	} else {
		b.WriteString("WARNING: No workspace context was attached. Still avoid meta canvas-process diagrams.\n")
	}
	raw, err := eff.GenerateResponse(ctx, b.String(), nil)
	if err != nil {
		return "", err
	}
	return extractMermaidSource(raw), nil
}

func (a *Agent) workspaceArchitectureContext(msg *protocol.Message, prompt string) string {
	var b strings.Builder
	AppendWorkspaceContext(&b, msg)
	path := ""
	if a != nil {
		path = a.resolveWorkspacePath(msg)
	}
	if path == "" {
		path = workspacePathFromMessage(msg)
	}
	if path != "" {
		appendWorkspaceManifestSnippets(&b, path)
	}
	if b.Len() == 0 {
		return mermaidContextSnippet(prompt)
	}
	out := b.String()
	const max = 14000
	if len(out) > max {
		return out[:max] + "\n…(truncated)"
	}
	return out
}

func appendWorkspaceManifestSnippets(b *strings.Builder, root string) {
	root = strings.TrimSpace(root)
	if root == "" {
		return
	}
	rels := []string{
		"README.md", "DOCS.md", "docs/ARCHITECTURE.md", "docs/architecture.md",
		"package.json", "go.mod", "Cargo.toml", "desktop/package.json",
	}
	b.WriteString("\n=== KEY MANIFESTS / DOCS ===\n")
	wrote := false
	for _, rel := range rels {
		p := filepath.Join(root, rel)
		data, err := os.ReadFile(p)
		if err != nil || len(data) == 0 {
			continue
		}
		wrote = true
		text := string(data)
		const maxFile = 2500
		if len(text) > maxFile {
			text = text[:maxFile] + "\n…(truncated)"
		}
		b.WriteString("### ")
		b.WriteString(rel)
		b.WriteString("\n")
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	if !wrote {
		b.WriteString("(no README/package manifests readable)\n")
	}
}

func mermaidContextSnippet(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	if i := strings.Index(prompt, ai.SystemPromptSeparator); i >= 0 {
		prompt = strings.TrimSpace(prompt[i+len(ai.SystemPromptSeparator):])
	}
	const max = 12000
	if len(prompt) > max {
		return prompt[:max] + "\n…(truncated)"
	}
	return prompt
}

func extractMermaidSource(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if m := mermaidFenceRE.FindStringSubmatch(raw); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	lines := strings.Split(raw, "\n")
	if len(lines) > 0 && strings.EqualFold(strings.TrimSpace(lines[0]), "mermaid") {
		raw = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}
	return strings.TrimSpace(raw)
}

func sanitizeMermaidSource(raw string) string {
	raw = extractMermaidSource(raw)
	if raw == "" {
		return ""
	}
	trimmed := strings.TrimSpace(raw)
	init := ""
	if m := mermaidInitRE.FindString(trimmed); m != "" {
		init = strings.TrimSpace(m)
		trimmed = strings.TrimSpace(mermaidInitRE.ReplaceAllString(trimmed, ""))
	}
	if trimmed == "" {
		return ""
	}
	if !mermaidStartRE.MatchString(trimmed) {
		if strings.Contains(trimmed, "-->") || strings.Contains(trimmed, "---") {
			trimmed = "flowchart TD\n" + trimmed
		} else {
			return ""
		}
	}
	trimmed = quoteUnsafeMermaidBracketLabels(trimmed)
	trimmed = quoteUnsafeMermaidParenLabels(trimmed)
	if init != "" {
		return init + "\n" + trimmed
	}
	return trimmed
}

var (
	mermaidBracketLabelRE = regexp.MustCompile(`(?m)(\b[\w-]+)\[([^\]"\n]*)\]`)
	mermaidParenLabelRE   = regexp.MustCompile(`(?m)(\b[\w-]+)\(([^)"\n]*)\)`)
)

func mermaidLabelNeedsQuotes(label string) bool {
	label = strings.TrimSpace(label)
	if label == "" || strings.HasPrefix(label, "\"") {
		return false
	}
	for _, r := range label {
		switch r {
		case '/', '\\', '.', '@', ':', ' ', '(', ')', '[', ']', '{', '}', ',', '|':
			return true
		}
	}
	return false
}

func escapeMermaidQuotedLabel(label string) string {
	label = strings.ReplaceAll(label, `"`, "#quot;")
	label = strings.ReplaceAll(label, "]", "")
	label = strings.ReplaceAll(label, "[", "")
	return label
}

func quoteUnsafeMermaidBracketLabels(src string) string {
	return mermaidBracketLabelRE.ReplaceAllStringFunc(src, func(match string) string {
		sub := mermaidBracketLabelRE.FindStringSubmatch(match)
		if len(sub) != 3 {
			return match
		}
		id, label := sub[1], strings.TrimSpace(sub[2])
		if !mermaidLabelNeedsQuotes(label) {
			return match
		}
		return id + `["` + escapeMermaidQuotedLabel(label) + `"]`
	})
}

func quoteUnsafeMermaidParenLabels(src string) string {
	return mermaidParenLabelRE.ReplaceAllStringFunc(src, func(match string) string {
		sub := mermaidParenLabelRE.FindStringSubmatch(match)
		if len(sub) != 3 {
			return match
		}
		id, label := sub[1], strings.TrimSpace(sub[2])
		if !mermaidLabelNeedsQuotes(label) {
			return match
		}
		return id + `("` + escapeMermaidQuotedLabel(label) + `")`
	})
}

// looksLikeMetaCanvasProcessMermaid rejects diagrams about Neural Junkie canvas plumbing
// or "generate a diagram" flows instead of the shared application architecture.
func looksLikeMetaCanvasProcessMermaid(src string) bool {
	lower := strings.ToLower(src)
	// Single strong hits: Neural Junkie product terms that do not belong in third-party app diagrams.
	for _, cue := range []string{
		"neural canvas",
		"create artifact",
		"update artifact",
		"create_artifact",
		"update_artifact",
		"nj.mermaid",
		"neural junkie",
		"generate mermaid code",
		"process user request",
	} {
		if strings.Contains(lower, cue) {
			return true
		}
	}
	cues := []string{
		"generate mermaid",
		"create mermaid",
		"mermaid code",
		"user request",
		"return diagram",
		"diagram generated",
		"end session",
		"mermaid api",
		"mermaid diagram viewer",
	}
	hits := 0
	for _, cue := range cues {
		if strings.Contains(lower, cue) {
			hits++
		}
	}
	return hits >= 2
}

// mermaidMentionsWorkspaceNodes requires the diagram to reference real workspace dirs
// so local models cannot ship a generic invented architecture.
func mermaidMentionsWorkspaceNodes(src string, nodes []string) bool {
	if len(nodes) == 0 {
		return true
	}
	lower := strings.ToLower(src)
	hits := 0
	for _, n := range nodes {
		n = strings.TrimSpace(strings.ToLower(n))
		if n == "" {
			continue
		}
		if strings.Contains(lower, n) {
			hits++
		}
	}
	need := 2
	if len(nodes) < 2 {
		need = 1
	}
	return hits >= need
}

// looksLikeInvalidMermaidArchitecture rejects common local-model mixes that fail to parse
// (e.g. flowchart/graph with sequenceDiagram "participant" / "->>" syntax).
func looksLikeInvalidMermaidArchitecture(src string) bool {
	body := strings.TrimSpace(mermaidInitRE.ReplaceAllString(strings.TrimSpace(src), ""))
	if body == "" {
		return true
	}
	first := strings.ToLower(strings.Fields(body)[0])
	lower := strings.ToLower(body)
	switch {
	case strings.HasPrefix(first, "flowchart"), first == "graph":
		if strings.Contains(lower, "participant ") || strings.Contains(lower, "->>") ||
			strings.Contains(lower, "-->>") {
			return true
		}
	}
	return false
}

func looksLikeBrokenMermaidBrackets(src string) bool {
	body := mermaidInitRE.ReplaceAllString(strings.TrimSpace(src), "")
	for _, line := range strings.Split(body, "\n") {
		open, close := strings.Count(line, "["), strings.Count(line, "]")
		if open != close {
			return true
		}
		q := strings.Count(line, `"`)
		if q%2 != 0 {
			return true
		}
	}
	return false
}

func fallbackMermaidFromWorkspace(msg *protocol.Message, wsContext string) string {
	name := workspaceDisplayName(msg)
	if name == "" {
		name = "App"
	}
	nodes := topLevelArchitectureNodes(msg, wsContext)
	if len(nodes) == 0 {
		nodes = []string{"src", "services", "data"}
	}
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	b.WriteString(fmt.Sprintf("  User[\"User / Client\"] --> App[\"%s\"]\n", escapeMermaidQuotedLabel(name)))
	srcID := ""
	tauriID := ""
	for i, n := range nodes {
		if i >= 10 {
			break
		}
		id := fmt.Sprintf("N%d", i+1)
		b.WriteString(fmt.Sprintf("  App --> %s[\"%s\"]\n", id, escapeMermaidQuotedLabel(sanitizeMermaidLabel(n))))
		switch strings.ToLower(n) {
		case "src":
			srcID = id
		case "src-tauri":
			tauriID = id
		}
	}
	// Second-level structure from disk when available (keeps diagrams project-specific).
	root := workspacePathFromMessage(msg)
	if srcID != "" && root != "" {
		for i, child := range listDirNames(filepath.Join(root, "src"), 6) {
			cid := fmt.Sprintf("S%d", i+1)
			b.WriteString(fmt.Sprintf("  %s --> %s[\"%s\"]\n", srcID, cid, escapeMermaidQuotedLabel(child)))
		}
	}
	if tauriID != "" && root != "" {
		for i, child := range listDirNames(filepath.Join(root, "src-tauri"), 5) {
			if child == "target" || child == "icons" {
				continue
			}
			cid := fmt.Sprintf("T%d", i+1)
			b.WriteString(fmt.Sprintf("  %s --> %s[\"%s\"]\n", tauriID, cid, escapeMermaidQuotedLabel(child)))
		}
	}
	return b.String()
}

func listDirNames(dir string, limit int) []string {
	entries, err := os.ReadDir(dir)
	if err != nil || limit <= 0 {
		return nil
	}
	skip := map[string]bool{
		"node_modules": true, "dist": true, "build": true, "target": true,
		".git": true, "__pycache__": true,
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if skip[name] || strings.HasPrefix(name, ".") {
			continue
		}
		if !e.IsDir() {
			continue
		}
		out = append(out, name)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func topLevelArchitectureNodes(msg *protocol.Message, wsContext string) []string {
	tree := workspaceFileTree(msg)
	if tree == "" {
		tree = wsContext
	}
	skip := map[string]bool{
		".git": true, ".github": true, ".venv": true, ".venv-icon": true, "node_modules": true,
		"dist": true, "build": true, "target": true, ".neural-junkie": true, "coverage": true,
		".idea": true, ".vscode": true, "__pycache__": true,
		"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true, "go.sum": true,
	}
	prefer := []string{
		"cmd", "internal", "desktop", "src", "src-tauri", "apps", "packages", "services",
		"server", "api", "backend", "frontend", "docs", "scripts", "public", "assets",
	}
	seen := map[string]bool{}
	var found []string
	for _, line := range strings.Split(tree, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "📁")
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "/") || strings.Contains(line, "\\") {
			// Keep only top-level entries from tree listings.
			continue
		}
		name := strings.TrimSuffix(line, "/")
		if skip[name] || skip[strings.ToLower(name)] || strings.HasPrefix(name, ".") {
			continue
		}
		// Skip lockfiles / source files — architecture nodes should be directories/packages.
		if strings.Contains(name, ".") && !strings.HasPrefix(name, "src-") {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		found = append(found, name)
	}
	var ordered []string
	for _, p := range prefer {
		for _, f := range found {
			if strings.EqualFold(f, p) {
				ordered = append(ordered, f)
			}
		}
	}
	for _, f := range found {
		dup := false
		for _, o := range ordered {
			if o == f {
				dup = true
				break
			}
		}
		if !dup {
			ordered = append(ordered, f)
		}
	}
	return ordered
}

func workspaceFileTree(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return ""
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	tree, _ := ctxMap["file_tree"].(string)
	return tree
}

func workspacePathFromMessage(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return ""
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	if path, _ := ctxMap["workspace_path"].(string); strings.TrimSpace(path) != "" {
		return strings.TrimSpace(path)
	}
	if path, _ := ctxMap["path"].(string); strings.TrimSpace(path) != "" {
		return strings.TrimSpace(path)
	}
	return ""
}

func workspaceDisplayName(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return ""
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	if name, _ := ctxMap["workspace_name"].(string); strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if name, _ := ctxMap["name"].(string); strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if path := workspacePathFromMessage(msg); path != "" {
		return filepath.Base(path)
	}
	return ""
}

func mermaidCanvasTitle(msg *protocol.Message) string {
	if name := workspaceDisplayName(msg); name != "" {
		return name + " architecture"
	}
	return "Architecture Mermaid diagram"
}

func sanitizeMermaidID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "App"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" || (out[0] >= '0' && out[0] <= '9') {
		out = "App_" + out
	}
	return out
}

func sanitizeMermaidLabel(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "]", "")
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "\"", "")
	if s == "" {
		return "component"
	}
	return s
}
