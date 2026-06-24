package fileedit

import (
	"fmt"
	"strings"
)

// SelectionScope describes an optional editor selection constraint.
type SelectionScope struct {
	Path      string
	StartLine int
	EndLine   int
	Text      string
}

// ValidateSelectionScope ensures a patch stays within selection bounds when scope is set.
// Import-only changes above the selection (lines 1..startLine-1) are allowed when
// every changed old line is inside [StartLine, EndLine] or is an import line.
func ValidateSelectionScope(scope *SelectionScope, oldContent, newContent string) error {
	if scope == nil || scope.StartLine <= 0 || scope.EndLine <= 0 {
		return nil
	}
	if strings.TrimSpace(scope.Path) == "" {
		return nil
	}

	oldFrom, oldTo, _, _ := ChangedLineRange(oldContent, newContent)
	if oldFrom == 0 && oldTo == 0 {
		return nil
	}

	for line := oldFrom; line <= oldTo; line++ {
		if line >= scope.StartLine && line <= scope.EndLine {
			continue
		}
		if isImportLine(oldContent, line) {
			continue
		}
		return &PatchError{
			Code: ErrOutOfScope,
			Message: fmt.Sprintf(
				"edit touches line %d outside selection (lines %d–%d); keep changes scoped to the selection",
				line, scope.StartLine, scope.EndLine,
			),
		}
	}

	// When selection text is present, require old_string anchor to appear in selection for search_replace flows.
	if strings.TrimSpace(scope.Text) != "" {
		// Verify at least one changed region overlaps selection text.
		sel := strings.TrimSpace(scope.Text)
		oldLines := splitLines(oldContent)
		newLines := splitLines(newContent)
		changed := extractChangedSnippet(oldLines, newLines, scope.StartLine, scope.EndLine)
		if changed != "" && !strings.Contains(sel, strings.TrimSpace(changed)) && !strings.Contains(strings.TrimSpace(changed), sel) {
			// Allow when edits are subset of selection window even if snippet text diverged after patch
			if oldFrom < scope.StartLine || oldTo > scope.EndLine {
				return &PatchError{Code: ErrOutOfScope, Message: "patch must stay within the selected lines"}
			}
		}
	}
	return nil
}

func isImportLine(content string, lineNum int) bool {
	lines := splitLines(content)
	if lineNum <= 0 || lineNum > len(lines) {
		return false
	}
	trim := strings.TrimSpace(lines[lineNum-1])
	return strings.HasPrefix(trim, "import ") ||
		strings.HasPrefix(trim, "from ") ||
		strings.HasPrefix(trim, "use ") ||
		strings.HasPrefix(trim, "#include")
}

func extractChangedSnippet(oldLines, newLines []string, start, end int) string {
	var b strings.Builder
	for i := start - 1; i < end && i < len(oldLines); i++ {
		if i >= len(newLines) || oldLines[i] != newLines[i] {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(oldLines[i])
		}
	}
	return b.String()
}

// RequireOldStringInSelection returns an error when old must be anchored in selection text.
func RequireOldStringInSelection(scope *SelectionScope, old string) error {
	if scope == nil || strings.TrimSpace(scope.Text) == "" {
		return nil
	}
	normSel := strings.ReplaceAll(scope.Text, "\r\n", "\n")
	normOld := strings.ReplaceAll(old, "\r\n", "\n")
	if !strings.Contains(normSel, normOld) {
		return &PatchError{
			Code: ErrOutOfScope,
			Message: "old_string must appear inside the user selection when a selection is active",
		}
	}
	return nil
}
