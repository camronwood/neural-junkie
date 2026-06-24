package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

type fileChangeDirective struct {
	Operation  string
	Path       string
	OldPath    string
	NewPath    string
	OldContent string
	NewContent string
}

var fileChangeBlockRegex = regexp.MustCompile(`(?s)\[FILE_CHANGE\](.*?)\[/FILE_CHANGE\]`)
var editorLineNumberPrefixRegex = regexp.MustCompile(`(?m)^\s*\d+\s*\|\s?`)

// userNamedFileRegex captures "call new-tab.txt", `named foo.md`, etc.
var userNamedFileRegex = regexp.MustCompile(`(?i)\b(?:call|named|called|name)\s+['"]?([a-zA-Z0-9][a-zA-Z0-9._\-]{0,220}\.[a-zA-Z0-9]{1,16})['"]?\b`)

// userNamedFileStemRegex captures extensionless names like "called nj-artical-1".
var userNamedFileStemRegex = regexp.MustCompile(`(?i)\b(?:call|named|called|name)\s+['"]?([a-zA-Z0-9][a-zA-Z0-9._\-]{0,220})['"]?\b`)

var looseOutputFileRegex = regexp.MustCompile(`\b([a-zA-Z0-9][a-zA-Z0-9._\-]*\.(?:txt|md|go|ts|tsx|jsx|js|mjs|cjs|json|yaml|yml|rs|py|html|css|sh|tab))\b`)

var absolutePathFileChangeRE = regexp.MustCompile(`(?m)\[FILE_CHANGE[^\]]*path="/Users/[^"\]]*"[^\]]*\]`)

func sanitizeAbsolutePathFileChangeFromResponse(response string) string {
	return strings.TrimSpace(absolutePathFileChangeRE.ReplaceAllString(response, ""))
}

// stripFileChangeBlocksFromResponse removes [FILE_CHANGE] blocks from model text (ask mode, malformed proposals).
func stripFileChangeBlocksFromResponse(response string) string {
	return strings.TrimSpace(fileChangeBlockRegex.ReplaceAllString(response, ""))
}

var askModeFileChangeMentionRE = regexp.MustCompile(`(?i)\[file_change\]|propose_file_edit`)
var askModeLooseFileChangeRE = regexp.MustCompile(`(?is)\[FILE_CHANGE\][\s\S]*?(?:\n\n|$)`)

// sanitizeAskModeResponse strips file-edit mechanisms from ask-mode advisory replies.
func sanitizeAskModeResponse(response string) string {
	out := stripFileChangeBlocksFromResponse(response)
	out = askModeLooseFileChangeRE.ReplaceAllString(out, "")
	out = askModeFileChangeMentionRE.ReplaceAllString(out, "")
	return strings.TrimSpace(out)
}

func isAskModeReadOnly(sourceMsg *protocol.Message) bool {
	if sourceMsg == nil {
		return false
	}
	caps := protocol.ResolveTurnCapabilities(sourceMsg)
	return !caps.CanProposeFiles && (caps.ComposerMode == "ask" || caps.ComposerMode == "plan")
}

func sanitizeInternalToolNames(response string) string {
	replacer := strings.NewReplacer(
		"ProposeFileEdit", "a file-change proposal",
		"ProposeFileCreate", "a file-change proposal",
		"ProposeFileDelete", "a file-change proposal",
		"ProposeFileMove", "a file-change proposal",
	)
	return replacer.Replace(response)
}

