package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	priorReferenceMinChars       = 400
	priorReferenceInjectMaxBytes = 24 * 1024
)

var priorReferencePhraseRE = regexp.MustCompile(`(?i)\b(few messages back|what you wrote|you created|that article|that artical|artical content|earlier you|from before|you wrote|what you said|the article you|previous reply|last reply|message(s)? (ago|back)|use (the |that )?(article|artical|content|text|reply))\b`)

var priorReferenceExplicitBackRE = regexp.MustCompile(`(?i)\b(few messages back|from before|earlier you|messages? back|what you wrote|you wrote)\b`)

var priorReferenceNumberedListRE = regexp.MustCompile(`(?m)^\d+\.\s`)

// userReferencesPriorAssistantContent reports when the user points at earlier assistant output.
// A bare export request ("store that in a markdown file") is not itself a back-reference —
// only an explicit pointer to earlier content ("a few messages back", "what you wrote") counts,
// unless the user also uses that explicit back-reference language.
func userReferencesPriorAssistantContent(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if fileExportRE.MatchString(content) && !priorReferenceExplicitBackRE.MatchString(content) {
		return false
	}
	return priorReferencePhraseRE.MatchString(content)
}

func assistantMessageSkippableForPriorReference(m *protocol.Message) bool {
	if m == nil {
		return true
	}
	switch m.Type {
	case protocol.MessageTypeAgentJoin, protocol.MessageTypeSystemInfo:
		return true
	}
	if protocol.IsUserLikeSender(m.From) {
		return true
	}
	body := strings.TrimSpace(m.Content)
	if len(body) < 80 {
		return true
	}
	return false
}

func looksLikePriorAssistantMarkdown(content string) bool {
	content = strings.TrimSpace(content)
	if len(content) < priorReferenceMinChars {
		return false
	}
	if strings.Contains(content, "###") {
		return true
	}
	if strings.Contains(content, "\n---\n") || strings.HasPrefix(content, "---\n") {
		return true
	}
	if priorReferenceNumberedListRE.MatchString(content) {
		return true
	}
	return len(content) >= priorReferenceMinChars*2
}

// findPriorAssistantContent scans channel history newest-first for long assistant markdown.
// Returns the most recent qualifying message (not the longest), so a later short correction
// beats an older long hallucinated plan.
func findPriorAssistantContent(history []*protocol.Message, skipMsgID, agentID string, minChars int) string {
	if minChars <= 0 {
		minChars = priorReferenceMinChars
	}
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
		if !looksLikePriorAssistantMarkdown(body) && len(body) < minChars*2 {
			continue
		}
		return body
	}
	return ""
}

const priorReferenceMissingHistoryReply = "I don't have that earlier reply in this channel's history (possibly cleared by restart). Paste the text or ask me to regenerate."

// tryPriorReferenceResponse answers deterministically when prior content is referenced but missing.
func (a *Agent) tryPriorReferenceResponse(msg *protocol.Message) (string, bool) {
	if a == nil || msg == nil || !ShouldRunPriorReference(a.effectiveKnowledgePlanFromMessage(msg)) {
		return "", false
	}
	// Semantic classifiers often stamp prior_reference on presence checks ("are you there?").
	// Only soft-fail when the user actually pointed at earlier assistant content.
	if !userReferencesPriorAssistantContent(msg.Content) {
		return "", false
	}
	history := a.historyForPriorReference(msg.Channel)
	if findPriorAssistantContent(history, msg.ID, a.Info.ID, priorReferenceMinChars) != "" {
		return "", false
	}
	return priorReferenceMissingHistoryReply, true
}

// shouldInjectPriorAssistantContent reports when the prompt should include prior assistant output.
func shouldInjectPriorAssistantContent(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if userReferencesPriorAssistantContent(msg.Content) {
		return true
	}
	if userRequestsFileExport(msg.Content) || msg.IdeEditorModeIsExport() {
		return true
	}
	return false
}

