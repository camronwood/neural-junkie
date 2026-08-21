package plans

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	yamlPlanSlugRE   = regexp.MustCompile(`(?i)yaml\s+plan:\s*([^\s]+)`)
	targetFileRE     = regexp.MustCompile(`(?i)target_file:\s*(\S+)`)
	projectNameRE      = regexp.MustCompile(`(?i)project_name:\s*(\S+)`)
	planForTitleRE     = regexp.MustCompile(`(?i)(?:^|\n)\s*(?:plan for|plan:)\s*(.+?)(?:\s+yaml\s+plan:|\s*$)`)
	planAnchorRE       = regexp.MustCompile(`(?i)yaml\s+plan:|(?:^|\n)\s*(?:actions|steps|todos)\s*:`)
	stepItemRE         = regexp.MustCompile(`(?i)-\s*(?:description|step|action|content):\s*`)
	numberedStepRE     = regexp.MustCompile(`(?m)^\s*\d+\.\s+(.+)$`)
	outOfScopeRE       = regexp.MustCompile(`(?i)##\s+out of scope`)
)

// Parseable reports whether content is valid Cursor-shaped plan markdown.
func Parseable(content string) bool {
	_, err := Parse(content)
	return err == nil
}

// NormalizeMarkdown repairs near-miss local-model plan output into parseable markdown.
// The second return value is true when the input was rewritten.
func NormalizeMarkdown(content string) (string, bool) {
	if Parseable(content) {
		return content, false
	}
	raw := stripMarkdownFence(strings.TrimSpace(content))
	if doc, ok := normalizePseudoPlan(raw); ok {
		return Render(doc), true
	}
	return content, false
}

// PrepareMarkdown returns parseable plan markdown, normalizing when needed.
func PrepareMarkdown(content string) string {
	if normalized, ok := NormalizeMarkdown(content); ok {
		return normalized
	}
	return content
}

func normalizePseudoPlan(raw string) (Document, bool) {
	if strings.TrimSpace(raw) == "" {
		return Document{}, false
	}
	if _, err := Parse(raw); err == nil {
		return Document{}, false
	}

	name := extractPlanName(raw)
	overview := extractPlanOverview(raw, name)
	todos := extractPseudoTodos(raw)
	if len(todos) == 0 {
		todos = extractNumberedTodos(raw)
	}
	if name == "" && overview == "" && len(todos) == 0 {
		return Document{}, false
	}
	if name == "" {
		if overview != "" {
			name = truncateWords(overview, 8)
		} else if len(todos) > 0 {
			name = truncateWords(todos[0].Content, 6)
		} else {
			name = "Implementation plan"
		}
	}
	if overview == "" {
		overview = name
	}
	for i := range todos {
		if strings.TrimSpace(todos[i].ID) == "" {
			todos[i].ID = slugify(todos[i].Content)
			if todos[i].ID == "" || todos[i].ID == "plan" {
				todos[i].ID = fmt.Sprintf("todo-%d", i+1)
			}
		}
		if strings.TrimSpace(todos[i].Status) == "" {
			todos[i].Status = "pending"
		}
	}

	body := extractPlanBody(raw)
	if !outOfScopeRE.MatchString(body) {
		if body != "" {
			body += "\n\n"
		}
		body += "## Out of scope\n\n- Follow-ups not listed in todos.\n"
	}

	return Document{
		Name:      name,
		Overview:  overview,
		Todos:     todos,
		IsProject: false,
		Body:      strings.TrimSpace(body),
	}, true
}

func extractPlanName(raw string) string {
	if m := planForTitleRE.FindStringSubmatch(raw); len(m) > 1 {
		return cleanPlanTitle(m[1])
	}
	if m := yamlPlanSlugRE.FindStringSubmatch(raw); len(m) > 1 {
		return humanizeSlug(m[1])
	}
	if m := projectNameRE.FindStringSubmatch(raw); len(m) > 1 {
		return humanizeSlug(m[1])
	}
	return ""
}

func extractPlanOverview(raw, name string) string {
	if m := targetFileRE.FindStringSubmatch(raw); len(m) > 1 {
		return "Changes targeting " + m[1] + "."
	}
	// First sentence of prose before the pseudo-yaml anchor.
	if loc := planAnchorRE.FindStringIndex(raw); loc != nil && loc[0] > 0 {
		prose := strings.TrimSpace(raw[:loc[0]])
		prose = strings.TrimPrefix(prose, "Based on my research")
		prose = strings.TrimPrefix(prose, "based on my research")
		prose = strings.TrimSpace(prose)
		if idx := strings.Index(prose, ". "); idx > 0 && idx < 180 {
			return strings.TrimSpace(prose[:idx+1])
		}
		if len(prose) > 0 && len(prose) < 220 {
			return prose
		}
	}
	if name != "" {
		return name + "."
	}
	return ""
}

func extractPseudoTodos(raw string) []Todo {
	loc := planAnchorRE.FindStringIndex(raw)
	segment := raw
	if loc != nil {
		segment = raw[loc[0]:]
	}
	parts := stepItemRE.Split(segment, -1)
	if len(parts) <= 1 {
		return nil
	}
	var todos []Todo
	for i, part := range parts[1:] {
		content := cleanTodoContent(part)
		if content == "" {
			continue
		}
		todos = append(todos, Todo{
			ID:      slugify(content),
			Content: content,
			Status:  "pending",
		})
		if i >= 11 {
			break
		}
	}
	return todos
}

func extractNumberedTodos(raw string) []Todo {
	matches := numberedStepRE.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil
	}
	var todos []Todo
	for i, m := range matches {
		content := cleanTodoContent(m[1])
		if content == "" {
			continue
		}
		todos = append(todos, Todo{
			ID:      slugify(content),
			Content: content,
			Status:  "pending",
		})
		if i >= 11 {
			break
		}
	}
	return todos
}

func extractPlanBody(raw string) string {
	if loc := planAnchorRE.FindStringIndex(raw); loc != nil && loc[0] > 0 {
		prose := strings.TrimSpace(raw[:loc[0]])
		if prose != "" {
			return "# Plan\n\n" + prose + "\n"
		}
	}
	return ""
}

func cleanPlanTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".")
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		s = strings.TrimSpace(s[:120])
	}
	return s
}

func cleanTodoContent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Trim trailing pseudo-yaml fields local models append inline.
	cutMarkers := []string{
		" location:", " implementation:", " rationale:", " verification:",
		" notes:", " target_file:", " - rationale:", " - verification:", " - notes:",
	}
	lower := strings.ToLower(s)
	for _, marker := range cutMarkers {
		if idx := strings.Index(lower, marker); idx > 0 {
			s = strings.TrimSpace(s[:idx])
			lower = strings.ToLower(s)
		}
	}
	s = strings.TrimSpace(s)
	if len(s) > 240 {
		s = strings.TrimSpace(s[:240])
	}
	return s
}

func humanizeSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	slug = strings.NewReplacer("-", " ", "_", " ", ":", " ").Replace(slug)
	words := strings.Fields(slug)
	for i, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func truncateWords(s string, max int) string {
	words := strings.Fields(strings.TrimSpace(s))
	if len(words) == 0 {
		return ""
	}
	if len(words) <= max {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:max], " ")
}