func (a *Agent) maybeSubmitFileChangeFromResponse(ctx context.Context, response, channel string, sourceMsg *protocol.Message) (string, bool, error) {
	if isAskModeReadOnly(sourceMsg) {
		if fileChangeBlockRegex.MatchString(response) {
			log.Printf("[%s] ask_mode_file_change_stripped", a.Info.Name)
		}
		return stripFileChangeBlocksFromResponse(response), false, nil
	}

	if looseFileChangeParseEnabled(sourceMsg) {
		if cleaned, ok, err := a.submitAllFileChangesFromResponse(ctx, response, channel, sourceMsg); ok || err != nil {
			return cleaned, ok, err
		}
	}

	match := fileChangeBlockRegex.FindStringSubmatch(response)
	if len(match) < 2 {
		if loose, ok := parseLooseFileChange(response); ok && looseFileChangeParseEnabled(sourceMsg) {
			path := loose.Path
			if sourceMsg != nil && !isValidFileChangeRelPath(path) {
				if alt := preferImplementationTargetPath(a.resolveWorkspacePath(sourceMsg), sourceMsg.Content, path); alt != "" {
					path = alt
					log.Printf("[%s] loose_file_change_path_replaced(invalid=%q,target=%s)", a.Info.Name, loose.Path, path)
				}
			}
			if !isValidFileChangeRelPath(path) {
				log.Printf("[%s] loose_file_change_rejected(path=%q)", a.Info.Name, loose.Path)
			} else if err := a.proposeFileChangePreferEditOrCreate(ctx, channel, path, loose.NewContent, sourceMsg); err != nil {
				return response, false, err
			} else {
				log.Printf("[%s] loose_file_change_used(path=%s)", a.Info.Name, path)
				return stripLooseFileChangeBlock(response), true, nil
			}
		}
		// Deterministic fallback: user asked to write/create/save files and the model
		// returned fenced content (or explicit approval phrases) but omitted [FILE_CHANGE].
		if sourceMsg == nil || !legacyFileChangeParseEnabled() || !a.shouldUseFileChangeFenceFallback(sourceMsg) {
			log.Printf("[%s] file_change_fence_fallback_skipped(reason=no_explicit_proposal_intent)", a.Info.Name)
			return response, false, nil
		}
		lowerResp := strings.ToLower(response)
		if strings.Contains(lowerResp, "would you like me to propose") ||
			strings.Contains(lowerResp, "i submitted a file change proposal") {
			log.Printf("[%s] file_change_fence_fallback_skipped(reason=response_is_question_or_already_submitted)", a.Info.Name)
			return response, false, nil
		}

		pathSource := sourceMsg.Content
		if userAffirmsPendingImplementation(pathSource) {
			for i := len(a.channelHistory(sourceMsg.Channel)) - 1; i >= 0; i-- {
				m := a.channelHistory(sourceMsg.Channel)[i]
				if m == nil || m.ID == sourceMsg.ID {
					continue
				}
				if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
					pathSource = m.Content
					break
				}
			}
		}
		namedPath := strings.TrimSpace(extractLikelyOutputPathFromUserMessage(pathSource))
		if namedPath == "" {
			namedPath = preferImplementationTargetPathForMessage(a, sourceMsg)
		}
		newContent := stripEditorLineNumberPrefixes(extractCodeFenceForPath(response, namedPath))
		if strings.TrimSpace(newContent) == "" {
			newContent = stripEditorLineNumberPrefixes(extractAnyCodeFenceContent(response))
		}
		if resolved := a.substituteFileExportContent(sourceMsg, newContent); strings.TrimSpace(resolved) != "" {
			newContent = resolved
		}
		if strings.TrimSpace(newContent) == "" {
			log.Printf("[%s] file_change_fence_fallback_skipped(reason=no_fenced_content)", a.Info.Name)
			return response, false, nil
		}
		if looksLikePlaceholderProposalContent(newContent) {
			log.Printf("[%s] file_change_fence_fallback_skipped(reason=placeholder_content)", a.Info.Name)
			return response, false, nil
		}
		if namedPath != "" && !fencedContentPlausibleForPath(namedPath, "", newContent) {
			if alt := preferImplementationTargetPath(a.resolveWorkspacePath(sourceMsg), sourceMsg.Content, ""); alt != "" && alt != namedPath {
				if body := stripEditorLineNumberPrefixes(extractCodeFenceForPath(response, alt)); fencedContentPlausibleForPath(alt, "", body) {
					namedPath = alt
					newContent = body
				}
			}
		}

		activePath := strings.TrimSpace(extractActiveOpenFilePath(sourceMsg))
		wantCreate := userWantsCreateOperation(sourceMsg.Content)

		switch {
		case wantCreate && namedPath != "":
			if err := a.validateProposalForSession(ctx, sourceMsg, namedPath, ProposalOpCreate); err != nil {
				return response, false, err
			}
			if err := a.proposeFileCreateInChannel(channel, namedPath, newContent, sourceMsg); err != nil {
				return response, false, err
			}
			log.Printf("[%s] fallback_path_used(operation=create,target=%s)", a.Info.Name, namedPath)
			return response, true, nil
		case activePath != "":
			if err := a.validateProposalForSession(ctx, sourceMsg, activePath, ProposalOpEdit); err != nil {
				return response, false, err
			}
			if err := a.proposeFileEditInChannel(channel, activePath, "", newContent, sourceMsg); err != nil {
				return response, false, err
			}
			log.Printf("[%s] fallback_path_used(operation=edit,target=%s)", a.Info.Name, activePath)
			return response, true, nil
		case namedPath != "":
			wsPath := a.resolveWorkspacePath(sourceMsg)
			op := ProposalOpCreate
			if InferProposalOperation(wsPath, namedPath) == ProposalOpEdit {
				op = ProposalOpEdit
			}
			if err := a.validateProposalForSession(ctx, sourceMsg, namedPath, op); err != nil {
				return response, false, err
			}
			if op == ProposalOpEdit {
				if err := a.proposeFileEditInChannel(channel, namedPath, "", newContent, sourceMsg); err != nil {
					return response, false, err
				}
			} else if err := a.proposeFileCreateInChannel(channel, namedPath, newContent, sourceMsg); err != nil {
				return response, false, err
			}
			log.Printf("[%s] fallback_path_used(operation=%s,target=%s)", a.Info.Name, op, namedPath)
			return response, true, nil
		default:
			if alt := preferImplementationTargetPathForMessage(a, sourceMsg); alt != "" {
				if body := stripEditorLineNumberPrefixes(extractCodeFenceForPath(response, alt)); strings.TrimSpace(body) != "" {
					newContent = body
				}
				op := ProposalOpEdit
				if userWantsCreateOperation(sourceMsg.Content) {
					op = ProposalOpCreate
				}
				if err := a.validateProposalForSession(ctx, sourceMsg, alt, op); err != nil {
					return response, false, err
				}
				if op == ProposalOpCreate {
					if err := a.proposeFileCreateInChannel(channel, alt, newContent, sourceMsg); err != nil {
						return response, false, err
					}
				} else if err := a.proposeFileEditInChannel(channel, alt, "", newContent, sourceMsg); err != nil {
					return response, false, err
				}
				log.Printf("[%s] fallback_path_used(operation=%s,target=%s)", a.Info.Name, op, alt)
				return response, true, nil
			}
			log.Printf("[%s] file_change_fence_fallback_skipped(reason=missing_target_path)", a.Info.Name)
			return response, false, nil
		}
	}

	directive, err := parseFileChangeDirective(match[1])
	if err != nil && sourceMsg != nil && userRequestsImplementationForMessage(a, sourceMsg) {
		if alt := preferImplementationTargetPath(a.resolveWorkspacePath(sourceMsg), sourceMsg.Content, ""); alt != "" {
			if body := stripEditorLineNumberPrefixes(extractAnyCodeFenceContent(match[1])); strings.TrimSpace(body) != "" {
				if err2 := a.validateProposalForSession(ctx, sourceMsg, alt, ProposalOpEdit); err2 == nil {
					if err2 = a.proposeFileEditInChannel(channel, alt, "", body, sourceMsg); err2 == nil {
						log.Printf("[%s] implement_path_recovery(edit,target=%s)", a.Info.Name, alt)
						cleaned := strings.TrimSpace(fileChangeBlockRegex.ReplaceAllString(response, ""))
						return cleaned, true, nil
					}
				}
			}
		}
	}
	if err != nil {
		// Strip malformed directives from user-visible chat to avoid leaking
		// internal syntax while still surfacing a clean response.
		cleaned := strings.TrimSpace(fileChangeBlockRegex.ReplaceAllString(response, ""))
		return cleaned, false, err
	}

	switch directive.Operation {
	case "create":
		directive.Path = a.ResolveProposalPath(ctx, sourceMsg, directive.Path)
		if err := a.proposeFileChangePreferEditOrCreate(ctx, channel, directive.Path, directive.NewContent, sourceMsg); err != nil {
			return response, false, err
		}
	case "edit":
		directive.Path = a.ResolveProposalPath(ctx, sourceMsg, directive.Path)
		if err := a.validateProposalForSession(ctx, sourceMsg, directive.Path, ProposalOpEdit); err != nil {
			return response, false, err
		}
		if err := a.proposeFileEditInChannel(channel, directive.Path, directive.OldContent, directive.NewContent, sourceMsg); err != nil {
			return response, false, err
		}
	case "delete":
		if err := a.proposeFileDeleteInChannel(channel, directive.Path, sourceMsg); err != nil {
			return response, false, err
		}
	case "move":
		if err := a.proposeFileMoveInChannel(channel, directive.OldPath, directive.NewPath); err != nil {
			return response, false, err
		}
	default:
		return response, false, fmt.Errorf("unsupported file change operation: %s", directive.Operation)
	}
	log.Printf("[%s] directive_path_used(operation=%s,path=%s)", a.Info.Name, directive.Operation, directive.Path)

	cleaned := strings.TrimSpace(fileChangeBlockRegex.ReplaceAllString(response, ""))
	return cleaned, true, nil
}

