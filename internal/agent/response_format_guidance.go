package agent

import "strings"

// personaMarkdownStructureGuidance is a light default: structure when enumerating,
// without forcing markdown on every short reply.
const personaMarkdownStructureGuidance = "When listing, comparing, or stepping through items, use GitHub-flavored markdown with one list item per line (blank line before lists and headings). Do not force markdown structure on short yes/no answers.\n\n"

// userRequestsResponseFormat is true when the user explicitly asks for a structured
// reply shape (bullets, numbered list, table, checklist, headings, or markdown).
func userRequestsResponseFormat(content string) bool {
	lower := strings.ToLower(content)
	if lower == "" {
		return false
	}
	phrases := []string{
		"bullet list",
		"bulleted list",
		"as bullets",
		"as a bullet",
		"in bullets",
		"bullet points",
		"bullet point",
		"as a list",
		"in a list",
		"as a numbered list",
		"numbered list",
		"numbered steps",
		"step by step",
		"step-by-step",
		"as a table",
		"markdown table",
		"in a table",
		"as a checklist",
		"as checklist",
		"make a checklist",
		"give me a checklist",
		"checklist format",
		"as markdown",
		"in markdown",
		"using markdown",
		"with headings",
		"use headings",
		"markdown sections",
	}
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	// Loose "give me / reply with / format as … bullets|list|table"
	if strings.Contains(lower, "bullet") && (strings.Contains(lower, "list") ||
		strings.Contains(lower, "format") || strings.Contains(lower, "give me") ||
		strings.Contains(lower, "reply") || strings.Contains(lower, "as ")) {
		return true
	}
	return false
}

// getResponseFormatGuidance returns shape-specific instructions when the user
// asked for structured output. Callers must check userRequestsResponseFormat first.
func getResponseFormatGuidance(content string) string {
	lower := strings.ToLower(content)
	base := "The user asked for a specific reply format. Honor it with real GitHub-flavored markdown — " +
		"not inline \"1. 2. 3.\" or \"- a - b\" jammed into one paragraph. " +
		"Keep a brief intro sentence, then put the structured content in the body. " +
		"One list item per line; blank line before lists and headings; sub-bullets on separate lines starting with -."

	switch {
	case strings.Contains(lower, "table"):
		return base + " Use a markdown pipe table with a header row and separator."
	case strings.Contains(lower, "checklist") || strings.Contains(lower, "check list"):
		return base + " Use a markdown checklist (- [ ] / - [x]) with one item per line."
	case strings.Contains(lower, "numbered") || strings.Contains(lower, "step by step") ||
		strings.Contains(lower, "step-by-step") || strings.Contains(lower, "steps"):
		return base + " Use an ordered list (1. 2. 3.) with one item per line."
	case strings.Contains(lower, "heading") || strings.Contains(lower, "sections"):
		return base + " Use ### headings for each section."
	case strings.Contains(lower, "bullet") || strings.Contains(lower, "list"):
		return base + " Use an unordered list (- ) with one item per line."
	default:
		return base + " Match the format they named (lists, headings, or tables as appropriate)."
	}
}
