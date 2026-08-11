// Package canvasdoc is the Neural Canvas block-document contract.
// Agents write markdown or JSON; the host unwraps, compiles, and renders blocks.
package canvasdoc

import (
	"encoding/json"
	"strings"
)

const (
	RendererID    = "nj.document"
	MediaType     = "application/vnd.neural-junkie.document+json"
	SchemaVersion = 1

	TypeHeading  = "heading"
	TypeMarkdown = "markdown"
	TypeList     = "list"
	TypeTable    = "table"
	TypeCallout  = "callout"
	TypeMermaid  = "mermaid"
	TypeImage    = "image"
	TypeColumns  = "columns"
)

// Document is the nj.document payload.
type Document struct {
	SchemaVersion int     `json:"schema_version"`
	Title         string  `json:"title,omitempty"`
	Blocks        []Block `json:"blocks"`
}

// Block is one hosted canvas element. Discriminator is type, never kind.
type Block struct {
	Type    string              `json:"type"`
	Level   int                 `json:"level,omitempty"`
	Text    string              `json:"text,omitempty"`
	Source  string              `json:"source,omitempty"`
	Ordered bool                `json:"ordered,omitempty"`
	Items   []string            `json:"items,omitempty"`
	Columns []TableColumn       `json:"columns,omitempty"`
	Rows    []map[string]string `json:"rows,omitempty"`
	Tone    string              `json:"tone,omitempty"`
	Title   string              `json:"title,omitempty"`
	Body    string              `json:"body,omitempty"`
	Src     string              `json:"src,omitempty"`
	Alt     string              `json:"alt,omitempty"`
	Caption string              `json:"caption,omitempty"`
	Cols    [][]Block           `json:"cols,omitempty"`
}

// TableColumn matches the nj.table renderer contract.
type TableColumn struct {
	Key   string `json:"key"`
	Label string `json:"label,omitempty"`
}

// Blank returns an empty titled page.
func Blank(title string) Document {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Canvas"
	}
	return Document{
		SchemaVersion: SchemaVersion,
		Title:         title,
		Blocks: []Block{{
			Type:  TypeHeading,
			Level: 1,
			Text:  title,
		}},
	}
}

// Normalize fills schema_version and drops nil blocks.
func Normalize(doc Document) Document {
	doc.SchemaVersion = SchemaVersion
	if doc.Blocks == nil {
		doc.Blocks = []Block{}
	}
	return doc
}

// Parse accepts a document JSON object (optionally fenced). Returns false when
// the text is not a block document.
func Parse(raw string) (Document, bool) {
	raw = strings.TrimSpace(stripOuterFence(raw))
	if raw == "" || raw[0] != '{' {
		return Document{}, false
	}
	var doc Document
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return Document{}, false
	}
	if doc.Blocks == nil {
		return Document{}, false
	}
	return Normalize(doc), true
}

// FromModelOutput prefers document JSON and otherwise compiles markdown.
func FromModelOutput(raw string) Document {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Document{SchemaVersion: SchemaVersion, Blocks: []Block{}}
	}
	if doc, ok := Parse(raw); ok {
		return doc
	}
	return CompileMarkdown(raw)
}

// Unwrap accepts every historical markdown payload shape and a document object.
func Unwrap(payload json.RawMessage) Document {
	if len(payload) == 0 {
		return Document{SchemaVersion: SchemaVersion, Blocks: []Block{}}
	}
	trimmed := strings.TrimSpace(string(payload))
	if doc, ok := Parse(trimmed); ok {
		return doc
	}
	var asString string
	if err := json.Unmarshal(payload, &asString); err == nil {
		if doc, ok := Parse(asString); ok {
			return doc
		}
		return CompileMarkdown(asString)
	}
	var asObj map[string]any
	if err := json.Unmarshal(payload, &asObj); err == nil {
		for _, key := range []string{"markdown", "content", "text", "body"} {
			if v, ok := asObj[key].(string); ok {
				return CompileMarkdown(v)
			}
		}
	}
	return CompileMarkdown(string(payload))
}

// Marshal returns the JSON payload and a markdown fallback.
func Marshal(doc Document) (json.RawMessage, string, error) {
	doc = Normalize(doc)
	fallback := ToMarkdown(doc)
	data, err := json.Marshal(doc)
	return data, fallback, err
}

// IsPageRenderer reports markdown or document canvases (the collaborative page).
func IsPageRenderer(id, media string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	media = strings.ToLower(strings.TrimSpace(media))
	if id == RendererID || id == "nj.markdown" {
		return true
	}
	return strings.Contains(media, "markdown") || strings.Contains(media, "document+json")
}

// FirstHeading returns the first heading text.
func FirstHeading(doc Document) string {
	for _, block := range doc.Blocks {
		if block.Type == TypeHeading && strings.TrimSpace(block.Text) != "" {
			return strings.TrimSpace(block.Text)
		}
	}
	return ""
}

// EnsureHeading sets or prepends an H1.
func EnsureHeading(doc Document, title string) Document {
	title = strings.TrimSpace(title)
	if title == "" {
		return doc
	}
	for i, block := range doc.Blocks {
		if block.Type == TypeHeading && block.Level <= 1 {
			doc.Blocks[i].Text = title
			doc.Blocks[i].Level = 1
			return doc
		}
	}
	doc.Blocks = append([]Block{{Type: TypeHeading, Level: 1, Text: title}}, doc.Blocks...)
	return doc
}

func stripOuterFence(raw string) string {
	s := strings.TrimSpace(raw)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		lang := strings.TrimSpace(s[:nl])
		if strings.EqualFold(lang, "json") || strings.EqualFold(lang, "document") || lang == "" {
			s = s[nl+1:]
		}
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
