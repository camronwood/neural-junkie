package fileedit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Patch error codes.
const (
	ErrNotFound     = "not_found"
	ErrNotUnique    = "not_unique"
	ErrApplyFailed  = "apply_failed"
	ErrInvalidPatch = "invalid_patch"
	ErrOutOfScope   = "out_of_scope"
)

// PatchError describes a patch operation failure.
// Optional Fingerprint/Near/Hint enrich miss and mismatch repair loops.
type PatchError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Near        string `json:"near,omitempty"`
	Hint        string `json:"hint,omitempty"`
}

func (e *PatchError) Error() string {
	if e == nil {
		return "patch error"
	}
	if e.Fingerprint != "" || e.Near != "" || e.Hint != "" {
		return e.JSONString()
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// JSONString returns a structured tool-error payload for model repair loops.
func (e *PatchError) JSONString() string {
	if e == nil {
		return `{"code":"patch_error","message":"patch error"}`
	}
	b, err := json.Marshal(struct {
		Code        string `json:"code"`
		Message     string `json:"message"`
		Fingerprint string `json:"fingerprint,omitempty"`
		Near        string `json:"near,omitempty"`
		Hint        string `json:"hint,omitempty"`
	}{
		Code:        e.Code,
		Message:     e.Message,
		Fingerprint: e.Fingerprint,
		Near:        e.Near,
		Hint:        e.Hint,
	})
	if err != nil {
		return fmt.Sprintf(`{"code":%q,"message":%q}`, e.Code, e.Message)
	}
	return string(b)
}

// EnrichWithFileHints attaches fingerprint/near/hint for miss/unique/apply repair.
func EnrichWithFileHints(err error, fileContent, needle string) error {
	pe, ok := err.(*PatchError)
	if !ok || pe == nil {
		return err
	}
	switch pe.Code {
	case ErrNotFound, ErrNotUnique, ErrApplyFailed:
	default:
		return err
	}
	out := &PatchError{
		Code:        pe.Code,
		Message:     pe.Message,
		Fingerprint: ContentFingerprint(fileContent),
		Near:        ClosestLineHint(fileContent, needle),
		Hint:        "read_file and copy exact content, then retry with a unique old_string or refreshed patch",
	}
	if pe.Code == ErrNotUnique {
		out.Hint = "include more surrounding context in old_string or set replace_all"
	}
	if pe.Code == ErrApplyFailed {
		out.Hint = "read_file for current content and regenerate the unified diff"
		if out.Near == "" && strings.Contains(pe.Message, "expected ") {
			// Fall back to embedding the mismatch line when ClosestLineHint finds nothing.
			out.Near = truncateHint(pe.Message, 120)
		}
	}
	return out
}

// ContentFingerprint returns a short hash of file content for repair hints.
func ContentFingerprint(content string) string {
	sum := sha256.Sum256([]byte(strings.ReplaceAll(content, "\r\n", "\n")))
	return hex.EncodeToString(sum[:6])
}

// ClosestLineHint returns a short preview of a file line that partially matches needle.
func ClosestLineHint(content, needle string) string {
	needle = strings.TrimSpace(strings.ReplaceAll(needle, "\r\n", "\n"))
	if needle == "" {
		return ""
	}
	firstNeedleLine := strings.Split(needle, "\n")[0]
	firstNeedleLine = strings.TrimSpace(firstNeedleLine)
	if firstNeedleLine == "" {
		return ""
	}
	lines := splitLines(content)
	bestIdx := -1
	bestScore := 0
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == firstNeedleLine {
			return fmt.Sprintf("L%d:%q", i+1, truncateHint(trim, 80))
		}
		score := 0
		if strings.Contains(trim, firstNeedleLine) || strings.Contains(firstNeedleLine, trim) {
			score = 2
		} else if len(trim) > 8 && len(firstNeedleLine) > 8 {
			// crude prefix overlap
			n := minInt(len(trim), len(firstNeedleLine), 24)
			if trim[:n] == firstNeedleLine[:n] {
				score = 1
			}
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return ""
	}
	return fmt.Sprintf("L%d:%q", bestIdx+1, truncateHint(strings.TrimSpace(lines[bestIdx]), 80))
}

func truncateHint(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func minInt(vals ...int) int {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