// appendPriorReferenceGuidance injects referenced assistant content into the prompt when found.
func (a *Agent) appendPriorReferenceGuidance(prompt string, msg *protocol.Message, history []*protocol.Message) string {
	if a == nil || msg == nil {
		return prompt
	}
	plan := a.effectiveKnowledgePlanFromMessage(msg)
	runPrior := ShouldRunPriorReference(plan)
	runExport := userRequestsFileExport(msg.Content) || msg.IdeEditorModeIsExport()
	if !runPrior && !runExport {
		return prompt
	}
	content := findPriorAssistantContent(history, msg.ID, a.Info.ID, priorReferenceMinChars)
	if content == "" {
		return prompt
	}
	if len(content) > priorReferenceInjectMaxBytes {
		content = content[:priorReferenceInjectMaxBytes] + "\n…(prior content truncated)\n"
	}
	block := fmt.Sprintf(
		"\n=== PRIOR ASSISTANT CONTENT (referenced) ===\n"+
			"The user referenced content from an earlier reply. Use this verbatim when exporting to a file — do not invent a generic template.\n\n%s\n\n",
		content,
	)
	if runPrior {
		a.recordKnowledgeExecutedFor(msg.ID, "prior_reference")
	}
	return prompt + block
}

// resolveFileExportContent picks the body for a file-export proposal from channel history.
func resolveFileExportContent(a *Agent, msg *protocol.Message, history []*protocol.Message) (content string, source string) {
	if a == nil || msg == nil {
		return "", ""
	}
	if history == nil {
		history = a.historyForPriorReference(msg.Channel)
	}
	if userReferencesPriorAssistantContent(msg.Content) ||
		strings.Contains(strings.ToLower(msg.Content), "artical you created") ||
		strings.Contains(strings.ToLower(msg.Content), "article you created") ||
		fileExportRE.MatchString(msg.Content) {
		if prior := findPriorAssistantContent(history, msg.ID, a.Info.ID, priorReferenceMinChars); prior != "" {
			return prior, "prior_assistant"
		}
	}
	if p := longestValidPathIn(DetectFilePaths(msg.Content)); p != "" {
		if ws := a.resolveWorkspacePath(msg); ws != "" {
			if body, err := readWorkspaceFileTail(ws, p, 64*1024); err == nil && strings.TrimSpace(body) != "" {
				return body, "named_path"
			}
		}
	}
	return "", ""
}

func readWorkspaceFileTail(wsRoot, relPath string, maxBytes int) (string, error) {
	relPath = normalizeFileChangeRelPath(relPath)
	if relPath == "" || !isValidFileChangeRelPath(relPath) {
		return "", fmt.Errorf("invalid path")
	}
	wsRoot = strings.TrimSpace(wsRoot)
	if wsRoot == "" {
		return "", fmt.Errorf("no workspace")
	}
	data, err := os.ReadFile(filepath.Join(wsRoot, relPath))
	if err != nil {
		return "", err
	}
	if maxBytes > 0 && len(data) > maxBytes {
		data = data[:maxBytes]
	}
	return string(data), nil
}

// preferFileExportTargetPath returns the user-named export path when present.
func preferFileExportTargetPath(msg *protocol.Message) string {
	if msg == nil {
		return ""
	}
	if p := strings.TrimSpace(extractLikelyOutputPathFromUserMessage(msg.Content)); p != "" {
		return p
	}
	return ""
}

// substituteFileExportContent replaces empty or placeholder proposal bodies from channel history.
func (a *Agent) substituteFileExportContent(sourceMsg *protocol.Message, content string) string {
	content = strings.TrimSpace(content)
	if sourceMsg == nil {
		return content
	}
	exportTurn := userRequestsFileExport(sourceMsg.Content) || sourceMsg.IdeEditorModeIsExport()
	if !exportTurn {
		return content
	}
	if content != "" && !looksLikePlaceholderProposalContent(content) {
		return content
	}
	resolved, source := resolveFileExportContent(a, sourceMsg, nil)
	if strings.TrimSpace(resolved) == "" || looksLikePlaceholderProposalContent(resolved) {
		return content
	}
	log.Printf("[%s] file_export_content_resolved(source=%s,len=%d)", a.Info.Name, source, len(resolved))
	return resolved
}

// LooksLikeCorruptSourceContent reports git-diff debris or LLM stub text in source files.
func LooksLikeCorruptSourceContent(content string) bool {
	trim := strings.TrimSpace(content)
	if trim == "" {
		return false
	}
	if strings.HasPrefix(trim, "diff --git") {
		return true
	}
	lower := strings.ToLower(trim)
	if strings.Contains(lower, "your valid javascript code here") {
		return true
	}
	if strings.Contains(lower, "// your valid javascript") {
		return true
	}
	return false
}

func looksLikePlaceholderProposalContent(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if LooksLikeCorruptSourceContent(content) {
		return true
	}
	return collaboration.LooksLikePlaceholderContent(content)
}
