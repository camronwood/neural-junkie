package fileedit

import (
	"crypto/sha256"
	"encoding/hex"
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
type PatchError struct {
	Code    string
	Message string
}

func (e *PatchError) Error() string {
	if e == nil {
		return "patch error"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
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
