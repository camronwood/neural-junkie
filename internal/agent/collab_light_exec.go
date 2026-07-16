package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// collabLightMarkdownEligible is true when this turn should use the bounded markdown light-exec path.
func collabLightMarkdownEligible(msg *protocol.Message) bool {
	return msg != nil && msg.Type == protocol.MessageTypeCollabTask && collabTaskPrefersLightExecution(msg)
}

func collabLightTaskFromMessage(msg *protocol.Message) collaboration.CollaborationTask {
	if msg == nil {
		return collaboration.CollaborationTask{}
	}
	title, _ := msg.Metadata["task_title"].(string)
	desc, _ := msg.Metadata["task_description"].(string)
	if title == "" && desc == "" {
		desc = msg.Content
	}
	return collaboration.CollaborationTask{Title: title, Description: desc}
}

func collabLightDeliverablePath(msg *protocol.Message) string {
	task := collabLightTaskFromMessage(msg)
	paths := collaboration.ReferencedDeliverablePaths(task)
	for _, p := range paths {
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".md" || ext == ".markdown" {
			return filepath.ToSlash(p)
		}
	}
	if len(paths) > 0 {
		return filepath.ToSlash(paths[0])
	}
	return ""
}

func collabLightReadSources(wsRoot string, msg *protocol.Message) map[string]string {
	out := map[string]string{}
	paths := taskContextPathsFromMessage(msg)
	if len(paths) == 0 {
		return out
	}
	dest := filepath.ToSlash(strings.TrimSpace(collabLightDeliverablePath(msg)))
	root := strings.TrimSpace(wsRoot)
	for _, rel := range paths {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			continue
		}
		// Never ground a deliverable in its own plan-approval stub (or prior draft).
		if dest != "" && (rel == dest || strings.HasSuffix(rel, "/"+filepath.Base(dest))) {
			continue
		}
		body, err := readWorkspaceFileTail(root, rel, 32*1024)
		if err != nil || strings.TrimSpace(body) == "" {
			continue
		}
		if collaboration.IsDeliverableStubContent([]byte(body)) {
			continue
		}
		out[rel] = body
	}
	return filterLightSourcesToTaskFocus(collabLightTaskFromMessage(msg), out)
}

// filterLightSourcesToTaskFocus drops unrelated fixture files (e.g. core/sample HelloWorld)
// when the task explicitly targets resource-api / schema / docs paths.
func filterLightSourcesToTaskFocus(task collaboration.CollaborationTask, sources map[string]string) map[string]string {
	if len(sources) == 0 {
		return sources
	}
	combined := strings.ToLower(strings.TrimSpace(task.Title + " " + task.Description))
	focusHints := []string{"resource-api", "json_endpoints", "docs/tim", "schema-registration", "api_schema"}
	wantsFocus := false
	for _, h := range focusHints {
		if strings.Contains(combined, h) {
			wantsFocus = true
			break
		}
	}
	if !wantsFocus {
		return sources
	}
	filtered := map[string]string{}
	for p, body := range sources {
		pl := strings.ToLower(p)
		keep := false
		for _, h := range focusHints {
			if strings.Contains(pl, h) {
				keep = true
				break
			}
		}
		if keep {
			filtered[p] = body
		}
	}
	return filtered
}

// acceptCollabLightRewrite keeps model rewrites that pass proposal validation and stay
// grounded in allowlisted sources; otherwise the caller should keep the seed body.
func acceptCollabLightRewrite(dest, rewritten string, sources map[string]string) bool {
	rewritten = strings.TrimSpace(rewritten)
	if rewritten == "" {
		return false
	}
	if validateProposalContent(dest, rewritten) != nil {
		return false
	}
	if collaboration.IsDeliverableStubContent([]byte(rewritten)) {
		return false
	}
	if len(sources) == 0 {
		return true
	}
	lower := strings.ToLower(rewritten)
	for path, body := range sources {
		base := strings.ToLower(filepath.Base(path))
		pl := strings.ToLower(path)
		if base != "" && strings.Contains(lower, base) {
			return true
		}
		if pl != "" && strings.Contains(lower, pl) {
			return true
		}
		// Distinctive snippet from source body (skip tiny/boilerplate lines).
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			if len(line) < 12 {
				continue
			}
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "package ") {
				continue
			}
			if strings.Contains(lower, strings.ToLower(line)) {
				return true
			}
			if len(line) > 40 {
				snippet := strings.ToLower(line[:40])
				if strings.Contains(lower, snippet) {
					return true
				}
			}
			break
		}
	}
	return false
}