func (a *Agent) submitAllFileChangesFromResponse(ctx context.Context, response, channel string, sourceMsg *protocol.Message) (string, bool, error) {
	var directives []*fileChangeDirective
	for _, m := range fileChangeBlockRegex.FindAllStringSubmatch(response, -1) {
		if len(m) < 2 {
			continue
		}
		if d, err := parseFileChangeDirective(m[1]); err == nil && d != nil {
			directives = append(directives, d)
		}
	}
	directives = append(directives, parseAllLooseFileChanges(response)...)

	seen := make(map[string]bool)
	proposed := false
	cleaned := response
	for _, directive := range directives {
		if directive == nil {
			continue
		}
		path := normalizeFileChangeRelPath(directive.Path)
		if path == "" || !isValidFileChangeRelPath(path) {
			continue
		}
		body := strings.TrimSpace(directive.NewContent)
		if body == "" {
			continue
		}
		key := path + "\x00" + body
		if seen[key] {
			continue
		}
		seen[key] = true

		switch strings.ToLower(strings.TrimSpace(directive.Operation)) {
		case "", "create":
			directive.Path = a.ResolveProposalPath(ctx, sourceMsg, path)
			if err := a.proposeFileChangePreferEditOrCreate(ctx, channel, directive.Path, body, sourceMsg); err != nil {
				return cleaned, proposed, err
			}
		case "edit":
			directive.Path = a.ResolveProposalPath(ctx, sourceMsg, path)
			if err := a.validateProposalForSession(ctx, sourceMsg, directive.Path, ProposalOpEdit); err != nil {
				return cleaned, proposed, err
			}
			if err := a.proposeFileEditInChannel(channel, directive.Path, directive.OldContent, body, sourceMsg); err != nil {
				return cleaned, proposed, err
			}
		default:
			continue
		}
		log.Printf("[%s] collab_file_change_proposed(operation=%s,path=%s)", a.Info.Name, directive.Operation, directive.Path)
		proposed = true
	}

	if !proposed {
		return response, false, nil
	}
	cleaned = strings.TrimSpace(fileChangeBlockRegex.ReplaceAllString(response, ""))
	cleaned = stripLooseFileChangeBlock(cleaned)
	return cleaned, true, nil
}

