package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	MetadataAmbientState       = "ambient_state"
	maxAmbientStateServerBytes = 16 * 1024
)

var (
	ambientRelevantRE      = regexp.MustCompile(`(?i)\b(code|file|editor|selection|line|implement|fix|debug|error|warning|diagnostic|test|build|terminal|command|shell|git|commit|branch|diff|staged|workspace|repo)\b`)
	ambientSensitivePathRE = regexp.MustCompile(`(?i)(^|/)(\.env(\.|$)|id_(rsa|dsa|ecdsa|ed25519)(\.|$)|credentials?(\.|$)|secrets?(\.|$)|[^/]*\.(pem|key|p12|pfx)|\.aws(/|$)|\.ssh(/|$))`)
	ambientANSIRe          = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\))`)
	ambientPrivateKeyRE    = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	ambientSecretRE        = regexp.MustCompile(`(?i)(\b(?:api[_-]?key|access[_-]?token|auth[_-]?token|client[_-]?secret|password|passwd|secret)\b\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s"',;]+)`)
)

func ambientStateRelevant(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	return msg.IdeRouteAgentType() != "" || ambientRelevantRE.MatchString(msg.Content)
}

func sanitizeAmbientText(s string, maxBytes int) string {
	s = ambientANSIRe.ReplaceAllString(s, "")
	s = ambientPrivateKeyRE.ReplaceAllString(s, "[REDACTED PRIVATE KEY]")
	s = ambientSecretRE.ReplaceAllString(s, `${1}[REDACTED]`)
	s = strings.Map(func(r rune) rune {
		if (unicode.IsControl(r) && r != '\n' && r != '\t') || r == '\u007f' {
			return -1
		}
		return r
	}, s)
	if maxBytes > 0 && len(s) > maxBytes {
		s = s[len(s)-maxBytes:]
		for !utf8.ValidString(s) && len(s) > 0 {
			s = s[1:]
		}
	}
	return s
}

func ambientStringMap(raw interface{}) map[string]interface{} {
	m, _ := raw.(map[string]interface{})
	return m
}

func sanitizeAmbientPath(raw interface{}) string {
	path, _ := raw.(string)
	return sanitizeAmbientText(path, 1024)
}

func sanitizeAmbientStateValue(raw interface{}) map[string]interface{} {
	root := ambientStringMap(raw)
	if root == nil {
		return nil
	}
	out := map[string]interface{}{}
	if editor := ambientStringMap(root["active_editor"]); editor != nil {
		path := sanitizeAmbientPath(editor["path"])
		if path != "" {
			safe := map[string]interface{}{"path": path}
			if cursor := ambientStringMap(editor["cursor"]); cursor != nil {
				safe["cursor"] = map[string]interface{}{"line": cursor["line"], "column": cursor["column"]}
			}
			if selection := ambientStringMap(editor["selection"]); selection != nil {
				sel := map[string]interface{}{"start_line": selection["start_line"], "end_line": selection["end_line"]}
				if !ambientSensitivePathRE.MatchString(strings.ReplaceAll(path, `\`, "/")) {
					if text, ok := selection["text"].(string); ok {
						sel["text"] = sanitizeAmbientText(text, 2048)
					}
				}
				safe["selection"] = sel
			}
			out["active_editor"] = safe
		}
	}
	if rows, ok := root["diagnostics"].([]interface{}); ok {
		safe := make([]interface{}, 0, min(len(rows), 40))
		for _, row := range rows {
			if len(safe) >= 40 {
				break
			}
			item := ambientStringMap(row)
			if item == nil {
				continue
			}
			safe = append(safe, map[string]interface{}{
				"path": sanitizeAmbientPath(item["path"]), "line": item["line"], "column": item["column"],
				"endLine": item["endLine"], "endColumn": item["endColumn"], "severity": item["severity"],
				"message": sanitizeAmbientText(fmt.Sprint(item["message"]), 512),
			})
		}
		if len(safe) > 0 {
			out["diagnostics"] = safe
		}
	}
	if terminal := ambientStringMap(root["terminal"]); terminal != nil {
		if tail, ok := terminal["failed_tail"].(string); ok && strings.TrimSpace(tail) != "" {
			out["terminal"] = map[string]interface{}{
				"cwd": sanitizeAmbientPath(terminal["cwd"]), "failed_tail": sanitizeAmbientText(tail, 4096),
			}
		}
	}
	if git := ambientStringMap(root["git"]); git != nil {
		safe := map[string]interface{}{}
		if branch, ok := git["branch"].(string); ok && strings.TrimSpace(branch) != "" {
			safe["branch"] = sanitizeAmbientText(branch, 256)
		}
		for _, key := range []string{"staged", "unstaged", "untracked"} {
			if rows, ok := git[key].([]interface{}); ok {
				paths := make([]interface{}, 0, min(len(rows), 40))
				for _, row := range rows {
					if len(paths) >= 40 {
						break
					}
					if path := sanitizeAmbientPath(row); path != "" {
						paths = append(paths, path)
					}
				}
				safe[key] = paths
			}
		}
		out["git"] = safe
	}
	if rows, ok := root["recent_edits"].([]interface{}); ok {
		safe := make([]interface{}, 0, min(len(rows), 20))
		for _, row := range rows {
			if len(safe) >= 20 {
				break
			}
			item := ambientStringMap(row)
			if path := sanitizeAmbientPath(item["path"]); path != "" {
				safe = append(safe, map[string]interface{}{"path": path, "edited_at": item["edited_at"]})
			}
		}
		if len(safe) > 0 {
			out["recent_edits"] = safe
		}
	}

	out["truncated"] = root["truncated"] == true
	for {
		encoded, _ := json.Marshal(out)
		if len(encoded) <= maxAmbientStateServerBytes {
			break
		}
		out["truncated"] = true
		if rows, ok := out["diagnostics"].([]interface{}); ok && len(rows) > 0 {
			out["diagnostics"] = rows[:len(rows)-1]
			continue
		}
		if rows, ok := out["recent_edits"].([]interface{}); ok && len(rows) > 0 {
			out["recent_edits"] = rows[:len(rows)-1]
			continue
		}
		if _, ok := out["terminal"]; ok {
			delete(out, "terminal")
			continue
		}
		delete(out, "git")
		break
	}
	return out
}

// SanitizeAmbientStateMetadata enforces relevance, redaction, shape and the 16KB server cap.
func SanitizeAmbientStateMetadata(msg *protocol.Message) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	raw, ok := msg.Metadata[MetadataAmbientState]
	if !ok {
		return
	}
	if !ambientStateRelevant(msg) {
		delete(msg.Metadata, MetadataAmbientState)
		return
	}
	safe := sanitizeAmbientStateValue(raw)
	if len(safe) == 0 {
		delete(msg.Metadata, MetadataAmbientState)
		return
	}
	msg.Metadata[MetadataAmbientState] = safe
}

// AppendAmbientState renders ephemeral IDE state into the current prompt only.
func AppendAmbientState(prompt *strings.Builder, msg *protocol.Message) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	state := ambientStringMap(msg.Metadata[MetadataAmbientState])
	if len(state) == 0 {
		return
	}
	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	prompt.WriteString("\n=== AMBIENT IDE STATE (EPHEMERAL) ===\n")
	prompt.WriteString("Use this only when relevant to the current request. It is a bounded snapshot and may be truncated.\n")
	prompt.Write(encoded)
	prompt.WriteString("\n=== END AMBIENT IDE STATE ===\n\n")
}
