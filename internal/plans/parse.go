package plans

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Todo is a Cursor-shaped plan checklist item.
type Todo struct {
	ID      string `json:"id" yaml:"id"`
	Content string `json:"content" yaml:"content"`
	Status  string `json:"status" yaml:"status"`
}

type frontmatter struct {
	Name      string `yaml:"name"`
	Overview  string `yaml:"overview"`
	Todos     []Todo `yaml:"todos"`
	IsProject bool   `yaml:"isProject"`
}

// Document is a parsed plan markdown file.
type Document struct {
	Name      string
	Overview  string
	Todos     []Todo
	IsProject bool
	Body      string
	Raw       string
}

// Parse extracts YAML frontmatter from Cursor-shaped plan markdown.
func Parse(content string) (Document, error) {
	raw := strings.TrimSpace(content)
	if raw == "" {
		return Document{}, fmt.Errorf("empty plan")
	}
	raw = stripMarkdownFence(raw)
	fm, body, err := splitFrontmatter(raw)
	if err != nil {
		return Document{}, err
	}
	var meta frontmatter
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return Document{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	name := strings.TrimSpace(meta.Name)
	overview := strings.TrimSpace(meta.Overview)
	if name == "" && overview == "" {
		return Document{}, fmt.Errorf("plan frontmatter missing name and overview")
	}
	if len(meta.Todos) == 0 {
		return Document{}, fmt.Errorf("plan frontmatter missing todos")
	}
	for i := range meta.Todos {
		if strings.TrimSpace(meta.Todos[i].Status) == "" {
			meta.Todos[i].Status = "pending"
		}
		if strings.TrimSpace(meta.Todos[i].ID) == "" {
			meta.Todos[i].ID = fmt.Sprintf("todo-%d", i+1)
		}
	}
	return Document{
		Name:      name,
		Overview:  overview,
		Todos:     meta.Todos,
		IsProject: meta.IsProject,
		Body:      strings.TrimSpace(body),
		Raw:       strings.TrimSpace(content),
	}, nil
}

func stripMarkdownFence(s string) string {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	rest := trimmed[3:]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		lang := strings.TrimSpace(rest[:nl])
		if lang == "" || strings.EqualFold(lang, "markdown") || strings.EqualFold(lang, "md") || strings.EqualFold(lang, "yaml") {
			rest = rest[nl+1:]
		}
	}
	if strings.HasSuffix(rest, "```") {
		rest = strings.TrimSpace(rest[:len(rest)-3])
	}
	return strings.TrimSpace(rest)
}

func splitFrontmatter(s string) (front string, body string, err error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "---") {
		return "", "", fmt.Errorf("missing yaml frontmatter")
	}
	rest := strings.TrimPrefix(s, "---")
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", "", fmt.Errorf("unclosed yaml frontmatter")
	}
	front = rest[:end]
	body = strings.TrimSpace(rest[end+len("\n---"):])
	body = strings.TrimPrefix(body, "\n")
	return front, body, nil
}