func parseFileChangeDirective(block string) (*fileChangeDirective, error) {
	d := &fileChangeDirective{
		Operation: strings.ToLower(extractDirectiveField(block, "operation")),
		Path:      extractDirectiveField(block, "path"),
		OldPath:   extractDirectiveField(block, "old_path"),
		NewPath:   extractDirectiveField(block, "new_path"),
	}

	d.NewContent = extractLabeledCodeFence(block, "new")
	d.OldContent = extractLabeledCodeFence(block, "old")

	if d.NewContent == "" {
		// Fallback: use first generic fence as new content.
		d.NewContent = extractFirstCodeFence(block)
	}
	d.NewContent = stripEditorLineNumberPrefixes(d.NewContent)
	d.OldContent = stripEditorLineNumberPrefixes(d.OldContent)

	d.Path = normalizeFileChangeRelPath(d.Path)
	d.OldPath = normalizeFileChangeRelPath(d.OldPath)
	d.NewPath = normalizeFileChangeRelPath(d.NewPath)

	switch d.Operation {
	case "create":
		if strings.TrimSpace(d.Path) == "" {
			return nil, fmt.Errorf("create directive missing path")
		}
		if !isValidFileChangeRelPath(d.Path) {
			return nil, fmt.Errorf("create directive has invalid path %q", d.Path)
		}
		if strings.TrimSpace(d.NewContent) == "" {
			return nil, fmt.Errorf("create directive missing new content")
		}
	case "edit":
		if strings.TrimSpace(d.Path) == "" {
			return nil, fmt.Errorf("edit directive missing path")
		}
		if !isValidFileChangeRelPath(d.Path) {
			return nil, fmt.Errorf("edit directive has invalid path %q", d.Path)
		}
		if strings.TrimSpace(d.NewContent) == "" {
			return nil, fmt.Errorf("edit directive missing new content")
		}
	case "delete":
		if strings.TrimSpace(d.Path) == "" {
			return nil, fmt.Errorf("delete directive missing path")
		}
		if !isValidFileChangeRelPath(d.Path) {
			return nil, fmt.Errorf("delete directive has invalid path %q", d.Path)
		}
	case "move":
		if strings.TrimSpace(d.OldPath) == "" || strings.TrimSpace(d.NewPath) == "" {
			return nil, fmt.Errorf("move directive missing old_path/new_path")
		}
	default:
		return nil, fmt.Errorf("missing or unsupported operation")
	}

	return d, nil
}

