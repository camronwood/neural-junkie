package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	looseFileChangeSameLineRE = regexp.MustCompile(`(?i)\[FILE_CHANGE\]\s*([^\s\n\[\]` + "`" + `]+)`)
)

// collabFileChangeParseEnabled is true on collaboration execution turns that must ship files.
func collabFileChangeParseEnabled(msg *protocol.Message) bool {
	if msg == nil || strings.TrimSpace(msg.GetCollaborationID()) == "" {
		return false
	}
	if strings.TrimSpace(msg.GetCollaborationPhase()) != "executing" {
		return false
	}
	switch msg.Type {
	case protocol.MessageTypeCollabTask:
		return true
	case protocol.MessageTypeCollabDiscussion:
		return strings.TrimSpace(msg.GetTaskID()) != ""
	default:
		return false
	}
}

func looseFileChangeParseEnabled(msg *protocol.Message) bool {
	return legacyFileChangeParseEnabled() || collabFileChangeParseEnabled(msg)
}

// parseAllLooseFileChanges extracts every non-canonical [FILE_CHANGE] block in a reply.
func parseAllLooseFileChanges(response string) []*fileChangeDirective {
	if !strings.Contains(strings.ToLower(response), "[file_change]") {
		return nil
	}
	lower := strings.ToLower(response)
	var out []*fileChangeDirective
	seen := make(map[string]bool)
	for i := 0; i < len(response); {
		idx := strings.Index(lower[i:], "[file_change]")
		if idx < 0 {
			break
		}
		abs := i + idx
		tail := response[abs:]
		if fileChangeBlockRegex.MatchString(tail) {
			if m := fileChangeBlockRegex.FindStringIndex(tail); len(m) == 2 {
				i = abs + m[1]
				continue
			}
		}
		path := resolveLooseFileChangePath(tail)
		if path == "" {
			i = abs + len("[file_change]")
			continue
		}
		segment := tail
		if nextIdx := strings.Index(strings.ToLower(tail[len("[file_change]"):]), "[file_change]"); nextIdx >= 0 {
			segment = tail[:len("[file_change]")+nextIdx]
		}
		body := stripEditorLineNumberPrefixes(extractAnyCodeFenceContent(segment))
		if strings.TrimSpace(body) == "" {
			body = stripEditorLineNumberPrefixes(looseFileChangeBodyAfterPath(segment, path))
		}
		if strings.TrimSpace(body) == "" {
			i = abs + len("[file_change]")
			continue
		}
		key := path + "\x00" + body
		if !seen[key] {
			seen[key] = true
			out = append(out, &fileChangeDirective{
				Operation:  "create",
				Path:       path,
				NewContent: body,
			})
		}
		if nextIdx := strings.Index(lower[abs+len("[file_change]"):], "[file_change]"); nextIdx >= 0 {
			i = abs + len("[file_change]") + nextIdx
		} else {
			break
		}
	}
	return out
}

// parseLooseFileChange handles agent variants that mention [FILE_CHANGE] without the
// canonical [/FILE_CHANGE] wrapper (common in collaboration task replies).
func parseLooseFileChange(response string) (*fileChangeDirective, bool) {
	if !strings.Contains(strings.ToLower(response), "[file_change]") {
		return nil, false
	}
	if fileChangeBlockRegex.MatchString(response) {
		return nil, false
	}

	idx := strings.Index(strings.ToLower(response), "[file_change]")
	tail := response[idx:]

	path := resolveLooseFileChangePath(tail)
	if path == "" {
		return nil, false
	}

	body := strings.TrimSpace(extractAnyCodeFenceContent(response))
	if body == "" {
		body = looseFileChangeBodyAfterPath(tail, path)
	}
	body = stripEditorLineNumberPrefixes(body)
	if strings.TrimSpace(body) == "" {
		return nil, false
	}

	return &fileChangeDirective{
		Operation:  "create",
		Path:       path,
		NewContent: body,
	}, true
}

func looseFileChangeBodyAfterPath(tail, path string) string {
	if body := extractLooseFileChangeContentField(tail); body != "" {
		return body
	}
	lines := strings.Split(tail, "\n")
	var bodyLines []string
	pastHeader := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !pastHeader {
			if strings.Contains(strings.ToLower(trimmed), "[file_change]") {
				re := regexp.MustCompile(`(?i)\[FILE_CHANGE\]\s*` + regexp.QuoteMeta(path) + `\s*(.*)`)
				if m := re.FindStringSubmatch(trimmed); len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
					rest := strings.TrimSpace(m[1])
					// Same-line residue is often "operation: create path: …" — not body.
					if !looksLikeFileChangeHeaderResidue(rest) {
						bodyLines = append(bodyLines, rest)
					}
				}
				pastHeader = true
			}
			continue
		}
		low := strings.ToLower(trimmed)
		if strings.HasPrefix(low, "path:") || strings.HasPrefix(low, "operation:") ||
			strings.HasPrefix(low, "old_path:") || strings.HasPrefix(low, "new_path:") ||
			strings.HasPrefix(low, "content:") ||
			trimmed == "---" || trimmed == "```" {
			continue
		}
		if strings.HasPrefix(low, "task_status:") {
			break
		}
		bodyLines = append(bodyLines, line)
	}
	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
	if looksLikeFileChangeDirectivePayload(body) {
		return ""
	}
	return body
}

func extractLooseFileChangeContentField(tail string) string {
	reInline := regexp.MustCompile(`(?is)\bcontent:\s*(?:"((?:\\.|[^"])*)"|'((?:\\.|[^'])*)'|(.*))$`)
	for _, line := range strings.Split(tail, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(strings.ToLower(trimmed), "content:") {
			continue
		}
		m := reInline.FindStringSubmatch(trimmed)
		if len(m) < 4 {
			continue
		}
		body := strings.TrimSpace(m[1] + m[2] + m[3])
		body = strings.ReplaceAll(body, `\"`, `"`)
		if body == "" || looksLikeFileChangeHeaderResidue(body) || looksLikeFileChangeDirectivePayload(body) {
			continue
		}
		return body
	}
	return ""
}

func looksLikeFileChangeHeaderResidue(s string) bool {
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "" {
		return true
	}
	if strings.HasPrefix(low, "operation:") || strings.HasPrefix(low, "path:") || strings.HasPrefix(low, "content:") {
		return true
	}
	if strings.Contains(low, "operation:") && strings.Contains(low, "path:") {
		return true
	}
	return false
}

func stripLooseFileChangeBlock(response string) string {
	idx := strings.Index(strings.ToLower(response), "[file_change]")
	if idx < 0 {
		return strings.TrimSpace(response)
	}
	prefix := strings.TrimSpace(response[:idx])
	suffix := response[idx:]
	if i := strings.Index(strings.ToLower(suffix), "task_status:"); i >= 0 {
		suffix = suffix[i:]
	} else {
		suffix = ""
	}
	out := strings.TrimSpace(prefix)
	if suffix != "" {
		if out != "" {
			out += "\n\n"
		}
		out += strings.TrimSpace(suffix)
	}
	return strings.TrimSpace(out)
}
