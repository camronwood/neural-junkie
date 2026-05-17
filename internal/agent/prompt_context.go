package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Metadata keys for client-supplied prompt context (must match desktop).
const (
	MetadataUserRulesMarkdown    = "user_rules_markdown"
	MetadataPromptAttachments    = "prompt_attachments"
	MetadataGrantedHubDataAccess = "granted_hub_data_access"
)

const (
	maxUserRulesMarkdownBytes = 32 * 1024
	// Total JSON size for prompt_attachments after marshal (approximate guard).
	maxPromptAttachmentsBytes = 400 * 1024
	maxPerAttachmentContent     = 120 * 1024
	maxAttachmentFiles          = 12
)

// AppendUserAndAgentRules writes global (message metadata) and per-agent markdown
// into the system portion of a prompt, before ai.SystemPromptSeparator.
func AppendUserAndAgentRules(system *strings.Builder, msg *protocol.Message, self *protocol.AgentInfo) {
	if msg == nil || msg.Metadata == nil {
		if self != nil && strings.TrimSpace(self.CustomRulesMarkdown) != "" {
			writeAgentOnlyRules(system, self)
		}
		return
	}

	raw, ok := msg.Metadata[MetadataUserRulesMarkdown]
	userRules := ""
	if ok && raw != nil {
		if s, ok := raw.(string); ok {
			userRules = strings.TrimSpace(s)
		}
	}

	agentRules := ""
	if self != nil {
		agentRules = strings.TrimSpace(self.CustomRulesMarkdown)
	}

	if userRules == "" && agentRules == "" {
		return
	}

	system.WriteString("\n=== USER-CONFIGURED RULES ===\n")
	if userRules != "" {
		system.WriteString("The following instructions come from the user's global rules (markdown).\n")
		system.WriteString("Apply them when they do not conflict with safety or system policy.\n\n")
		system.WriteString(userRules)
		system.WriteString("\n\n")
	}
	if agentRules != "" {
		system.WriteString(fmt.Sprintf("The following instructions are scoped to you (%s) only (markdown).\n\n", self.Name))
		system.WriteString(agentRules)
		system.WriteString("\n\n")
	}
	system.WriteString("=== END USER-CONFIGURED RULES ===\n\n")
}

func writeAgentOnlyRules(system *strings.Builder, self *protocol.AgentInfo) {
	if self == nil {
		return
	}
	agentRules := strings.TrimSpace(self.CustomRulesMarkdown)
	if agentRules == "" {
		return
	}
	system.WriteString("\n=== USER-CONFIGURED RULES ===\n")
	system.WriteString(fmt.Sprintf("The following instructions are scoped to you (%s) only (markdown).\n\n", self.Name))
	system.WriteString(agentRules)
	system.WriteString("\n\n=== END USER-CONFIGURED RULES ===\n\n")
}

// AppendPromptAttachments appends dropped/attached files from message metadata (user-facing context).
func AppendPromptAttachments(user *strings.Builder, msg *protocol.Message) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	raw, ok := msg.Metadata[MetadataPromptAttachments]
	if !ok || raw == nil {
		return
	}
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		return
	}

	user.WriteString("\n=== ATTACHED FILES (USER UPLOAD) ===\n")
	user.WriteString("The user attached the following files for this message. Use them as primary context when relevant.\n")
	user.WriteString("Each line is prefixed with its line number.\n\n")

	for _, item := range arr {
		fm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		path, _ := fm["path"].(string)
		lang, _ := fm["language"].(string)
		content, _ := fm["content"].(string)
		if path == "" {
			path = "attachment"
		}
		if lang == "" {
			lang = inferLanguage(path)
		}
		numbered := addLineNumbers(content)
		user.WriteString(fmt.Sprintf("### %s (%s)\n```%s\n%s\n```\n\n", path, lang, lang, numbered))
	}
	user.WriteString("=== END ATTACHED FILES ===\n\n")
}

// PrependRulesAndAttachmentsForMonolithic prepends rules and attachments for agents that use a single prompt string.
func PrependRulesAndAttachmentsForMonolithic(sb *strings.Builder, msg *protocol.Message, self *protocol.AgentInfo) {
	var rules strings.Builder
	AppendUserAndAgentRules(&rules, msg, self)
	if rules.Len() > 0 {
		sb.WriteString(rules.String())
	}
	var attach strings.Builder
	AppendPromptAttachments(&attach, msg)
	if attach.Len() > 0 {
		sb.WriteString(attach.String())
	}
}

