package agent

import (
	"regexp"
	"strings"
)

var (
	looseFileChangeSameLineRE = regexp.MustCompile(`(?i)\[FILE_CHANGE\]\s*([^\s\n\[\]` + "`" + `]+)`)
)

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
	lines := strings.Split(tail, "\n")
	var bodyLines []string
	pastHeader := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !pastHeader {
			if strings.Contains(strings.ToLower(trimmed), "[file_change]") {
				re := regexp.MustCompile(`(?i)\[FILE_CHANGE\]\s*` + regexp.QuoteMeta(path) + `\s*(.*)`)
				if m := re.FindStringSubmatch(trimmed); len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
					bodyLines = append(bodyLines, strings.TrimSpace(m[1]))
				}
				pastHeader = true
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "path:") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "task_status:") {
			break
		}
		bodyLines = append(bodyLines, line)
	}
	return strings.TrimSpace(strings.Join(bodyLines, "\n"))
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
