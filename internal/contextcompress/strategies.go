package contextcompress

import (
	"strings"
)

func compressListLines(toolName, raw string, maxBytes, topN int) (body string, strategy string) {
	lines := strings.Split(raw, "\n")
	nonEmpty := 0
	for _, ln := range lines {
		if strings.TrimSpace(ln) != "" {
			nonEmpty++
		}
	}
	if nonEmpty <= topN && len(raw) <= maxBytes {
		return raw, StrategyNone
	}
	var kept []string
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		kept = append(kept, ln)
		if len(kept) >= topN {
			break
		}
	}
	strategy = StrategyGrepTopN
	if toolName == "glob_file_search" || toolName == "list_dir" || toolName == "list_key_files" {
		strategy = StrategyListTopN
	}
	if toolName == "semantic_search" || toolName == "search_codebase" || toolName == "search_by_path" {
		strategy = StrategySearchTopN
	}
	header := strings.Builder{}
	header.WriteString("Showing top ")
	header.WriteString(itoa(len(kept)))
	header.WriteString(" of ")
	header.WriteString(itoa(nonEmpty))
	header.WriteString(" lines (tool: ")
	header.WriteString(toolName)
	header.WriteString(").\n")
	body = header.String() + strings.Join(kept, "\n")
	if len(body) > maxBytes {
		body = body[:maxBytes] + "\n…(compressed preview truncated)\n"
	}
	return body, strategy
}

func compressReadFile(raw string, maxBytes int) (body string, strategy string) {
	if len(raw) <= maxBytes {
		return raw, StrategyNone
	}
	lines := strings.Split(raw, "\n")
	var sigs, head, tail []string
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if trim == "" {
			continue
		}
		if isSignatureLine(trim) {
			sigs = append(sigs, ln)
		}
	}
	for i := 0; i < len(lines) && i < defaultReadHeadLines; i++ {
		head = append(head, lines[i])
	}
	start := len(lines) - defaultReadTailLines
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		tail = append(tail, lines[i])
	}
	var b strings.Builder
	b.WriteString("File preview (signatures + head/tail). Full content available via nj_retrieve_context.\n")
	if len(sigs) > 0 {
		b.WriteString("\n--- signatures ---\n")
		b.WriteString(strings.Join(sigs, "\n"))
	}
	b.WriteString("\n--- head ---\n")
	b.WriteString(strings.Join(head, "\n"))
	b.WriteString("\n--- tail ---\n")
	b.WriteString(strings.Join(tail, "\n"))
	body = b.String()
	if len(body) > maxBytes {
		body = body[:maxBytes] + "\n…(preview truncated)\n"
	}
	return body, StrategyReadPreview
}

func compressLogOutput(raw string, maxBytes int) (body string, strategy string) {
	if len(raw) <= maxBytes {
		return raw, StrategyNone
	}
	lines := strings.Split(raw, "\n")
	start := len(lines) - defaultLogTailLines
	if start < 0 {
		start = 0
	}
	tail := lines[start:]
	var b strings.Builder
	b.WriteString("Log/command output tail (last ")
	b.WriteString(itoa(len(tail)))
	b.WriteString(" lines). Full output via nj_retrieve_context.\n")
	b.WriteString(strings.Join(tail, "\n"))
	body = b.String()
	if len(body) > maxBytes {
		body = body[:maxBytes] + "\n…(tail truncated)\n"
	}
	return body, StrategyLogTail
}

func compressGeneric(raw string, maxBytes int) (body string, strategy string) {
	if len(raw) <= maxBytes {
		return raw, StrategyNone
	}
	head := maxBytes * 2 / 3
	if head < 512 {
		head = 512
	}
	if head > len(raw) {
		head = len(raw)
	}
	body = raw[:head] + "\n…(content compressed; use nj_retrieve_context for full text)\n"
	return body, StrategyGeneric
}

func isSignatureLine(trim string) bool {
	if strings.HasPrefix(trim, "package ") ||
		strings.HasPrefix(trim, "import ") ||
		strings.HasPrefix(trim, "func ") ||
		strings.HasPrefix(trim, "type ") ||
		strings.HasPrefix(trim, "class ") ||
		strings.HasPrefix(trim, "interface ") ||
		strings.HasPrefix(trim, "export ") {
		return true
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