// SanitizeInboundMessageMetadata truncates user rules and prompt attachments on the hub
// so oversized payloads cannot exhaust memory.
func SanitizeInboundMessageMetadata(msg *protocol.Message) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	if raw, ok := msg.Metadata[MetadataUserRulesMarkdown]; ok {
		if s, ok := raw.(string); ok {
			if len(s) > maxUserRulesMarkdownBytes {
				msg.Metadata[MetadataUserRulesMarkdown] = truncateStringBytes(s, maxUserRulesMarkdownBytes)
			}
		}
	}
	if raw, ok := msg.Metadata[MetadataPromptAttachments]; ok {
		msg.Metadata[MetadataPromptAttachments] = sanitizePromptAttachmentsValue(raw)
	}
	protocol.SanitizeUserImagesMetadata(msg)
}

func truncateStringBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	s = s[:maxBytes]
	for !utf8.ValidString(s) && len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s + "\n\n[truncated: exceeded server limit]"
}

func sanitizePromptAttachmentsValue(raw interface{}) interface{} {
	arr, ok := raw.([]interface{})
	if !ok {
		return raw
	}
	out := make([]interface{}, 0, len(arr))
	total := 0
	for i, item := range arr {
		if i >= maxAttachmentFiles {
			break
		}
		fm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		path, _ := fm["path"].(string)
		lang, _ := fm["language"].(string)
		content, _ := fm["content"].(string)
		if len(content) > maxPerAttachmentContent {
			content = truncateStringBytes(content, maxPerAttachmentContent)
		}
		entry := map[string]interface{}{
			"path":     path,
			"language": lang,
			"content":  content,
		}
		enc, _ := json.Marshal(entry)
		if total+len(enc) > maxPromptAttachmentsBytes {
			break
		}
		total += len(enc)
		out = append(out, entry)
	}
	return out
}

// AppendGrantedHubDataAccess injects user-approved ~/.neural-junkie file/directory reads.
func AppendGrantedHubDataAccess(prompt *strings.Builder, msg *protocol.Message) int {
	if msg == nil || msg.Metadata == nil {
		return 0
	}
	raw, ok := msg.Metadata[MetadataGrantedHubDataAccess]
	if !ok || raw == nil {
		return 0
	}
	root, _ := raw.(map[string]interface{})
	if root == nil {
		return 0
	}
	entries, _ := root["entries"].([]interface{})
	if len(entries) == 0 {
		return 0
	}
	prompt.WriteString("\n=== GRANTED HUB DATA ACCESS ===\n")
	prompt.WriteString("The user explicitly allowed you to read Neural Junkie local hub data for this reply only.\n")
	prompt.WriteString("Treat this as authoritative for session/collab debugging; do not claim you lack access.\n")
	prompt.WriteString("When they ask to review a file, analyze the file content below — not unrelated thread/channel chat history.\n\n")
	if r, ok := root["root"].(string); ok && r != "" {
		prompt.WriteString(fmt.Sprintf("Hub data root: %s\n\n", r))
	}
	loaded := 0
	for _, item := range entries {
		em, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		path, _ := em["path"].(string)
		note, _ := em["note"].(string)
		skipped, _ := em["skipped"].(bool)
		content, _ := em["content"].(string)
		truncated, _ := em["truncated"].(bool)
		prompt.WriteString(fmt.Sprintf("### %s\n", path))
		if note != "" {
			prompt.WriteString(note + "\n")
		}
		if skipped {
			prompt.WriteString("(content not loaded — see note)\n\n")
			continue
		}
		if truncated {
			prompt.WriteString("(truncated for context limits)\n")
		}
		if strings.TrimSpace(content) != "" {
			prompt.WriteString("```json\n")
			prompt.WriteString(content)
			if !strings.HasSuffix(content, "\n") {
				prompt.WriteString("\n")
			}
			prompt.WriteString("```\n\n")
			loaded++
		}
	}
	prompt.WriteString("=== END GRANTED HUB DATA ACCESS ===\n\n")
	return loaded
}
