package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/artifacts"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	mermaidFenceRE = regexp.MustCompile("(?is)```(?:mermaid)?\\s*([\\s\\S]*?)```")
	mermaidStartRE = regexp.MustCompile(`(?i)^(graph|flowchart|sequenceDiagram|classDiagram|stateDiagram|erDiagram|journey|gantt|pie|mindmap|timeline|quadrantChart|sankey|xychart|block-beta)\b`)
	mermaidInitRE  = regexp.MustCompile(`(?is)^%%\{init:.*?\}%%\s*`)
)

// wantsMermaidCanvas reports create-style asks that should ship a new nj.mermaid.
// Revisions are selected structurally (ActionArtifact + open mermaid in channel).
func wantsMermaidCanvas(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" || !UserRequestsArtifact(c) {
		return false
	}
	if strings.Contains(c, "mermaid") || strings.Contains(c, "diagram") {
		return true
	}
	return strings.Contains(c, "architecture") && strings.Contains(c, "canvas")
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
	createAsk := wantsMermaidCanvas(msg.Content)
	prior := a.findRecentMermaidArtifact(msg)
	artifactTurn := false
	if decision, ok := protocol.ExtractTurnDecision(msg); ok && decision.Action == intent.ActionArtifact {
		artifactTurn = true
	} else if messageHasArtifactAction(msg) || neuralCanvasDeliverableTurn(msg) {
		artifactTurn = true
	}
	if !createAsk && !(artifactTurn && prior != nil) {
		return "", false
	}

	// Ensure create/update_artifact is allowed even if a misrouted edit/run stamp lingered.
	if neuralCanvasDeliverableTurn(msg) || createAsk {
		if decision, ok := protocol.ExtractTurnDecision(msg); ok && decision.Action != intent.ActionArtifact {
			if !neuralCanvasIsSecondaryToCodeChange(msg.Content) {
				decision.Action = intent.ActionArtifact
				decision.RequestedAction = intent.ActionArtifact
				decision.Mutation = intent.MutationExternal
				decision.PolicyOverrides = append(decision.PolicyOverrides, "explicit_neural_canvas")
				_ = protocol.StampTurnDecision(msg, decision)
			}
		}
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
	if id := recentMermaidArtifactID(a.channelHistory(msg.Channel), msg.ID); id != "" {
		if art, err := store.Get(id); err == nil && art != nil && isMermaidArtifact(art) {
			return art
		}
	}
	items, err := store.List(artifacts.Filter{ChannelID: messageChannel(msg), RendererID: "nj.mermaid"})
	if err != nil || len(items) == 0 {
		items, err = store.List(artifacts.Filter{RendererID: "nj.mermaid"})
		if err != nil || len(items) == 0 {
			return nil
		}
	}
	best := items[0]
	for i := 1; i < len(items); i++ {
		if items[i].UpdatedAt.After(best.UpdatedAt) {
			best = items[i]
		}
	}
	return &best
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