func stripEditorLineNumberPrefixes(content string) string {
	if content == "" {
		return content
	}
	return editorLineNumberPrefixRegex.ReplaceAllString(content, "")
}

func extractDirectiveField(block, field string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?mi)^\s*%s:\s*(.+)\s*$`, regexp.QuoteMeta(field)))
	m := re.FindStringSubmatch(block)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func extractLabeledCodeFence(block, label string) string {
	re := regexp.MustCompile(fmt.Sprintf("(?s)```%s\\s*\\n(.*?)\\n```", regexp.QuoteMeta(label)))
	m := re.FindStringSubmatch(block)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func extractFirstCodeFence(block string) string {
	re := regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\s*\\n(.*?)\\n```")
	m := re.FindStringSubmatch(block)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func extractLongestCodeFence(content string) string {
	re := regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\s*\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return ""
	}
	longest := ""
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		if len(m[1]) > len(longest) {
			longest = m[1]
		}
	}
	return longest
}

// extractAnyCodeFenceContent returns the longest fenced body, trying strict then relaxed patterns
// (models often omit the newline the strict extractor requires).
func extractAnyCodeFenceContent(content string) string {
	if s := extractLongestCodeFence(content); strings.TrimSpace(s) != "" {
		return s
	}
	relaxed := regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\s*\\n(.*?)```")
	longest := ""
	for _, m := range relaxed.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		if len(m[1]) > len(longest) {
			longest = m[1]
		}
	}
	if strings.TrimSpace(longest) != "" {
		return longest
	}
	anyFence := regexp.MustCompile("(?s)```(.*?)```")
	for _, m := range anyFence.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		body := strings.TrimSpace(m[1])
		// Skip single-line ```lang``` with no body
		if !strings.Contains(body, "\n") && strings.HasPrefix(body, "```") {
			continue
		}
		if len(body) > len(longest) {
			longest = body
		}
	}
	return longest
}