// buildCollabLightMarkdownBody synthesizes a grounded markdown deliverable from allowlisted sources.
func buildCollabLightMarkdownBody(task collaboration.CollaborationTask, sources map[string]string) string {
	var b strings.Builder
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = "Findings"
	}
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")
	if desc := strings.TrimSpace(task.Description); desc != "" {
		b.WriteString(desc)
		b.WriteString("\n\n")
	}
	if len(sources) == 0 {
		b.WriteString("- No allowlisted source content was available; fill in findings from workspace reads.\n")
		return b.String()
	}
	b.WriteString("## Sources\n\n")
	for path, body := range sources {
		b.WriteString("### `")
		b.WriteString(path)
		b.WriteString("`\n\n")
		snippet := strings.TrimSpace(body)
		if len(snippet) > 1200 {
			snippet = snippet[:1200] + "\n…"
		}
		b.WriteString(snippet)
		b.WriteString("\n\n")
	}
	b.WriteString("## Findings\n\n")
	n := 0
	for path := range sources {
		n++
		b.WriteString(fmt.Sprintf("- Cited `%s` as an allowlisted source for this deliverable.\n", path))
		if n >= 5 {
			break
		}
	}
	return b.String()
}

// runCollabLightMarkdownExecution is the positive light path: read allowlisted sources →
// propose [FILE_CHANGE] for the markdown deliverable → TASK_STATUS.
func (a *Agent) runCollabLightMarkdownExecution(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, bool, []string, error) {
	if a == nil || msg == nil {
		return "", false, nil, fmt.Errorf("light markdown exec: missing agent or message")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Sprintf("Light markdown exec aborted: %v", err), false, nil, nil
	}
	if err := a.refuseInactiveCollabProposal(msg); err != nil {
		return fmt.Sprintf("Light markdown exec skipped: %v", err), false, nil, nil
	}
	dest := collabLightDeliverablePath(msg)
	if dest == "" {
		return "Markdown light-exec could not infer a deliverable `.md` path from the task.", false, nil, nil
	}
	task := collabLightTaskFromMessage(msg)
	ws := a.resolveWorkspacePath(msg)
	sources := collabLightReadSources(ws, msg)
	seed := buildCollabLightMarkdownBody(task, sources)
	body := seed

	// Prefer a short model rewrite when valid and grounded; otherwise keep the seed.
	if eff != nil {
		if rewritten, err := a.generateCollabLightMarkdownWithProvider(ctx, msg, eff, dest, seed, sources); err == nil {
			if acceptCollabLightRewrite(dest, rewritten, sources) {
				body = rewritten
			} else if strings.TrimSpace(rewritten) != "" {
				log.Printf("[%s] light markdown rewrite rejected for %s; keeping seed body", a.Info.Name, dest)
			}
		}
	}
	// Cancel mid-rewrite must not fall through to a late FILE_CHANGE propose.
	if err := ctx.Err(); err != nil {
		return fmt.Sprintf("Light markdown exec aborted before propose: %v", err), false, nil, nil
	}
	if err := a.refuseInactiveCollabProposal(msg); err != nil {
		return fmt.Sprintf("Light markdown exec skipped before propose: %v", err), false, nil, nil
	}

	abs := filepath.Join(ws, filepath.FromSlash(dest))
	existing := ""
	if st, err := os.Stat(abs); err == nil && !st.IsDir() {
		if raw, err := os.ReadFile(abs); err == nil {
			existing = string(raw)
		}
	}
	var propErr error
	if existing != "" {
		propErr = a.proposeFileEditInChannel(ctx, msg.Channel, dest, existing, body, msg)
	} else {
		propErr = a.proposeFileCreateInChannel(ctx, msg.Channel, dest, body, msg)
	}
	if propErr != nil {
		log.Printf("[%s] light markdown propose %s: %v", a.Info.Name, dest, propErr)
		return fmt.Sprintf("Light markdown exec failed to propose `%s`: %v", dest, propErr), false, nil, nil
	}

	state := &ImplementationSessionState{
		FilesChanged:    []string{dest},
		RegisteredFiles: []string{dest},
		ProposedCount:   1,
	}
	summary := fmt.Sprintf("Light markdown deliverable proposed: `%s` (grounded in %d allowlisted source(s)).", dest, len(sources))
	out := appendCollabExecutionTaskStatus(summary, msg, state, true)
	return out, true, []string{dest}, nil
}

func (a *Agent) generateCollabLightMarkdownWithProvider(
	ctx context.Context,
	msg *protocol.Message,
	eff ai.AIProvider,
	dest string,
	seed string,
	sources map[string]string,
) (string, error) {
	if a == nil || eff == nil || msg == nil {
		return "", fmt.Errorf("missing provider")
	}
	var prompt strings.Builder
	prompt.WriteString("Write the full markdown contents for ")
	prompt.WriteString(dest)
	prompt.WriteString(" only. Use the allowlisted sources below. Do not invent paths outside the allowlist.\n\n")
	prompt.WriteString("SEED DRAFT:\n")
	prompt.WriteString(seed)
	prompt.WriteString("\n")
	_ = sources
	// Reuse chat generation with a synthetic content nudge; keep this optional.
	nudge := protocol.NewMessage(protocol.MessageTypeCollabTask, msg.Channel, msg.From, prompt.String())
	nudge.Metadata = msg.Metadata
	nudge.SetCollaborationID(msg.GetCollaborationID())
	nudge.SetCollaborationPhase(msg.GetCollaborationPhase())
	nudge.SetTaskID(msg.GetTaskID())
	return a.generateResponse(ctx, nudge, eff)
}
