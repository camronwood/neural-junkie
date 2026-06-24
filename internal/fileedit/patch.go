// Package fileedit provides Cursor-style patch operations for agent file edits.
package fileedit

import (
	"fmt"
	"strings"
)

// SearchReplace applies an exact substring replacement.
// When replaceAll is false, old must match exactly once.
func SearchReplace(content, old, new string, replaceAll bool) (result string, err error) {
	old = strings.ReplaceAll(old, "\r\n", "\n")
	new = strings.ReplaceAll(new, "\r\n", "\n")
	normalized := strings.ReplaceAll(content, "\r\n", "\n")

	count := strings.Count(normalized, old)
	if count == 0 {
		return "", &PatchError{Code: ErrNotFound, Message: "old_string not found in file"}
	}
	if !replaceAll && count > 1 {
		return "", &PatchError{Code: ErrNotUnique, Message: fmt.Sprintf("old_string matches %d times; include more context or set replace_all", count)}
	}
	if replaceAll {
		return strings.ReplaceAll(normalized, old, new), nil
	}
	return strings.Replace(normalized, old, new, 1), nil
}

// SearchReplaceWithFallback tries exact match, then common normalizations.
func SearchReplaceWithFallback(content, old, new string, replaceAll bool) (result string, strategy string, err error) {
	if result, err = SearchReplace(content, old, new, replaceAll); err == nil {
		return result, "exact", nil
	}
	if pe, ok := err.(*PatchError); !ok || pe.Code != ErrNotFound {
		return "", "", err
	}

	// Trim trailing newline mismatch.
	trimmedOld := strings.TrimRight(old, "\n")
	trimmedNew := strings.TrimRight(new, "\n")
	if trimmedOld != old {
		if result, err = SearchReplace(content, trimmedOld, trimmedNew, replaceAll); err == nil {
			return result, "trimmed_old_newline", nil
		}
	}

	// Expand tabs vs spaces is intentionally not attempted — too risky.

	return "", "", &PatchError{Code: ErrNotFound, Message: "old_string not found in file (exact match required)"}
}

// Hunk is one unified-diff hunk.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// DiffLine is one line in a hunk.
type DiffLine struct {
	Kind byte // ' ', '-', '+'
	Text string
}

// ApplyUnifiedPatch applies a unified diff patch to content.
func ApplyUnifiedPatch(content, patch string) (string, error) {
	hunks, err := ParseUnifiedPatch(patch)
	if err != nil {
		return "", err
	}
	lines := splitLines(content)
	for _, h := range hunks {
		lines, err = applyHunk(lines, h)
		if err != nil {
			return "", err
		}
	}
	return joinLines(lines), nil
}

func applyHunk(lines []string, h Hunk) ([]string, error) {
	idx := h.OldStart - 1
	if idx < 0 {
		idx = 0
	}
	if idx > len(lines) {
		return nil, &PatchError{Code: ErrApplyFailed, Message: fmt.Sprintf("hunk oldStart %d beyond file length %d", h.OldStart, len(lines))}
	}

	var out []string
	out = append(out, lines[:idx]...)
	cursor := idx

	for _, dl := range h.Lines {
		switch dl.Kind {
		case ' ':
			if cursor >= len(lines) {
				return nil, &PatchError{Code: ErrApplyFailed, Message: "unexpected context line beyond file end"}
			}
			if lines[cursor] != dl.Text {
				return nil, &PatchError{
					Code:    ErrApplyFailed,
					Message: fmt.Sprintf("context mismatch at line %d: expected %q got %q", cursor+1, dl.Text, lines[cursor]),
				}
			}
			out = append(out, dl.Text)
			cursor++
		case '-':
			if cursor >= len(lines) {
				return nil, &PatchError{Code: ErrApplyFailed, Message: "unexpected removal beyond file end"}
			}
			if lines[cursor] != dl.Text {
				return nil, &PatchError{
					Code:    ErrApplyFailed,
					Message: fmt.Sprintf("removal mismatch at line %d: expected %q got %q", cursor+1, dl.Text, lines[cursor]),
				}
			}
			cursor++
		case '+':
			out = append(out, dl.Text)
		default:
			return nil, &PatchError{Code: ErrApplyFailed, Message: "invalid diff line kind"}
		}
	}
	out = append(out, lines[cursor:]...)
	return out, nil
}

// ParseUnifiedPatch parses unified diff text into hunks.
func ParseUnifiedPatch(patch string) ([]Hunk, error) {
	patch = strings.ReplaceAll(patch, "\r\n", "\n")
	lines := strings.Split(patch, "\n")
	var hunks []Hunk
	var cur *Hunk

	flush := func() {
		if cur != nil && len(cur.Lines) > 0 {
			hunks = append(hunks, *cur)
		}
		cur = nil
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			flush()
			h, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			cur = &h
			continue
		}
		if cur == nil {
			continue
		}
		if len(line) == 0 {
			continue
		}
		kind := line[0]
		if kind != ' ' && kind != '-' && kind != '+' {
			continue
		}
		text := ""
		if len(line) > 1 {
			text = line[1:]
		}
		cur.Lines = append(cur.Lines, DiffLine{Kind: kind, Text: text})
	}
	flush()
	if len(hunks) == 0 {
		return nil, &PatchError{Code: ErrInvalidPatch, Message: "no hunks found in patch"}
	}
	return hunks, nil
}

func parseHunkHeader(line string) (Hunk, error) {
	// @@ -oldStart,oldCount +newStart,newCount @@
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@@") {
		return Hunk{}, &PatchError{Code: ErrInvalidPatch, Message: "invalid hunk header"}
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "@@"))
	line = strings.TrimSpace(strings.TrimSuffix(line, "@@"))
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return Hunk{}, &PatchError{Code: ErrInvalidPatch, Message: "invalid hunk header fields"}
	}
	oldStart, oldCount, err := parseRange(parts[0])
	if err != nil {
		return Hunk{}, err
	}
	newStart, newCount, err := parseRange(parts[1])
	if err != nil {
		return Hunk{}, err
	}
	return Hunk{OldStart: oldStart, OldCount: oldCount, NewStart: newStart, NewCount: newCount}, nil
}

func parseRange(s string) (start, count int, err error) {
	if len(s) == 0 {
		return 0, 0, &PatchError{Code: ErrInvalidPatch, Message: "empty range"}
	}
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		fmt.Sscanf(s[:i], "%d", &start)
		fmt.Sscanf(s[i+1:], "%d", &count)
	} else {
		fmt.Sscanf(s, "%d", &start)
		count = 1
	}
	if start <= 0 {
		return 0, 0, &PatchError{Code: ErrInvalidPatch, Message: "range start must be >= 1"}
	}
	if count < 0 {
		count = 0
	}
	return start, count, nil
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}