func extractLikelyOutputPathFromUserMessage(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if p := longestValidPathIn(DetectFilePaths(content)); p != "" {
		return p
	}
	if m := userNamedFileRegex.FindStringSubmatch(content); len(m) > 1 {
		return normalizeFileChangeRelPath(strings.TrimSpace(m[1]))
	}
	if userRequestsFileExport(content) || userReferencesPriorAssistantContent(content) {
		if m := userNamedFileStemRegex.FindStringSubmatch(content); len(m) > 1 {
			stem := strings.TrimSpace(m[1])
			if !strings.Contains(stem, ".") {
				stem += ".md"
			}
			if p := normalizeFileChangeRelPath(stem); isValidFileChangeRelPath(p) {
				return p
			}
		}
	}
	all := looseOutputFileRegex.FindAllString(content, -1)
	if len(all) == 0 {
		return ""
	}
	return normalizeFileChangeRelPath(strings.TrimSpace(all[len(all)-1]))
}

func isUserRequestingFileWrite(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	phrases := []string{
		"create a file", "create file", "create new file", "create that file", "please create that file",
		"new file",
		"write a file", "write file", "write the file", "write this file",
		"save to", "save this", "save the file", "save as", "save it", "save it as", "save it in",
		"put in a file", "put the tab", "put this in",
		"generate a file", "output to", "store in", "store the", "store that", "store it",
		"fill the file", "make a file", "make the file", "add a file", "markdown file",
		"complete file", "full tab", "complete tab", "turn it into",
		"write to disk", "same directory", "this folder", "next to the",
		"implement", "please implement", "implement that", "implement the",
		"code this", "build this", "apply the plan", "make the changes",
		"can you fix", "please fix", "fix it", "fix the", "debug this",
		"blank screen", "white screen", "not working", "broken",
	}
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func userWantsCreateOperation(content string) bool {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "new file") {
		return true
	}
	if strings.Contains(lower, "create") && (strings.Contains(lower, "file") || strings.Contains(lower, "tab")) {
		return true
	}
	return false
}

func isExplicitProposalIntent(content string) bool {
	lower := strings.TrimSpace(strings.ToLower(content))
	if lower == "" {
		return false
	}
	explicitPhrases := []string{
		"propose it", "please propose", "submit it", "submit the change",
		"apply it", "go ahead and update", "update the file", "make the change",
		"yes propose", "yes, propose", "yes please propose",
		"create the file", "create this file", "save the file", "write the file",
		"please implement", "implement that", "implement the plan", "go ahead and implement",
		"ok please implement", "please implement the",
	}
	for _, p := range explicitPhrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	// A short "yes/ok/do it" reply in this flow is treated as explicit confirmation.
	shortAffirmations := map[string]bool{
		"yes": true, "yes please": true, "ok": true, "okay": true, "do it": true, "go ahead": true,
	}
	return shortAffirmations[lower]
}

