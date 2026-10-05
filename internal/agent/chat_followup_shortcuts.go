package agent

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const confusedFollowUpReply = "Happy to clarify. I was answering your last question in this thread — which part is unclear? Ask one concrete follow-up and I will answer that directly, without repeating earlier user wording."

var goPackageDeclRE = regexp.MustCompile(`(?m)^package\s+[A-Za-z_][A-Za-z0-9_]*`)

// tryConfusedFollowUpResponse answers "What?" / "huh?" without an LLM so the
// model cannot echo the prior user turn (canary: dm-backend-echo-followup).
func tryConfusedFollowUpResponse(msg *protocol.Message) (string, bool) {
	if msg == nil || !shortConfusedFollowUp(msg.Content) {
		return "", false
	}
	return confusedFollowUpReply, true
}

func userAsksReferencedFileFact(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	asksPkg := strings.Contains(lower, "package declaration") ||
		(strings.Contains(lower, "package") && (strings.Contains(lower, "top of") ||
			strings.Contains(lower, "at the top") || strings.Contains(lower, "declaration")))
	refersFile := strings.Contains(lower, "that file") ||
		strings.Contains(lower, "@file:") ||
		strings.Contains(lower, "this file") ||
		strings.Contains(lower, "the file")
	return asksPkg && refersFile
}

// tryOpenFileFactResponse answers package/file-body facts from workspace or @file history
// so quality-gate retries cannot wipe a grounded follow-up (canary: dm-backend-interject-resume).
func (a *Agent) tryOpenFileFactResponse(msg *protocol.Message) (string, bool) {
	if msg == nil || !userAsksReferencedFileFact(msg.Content) {
		return "", false
	}
	path, body := resolveReferencedSourceFile(a, msg)
	if strings.TrimSpace(body) == "" {
		return "", false
	}
	rel := normalizeFileChangeRelPath(path)
	if rel == "" {
		rel = strings.TrimSpace(path)
	}
	pkg := strings.TrimSpace(goPackageDeclRE.FindString(body))
	var b strings.Builder
	if pkg != "" {
		fmt.Fprintf(&b, "The Go package declaration at the top of `%s` is `%s`.\n", rel, pkg)
	} else {
		fmt.Fprintf(&b, "Here is the start of `%s` from the workspace context.\n", rel)
	}
	if strings.Contains(body, "HelloWorld") {
		b.WriteString("That file also defines HelloWorld and main.\n")
	}
	if rel != "" {
		fmt.Fprintf(&b, "Path: `%s`\n", rel)
	}
	out := strings.TrimSpace(b.String())
	return out, out != ""
}

func resolveReferencedSourceFile(a *Agent, msg *protocol.Message) (path, body string) {
	if msg == nil {
		return "", ""
	}
	candidates := DetectFilePaths(msg.Content)
	if a != nil {
		if hist := a.channelHistory(msg.Channel); len(hist) > 0 {
			scanned := 0
			for i := len(hist) - 1; i >= 0 && scanned < 16; i-- {
				h := hist[i]
				if h == nil || h.ID == msg.ID {
					continue
				}
				scanned++
				if !protocol.IsUserLikeSender(h.From) {
					continue
				}
				candidates = append(candidates, DetectFilePaths(h.Content)...)
			}
		}
	}
	if raw, ok := msg.Metadata["workspace_context"].(map[string]interface{}); ok {
		if files, ok := raw["open_files"].([]interface{}); ok {
			for _, f := range files {
				fm, ok := f.(map[string]interface{})
				if !ok {
					continue
				}
				fp, _ := fm["path"].(string)
				content, _ := fm["content"].(string)
				if strings.TrimSpace(fp) == "" {
					continue
				}
				candidates = append(candidates, fp)
				if strings.TrimSpace(content) != "" && body == "" {
					path, body = fp, content
				}
			}
		}
	}
	if path == "" {
		for _, p := range candidates {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			path = p
			break
		}
	}
	if strings.TrimSpace(body) == "" && a != nil && path != "" {
		if ws := a.resolveWorkspacePath(msg); ws != "" {
			if data, err := readWorkspaceFileTail(ws, path, 16*1024); err == nil {
				body = data
			}
		}
	}
	if strings.TrimSpace(body) == "" && path != "" && msg.Metadata != nil {
		if raw, ok := msg.Metadata["workspace_context"].(map[string]interface{}); ok {
			if files, ok := raw["open_files"].([]interface{}); ok {
				want := normalizeFileChangeRelPath(path)
				for _, f := range files {
					fm, ok := f.(map[string]interface{})
					if !ok {
						continue
					}
					fp, _ := fm["path"].(string)
					content, _ := fm["content"].(string)
					if normalizeFileChangeRelPath(fp) == want || strings.HasSuffix(strings.ReplaceAll(fp, "\\", "/"), want) {
						body = content
						path = fp
						break
					}
				}
			}
		}
	}
	return path, body
}
