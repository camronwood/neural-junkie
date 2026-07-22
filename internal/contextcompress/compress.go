package contextcompress

import (
	"strings"
)

// Strategy names stamped in metadata and ref markers.
const (
	StrategyNone        = "none"
	StrategyGrepTopN    = "grep_top_n"
	StrategyListTopN    = "list_top_n"
	StrategySearchTopN  = "search_top_n"
	StrategyReadPreview = "read_preview"
	StrategyLogTail     = "log_tail"
	StrategyGeneric     = "generic"
	StrategySection     = "section_ccr"
)

// Result is the output of compression.
type Result struct {
	Text            string
	Ref             string
	OriginalBytes   int
	CompressedBytes int
	Strategy        string
	Stored          bool
}

// CompressToolResult applies type-aware compression to MCP tool output.
func CompressToolResult(store *Store, toolName, channelID, callID, raw string, opts Options) Result {
	opts = opts.normalized()
	originalBytes := len(raw)
	out := Result{
		Text:          raw,
		OriginalBytes: originalBytes,
		Strategy:      StrategyNone,
	}
	if !opts.Enabled || raw == "" {
		out.CompressedBytes = originalBytes
		return out
	}
	if IsErrorPrefix(raw) {
		out.CompressedBytes = originalBytes
		return out
	}

	maxBytes := opts.MaxToolBytes
	if len(raw) <= maxBytes {
		out.CompressedBytes = originalBytes
		return out
	}

	tool := strings.ToLower(strings.TrimSpace(toolName))
	var body, strategy string
	switch {
	case isListTool(tool):
		body, strategy = compressListLines(tool, raw, maxBytes, opts.ListTopN)
	case isReadTool(tool):
		body, strategy = compressReadFile(raw, maxBytes)
	case isLogTool(tool):
		body, strategy = compressLogOutput(raw, maxBytes)
	case strings.HasPrefix(tool, "summarize_"):
		body, strategy = compressGeneric(raw, maxBytes)
	default:
		body, strategy = compressGeneric(raw, maxBytes)
	}

	if strategy == StrategyNone {
		out.CompressedBytes = len(body)
		out.Text = body
		return out
	}

	ref := ""
	if store != nil {
		ref = store.Put(channelID, callID, toolName, []byte(raw))
	}
	marker := formatRefMarker(originalBytes, len(body), strategy, ref)
	out.Text = body + "\n" + marker
	out.Ref = ref
	out.Strategy = strategy
	out.CompressedBytes = len(out.Text)
	out.Stored = ref != ""
	return out
}

// CompressSection compresses a prompt section and stores the original when over maxBytes.
func CompressSection(store *Store, sectionLabel, channelID, callID, raw string, maxBytes int, opts Options) Result {
	return CompressSectionWithRetrieval(store, sectionLabel, channelID, callID, raw, maxBytes, opts, true)
}

// CompressSectionWithRetrieval emits a reversible CCR marker only when the
// receiving model can call nj_retrieve_context. Otherwise it returns the same
// deterministic excerpt without advertising or storing an unusable reference.
func CompressSectionWithRetrieval(store *Store, sectionLabel, channelID, callID, raw string, maxBytes int, opts Options, canRetrieve bool) Result {
	opts = opts.normalized()
	originalBytes := len(raw)
	out := Result{
		Text:          raw,
		OriginalBytes: originalBytes,
		Strategy:      StrategyNone,
	}
	if !opts.Enabled || raw == "" || len(raw) <= maxBytes {
		out.CompressedBytes = originalBytes
		return out
	}
	body, strategy := compressGeneric(raw, maxBytes)
	strategy = StrategySection
	if !canRetrieve {
		body = strings.ReplaceAll(
			body,
			"…(content compressed; use nj_retrieve_context for full text)",
			"…(content excerpted; retrieval unavailable)",
		)
		marker := formatExcerptMarker(originalBytes, len(body), strategy)
		out.Text = body + "\n" + marker
		out.Strategy = strategy
		out.CompressedBytes = len(out.Text)
		return out
	}
	ref := ""
	if store != nil {
		ref = store.Put(channelID, callID, sectionLabel, []byte(raw))
	}
	marker := formatRefMarker(originalBytes, len(body), strategy, ref)
	out.Text = body + "\n" + marker
	out.Ref = ref
	out.Strategy = strategy
	out.CompressedBytes = len(out.Text)
	out.Stored = ref != ""
	return out
}

func formatExcerptMarker(originalBytes, compressedBytes int, strategy string) string {
	return "[excerpted: " + itoa(originalBytes) + "→" + itoa(compressedBytes) +
		" bytes, strategy=" + strategy + ", retrieval unavailable.]"
}

func formatRefMarker(originalBytes, compressedBytes int, strategy, ref string) string {
	var b strings.Builder
	b.WriteString("[compressed: ")
	b.WriteString(itoa(originalBytes))
	b.WriteString("→")
	b.WriteString(itoa(compressedBytes))
	b.WriteString(" bytes, strategy=")
	b.WriteString(strategy)
	if ref != "" {
		b.WriteString(", ref=")
		b.WriteString(ref)
	}
	b.WriteString(". Use nj_retrieve_context to expand.]")
	return b.String()
}

func isListTool(tool string) bool {
	switch tool {
	case "grep", "glob_file_search", "list_dir", "semantic_search",
		"search_codebase", "search_by_path", "list_key_files":
		return true
	default:
		return false
	}
}

func isReadTool(tool string) bool {
	switch tool {
	case "read_file", "get_file_content":
		return true
	default:
		return false
	}
}

func isLogTool(tool string) bool {
	switch tool {
	case "run_command", "check_pod_logs", "kubectl_query", "cargo_clippy",
		"cargo_test", "run_typescript_check", "run_eslint":
		return true
	default:
		return false
	}
}

// IsErrorPrefix reports tool error strings that should not be compressed.
func IsErrorPrefix(text string) bool {
	return strings.HasPrefix(text, "ERROR: ") || strings.HasPrefix(text, "Error in ")
}