// shouldUseFileChangeFenceFallback allows converting fenced code into proposals on
// implementation turns, not only explicit "create a file" phrasing.
func (a *Agent) shouldUseFileChangeFenceFallback(sourceMsg *protocol.Message) bool {
	if sourceMsg == nil {
		return false
	}
	if isAskModeReadOnly(sourceMsg) {
		return false
	}
	content := sourceMsg.Content
	if sourceMsg.IdeEditorModeIsExport() || userRequestsFileExportForMessage(sourceMsg) {
		return true
	}
	if userRequestsContentDelivery(content) || isBareWorkspaceDirective(content) {
		return isExplicitProposalIntent(content) || isUserRequestingFileWrite(content)
	}
	if sourceMsg.ImplementationSession() {
		return true
	}
	if a != nil && userRequestsImplementationForMessage(a, sourceMsg) {
		return true
	}
	return isExplicitProposalIntent(content) || isUserRequestingFileWrite(content)
}

func extractActiveOpenFilePath(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	wsCtx, ok := msg.Metadata["workspace_context"]
	if !ok {
		return ""
	}
	ctxMap, ok := wsCtx.(map[string]interface{})
	if !ok {
		return ""
	}
	openFiles, ok := ctxMap["open_files"].([]interface{})
	if !ok || len(openFiles) == 0 {
		return ""
	}
	for _, f := range openFiles {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		isActive, _ := fm["is_active"].(bool)
		path, _ := fm["path"].(string)
		if isActive && strings.TrimSpace(path) != "" {
			return path
		}
	}
	if fm, ok := openFiles[0].(map[string]interface{}); ok {
		if path, ok := fm["path"].(string); ok {
			return path
		}
	}
	return ""
}

// File change proposal helper methods

// ProposeFileEdit proposes an edit to an existing file
func (a *Agent) ProposeFileEdit(path, oldContent, newContent string) error {
	return a.proposeFileEditInChannel(a.Context.CurrentChannel, path, oldContent, newContent, nil)
}

func (a *Agent) proposeFileEditInChannel(channel, path, oldContent, newContent string, sourceMsg *protocol.Message) error {
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	path = normalizeFileChangeRelPath(path)
	if !isValidFileChangeRelPath(path) {
		return fmt.Errorf("invalid file change path: %q", path)
	}
	newContent = stripEditorLineNumberPrefixes(newContent)
	if err := validateProposalContent(path, newContent); err != nil {
		return err
	}
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "edit",
		FilePath:    path,
		OldContent:  stripEditorLineNumberPrefixes(oldContent),
		NewContent:  stripEditorLineNumberPrefixes(newContent),
		Agent:       a.Info,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, channel, a.Info,
		fmt.Sprintf("📝 Proposing to edit file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal
	a.attachWorkspaceContextToProposalMessage(channel, msg, proposal)
	attachIdeSessionMetadataToProposal(msg, sourceMsg)

	return a.Hub.SendMessage(msg)
}

// ProposeFileCreate proposes creating a new file
func (a *Agent) ProposeFileCreate(path, content string) error {
	return a.proposeFileCreateInChannel(a.Context.CurrentChannel, path, content, nil)
}

func (a *Agent) proposeFileChangePreferEditOrCreate(ctx context.Context, channel, path, content string, sourceMsg *protocol.Message) error {
	if sourceMsg != nil && (sourceMsg.IdeEditorModeIsExport() || userRequestsFileExportForMessage(sourceMsg)) {
		if userPath := preferFileExportTargetPath(sourceMsg); userPath != "" {
			path = userPath
		}
	}
	path = a.ResolveProposalPath(ctx, sourceMsg, path)
	if !isValidFileChangeRelPath(path) {
		return fmt.Errorf("invalid file change path: %q", path)
	}
	content = a.substituteFileExportContent(sourceMsg, content)
	wsPath := ""
	if sourceMsg != nil {
		wsPath = a.resolveWorkspacePath(sourceMsg)
	}
	op := InferProposalOperation(wsPath, path)
	if err := a.validateProposalForSession(ctx, sourceMsg, path, op); err != nil {
		return err
	}
	if wsPath != "" {
		resolved := filepath.Join(wsPath, path)
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
			return a.proposeFileEditInChannel(channel, path, "", content, sourceMsg)
		}
	}
	return a.proposeFileCreateInChannel(channel, path, content, sourceMsg)
}

