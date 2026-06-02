package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/scananalysis"
)

func requestedBiologyScanTool(content string) string {
	lower := strings.ToLower(strings.TrimSpace(content))
	switch {
	case strings.Contains(lower, "summarize_scan_analysis"):
		return "summarize_scan_analysis"
	case strings.Contains(lower, "summarize_scan_summary"):
		return "summarize_scan_summary"
	default:
		return ""
	}
}

func userAsksAboutOpenScanFile(content string) bool {
	if userRequestsEditorDocumentReview(content) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(content))
	if strings.Contains(lower, "what") && strings.Contains(lower, "see") &&
		(strings.Contains(lower, "open") || strings.Contains(lower, "file")) {
		return true
	}
	for _, m := range []string{
		"summarize", "scan analysis", "scan summary", "qc", "quality check",
		"results.json", "summary_report.csv",
	} {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func workspaceContextHasScanAnalysis(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return false
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return false
	}
	if scan, ok := ctxMap["scan_analysis"].(map[string]interface{}); ok && len(scan) > 0 {
		if dir, _ := scan["analysis_dir"].(string); strings.TrimSpace(dir) != "" {
			return true
		}
	}
	if openFilesHaveScanDir(ctxMap, "scan_analysis_dir") {
		return true
	}
	return openFilesHaveScanAnalysisExportPath(ctxMap)
}

func openFilesHaveScanAnalysisExportPath(ctxMap map[string]interface{}) bool {
	if editor, ok := ctxMap["active_editor"].(map[string]interface{}); ok {
		if path, _ := editor["path"].(string); scananalysis.IsAnalysisExport(strings.TrimSpace(path)) {
			return true
		}
	}
	files, ok := ctxMap["open_files"].([]interface{})
	if !ok {
		return false
	}
	for _, f := range files {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		path, _ := fm["path"].(string)
		if scananalysis.IsAnalysisExport(strings.TrimSpace(path)) {
			return true
		}
	}
	return false
}

func openFilesHaveScanDir(ctxMap map[string]interface{}, key string) bool {
	files, ok := ctxMap["open_files"].([]interface{})
	if !ok {
		return false
	}
	for _, f := range files {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		dir, _ := fm[key].(string)
		if strings.TrimSpace(dir) != "" {
			return true
		}
	}
	if editor, ok := ctxMap["active_editor"].(map[string]interface{}); ok {
		dir, _ := editor[key].(string)
		return strings.TrimSpace(dir) != ""
	}
	return false
}

// requestedBiologyScanToolFromTurn resolves the scan MCP tool from this message or a recent user turn.
func (a *Agent) requestedBiologyScanToolFromTurn(msg *protocol.Message) string {
	if msg == nil {
		return ""
	}
	if name := requestedBiologyScanTool(msg.Content); name != "" {
		return name
	}
	lower := strings.ToLower(strings.TrimSpace(msg.Content))
	if !userRequestsEditorDocumentReview(msg.Content) &&
		!strings.Contains(lower, "editor") &&
		!strings.Contains(lower, "have open") &&
		!strings.Contains(lower, "haveopen") {
		return ""
	}
	for _, h := range a.recentUserMessages(msg.Channel, 12) {
		if name := requestedBiologyScanTool(h.Content); name != "" {
			return name
		}
	}
	return ""
}

// resolveBiologyScanToolForTurn picks the scan MCP tool for this user turn.
func (a *Agent) resolveBiologyScanToolForTurn(msg *protocol.Message) string {
	if name := a.requestedBiologyScanToolFromTurn(msg); name != "" {
		return name
	}
	if !userAsksAboutOpenScanFile(msg.Content) {
		return ""
	}
	if workspaceContextHasScanAnalysis(msg) {
		return "summarize_scan_analysis"
	}
	if workspaceContextHasScanSummary(msg) {
		return "summarize_scan_summary"
	}
	return ""
}

func (a *Agent) recentUserMessages(channel string, limit int) []*protocol.Message {
	if a.Hub == nil || limit <= 0 {
		return nil
	}
	msgs, err := a.Hub.GetMessages(channel, limit*4)
	if err != nil {
		return nil
	}
	var out []*protocol.Message
	for i := len(msgs) - 1; i >= 0 && len(out) < limit; i-- {
		m := msgs[i]
		if m == nil {
			continue
		}
		if m.Type != protocol.MessageTypeQuestion && m.Type != protocol.MessageTypeChat {
			continue
		}
		fromType := strings.ToLower(strings.TrimSpace(string(m.From.Type)))
		if fromType != "" && fromType != "human" && fromType != "user" {
			continue
		}
		out = append(out, m)
	}
	return out
}

// tryBiologyScanToolShortcut runs scan MCP tools directly when the user refers to the open
// scan viewer or names the tool — without asking the LLM for a path.
func (a *Agent) tryBiologyScanToolShortcut(ctx context.Context, msg *protocol.Message) (string, bool) {
	if a.Info.Type != protocol.AgentTypeBiology {
		return "", false
	}
	toolName := a.resolveBiologyScanToolForTurn(msg)
	if toolName == "" {
		return "", false
	}
	if !a.agentToolNames()[toolName] {
		return "", false
	}

	var (
		path string
		ok   bool
	)
	switch toolName {
	case "summarize_scan_analysis":
		path, ok = sharedScanAnalysisPath(msg)
	case "summarize_scan_summary":
		path, ok = sharedScanSummaryPath(msg)
	}
	if !ok {
		return "", false
	}

	input, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		return "", false
	}
	result, err := a.executeAgentTool(ctx, msg, toolName, input)
	if err != nil {
		return fmt.Sprintf("**%s** failed for `%s`: %v", toolName, path, err), true
	}
	log.Printf("[%s] Ran %s via editor-context shortcut on %s", a.Info.Name, toolName, path)
	if strings.TrimSpace(result) == "" {
		return fmt.Sprintf("**%s** completed for `%s` (no output).", toolName, path), true
	}
	return result, true
}