func (a *Agent) proposeFileCreateInChannel(channel, path, content string, sourceMsg *protocol.Message) error {
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	path = normalizeFileChangeRelPath(path)
	if !isValidFileChangeRelPath(path) {
		return fmt.Errorf("invalid file change path: %q", path)
	}
	content = stripEditorLineNumberPrefixes(content)
	if err := validateProposalContent(path, content); err != nil {
		return err
	}
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "create",
		FilePath:    path,
		NewContent:  content,
		Agent:       a.Info,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, channel, a.Info,
		fmt.Sprintf("📄 Proposing to create file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal
	a.attachWorkspaceContextToProposalMessage(channel, msg, proposal)
	attachIdeSessionMetadataToProposal(msg, sourceMsg)

	return a.Hub.SendMessage(msg)
}

// ProposeFileDelete proposes deleting a file
func (a *Agent) ProposeFileDelete(path string) error {
	return a.proposeFileDeleteInChannel(a.Context.CurrentChannel, path, nil)
}

func (a *Agent) proposeFileDeleteInChannel(channel, path string, sourceMsg *protocol.Message) error {
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "delete",
		FilePath:    path,
		Agent:       a.Info,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    true,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, channel, a.Info,
		fmt.Sprintf("🗑️ Proposing to delete file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal
	a.attachWorkspaceContextToProposalMessage(channel, msg, proposal)
	attachIdeSessionMetadataToProposal(msg, sourceMsg)

	return a.Hub.SendMessage(msg)
}

// ProposeFileMove proposes moving/renaming a file
func (a *Agent) ProposeFileMove(oldPath, newPath string) error {
	return a.proposeFileMoveInChannel(a.Context.CurrentChannel, oldPath, newPath)
}

func (a *Agent) proposeFileMoveInChannel(channel, oldPath, newPath string) error {
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "move",
		FilePath:    oldPath,
		OldPath:     oldPath,
		NewPath:     newPath,
		Agent:       a.Info,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, channel, a.Info,
		fmt.Sprintf("📁 Proposing to move file: %s → %s", oldPath, newPath))
	msg.Metadata["file_change_proposal"] = proposal
	a.attachWorkspaceContextToProposalMessage(channel, msg, proposal)

	return a.Hub.SendMessage(msg)
}

func (a *Agent) attachWorkspaceContextToProposalMessage(channel string, msg *protocol.Message, proposal *protocol.FileChangeProposal) {
	workspaceContext, ok := a.latestWorkspaceContext(channel)
	if !ok {
		return
	}
	msg.Metadata["workspace_context"] = workspaceContext
	if proposal.Metadata == nil {
		proposal.Metadata = make(map[string]interface{})
	}
	proposal.Metadata["workspace_context"] = workspaceContext
}

func (a *Agent) latestWorkspaceContext(channel string) (interface{}, bool) {
	if wc, ok := a.latestWorkspaceContextForChannel(channel); ok {
		return wc, true
	}
	// Collaboration runs in collab-* channels; user workspace metadata often
	// exists only on #general or another channel the human used first.
	if channel != "general" {
		if wc, ok := a.latestWorkspaceContextForChannel("general"); ok {
			return wc, true
		}
	}
	for _, ch := range a.historyChannelNames() {
		if ch == channel || ch == "general" {
			continue
		}
		if wc, ok := a.latestWorkspaceContextForChannel(ch); ok {
			return wc, true
		}
	}
	return nil, false
}

func (a *Agent) latestWorkspaceContextForChannel(channel string) (interface{}, bool) {
	history := a.channelHistory(channel)
	for i := len(history) - 1; i >= 0; i-- {
		if history[i] == nil || history[i].Metadata == nil {
			continue
		}
		if wsCtx, ok := history[i].Metadata["workspace_context"]; ok && wsCtx != nil {
			return wsCtx, true
		}
	}
	return nil, false
}
