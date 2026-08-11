package canvasdoc

import (
	"strings"
)

// ToMarkdown flattens a document for fallbacks, prompts, and export.
func ToMarkdown(doc Document) string {
	var b strings.Builder
	for i, block := range doc.Blocks {
		chunk := blockMarkdown(block)
		if chunk == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(chunk)
		_ = i
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return ""
	}
	return out + "\n"
}

func blockMarkdown(block Block) string {
	switch block.Type {
	case TypeHeading:
		level := block.Level
		if level < 1 {
			level = 1
		}
		if level > 3 {
			level = 3
		}
		return strings.Repeat("#", level) + " " + strings.TrimSpace(block.Text)
	case TypeMarkdown:
		return strings.TrimSpace(block.Source)
	case TypeList:
		var b strings.Builder
		for i, item := range block.Items {
			if i > 0 {
				b.WriteByte('\n')
			}
			if block.Ordered {
				b.WriteString(itoa(i + 1))
				b.WriteString(". ")
			} else {
				b.WriteString("- ")
			}
			b.WriteString(item)
		}
		return b.String()
	case TypeTable:
		return tableMarkdown(block)
	case TypeCallout:
		title := strings.TrimSpace(block.Title)
		body := strings.TrimSpace(block.Body)
		if title != "" && body != "" {
			return "> **" + title + "**\n>\n> " + strings.ReplaceAll(body, "\n", "\n> ")
		}
		if title != "" {
			return "> **" + title + "**"
		}
		if body != "" {
			return "> " + strings.ReplaceAll(body, "\n", "\n> ")
		}
		return ""
	case TypeMermaid:
		src := strings.TrimSpace(block.Source)
		if src == "" {
			return ""
		}
		return "```mermaid\n" + src + "\n```"
	case TypeImage:
		alt := block.Alt
		if alt == "" {
			alt = strings.TrimSpace(block.Caption)
		}
		if block.Src == "" {
			return ""
		}
		return "![" + alt + "](" + block.Src + ")"
	case TypeColumns:
		var parts []string
		for _, col := range block.Cols {
			var inner []string
			for _, child := range col {
				if chunk := blockMarkdown(child); chunk != "" {
					inner = append(inner, chunk)
				}
			}
			if len(inner) > 0 {
				parts = append(parts, strings.Join(inner, "\n\n"))
			}
		}
		return strings.Join(parts, "\n\n")
	default:
		return ""
	}
}

func tableMarkdown(block Block) string {
	if len(block.Columns) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('|')
	for _, col := range block.Columns {
		label := col.Label
		if label == "" {
			label = col.Key
		}
		b.WriteByte(' ')
		b.WriteString(label)
		b.WriteString(" |")
	}
	b.WriteByte('\n')
	b.WriteByte('|')
	for range block.Columns {
		b.WriteString(" --- |")
	}
	for _, row := range block.Rows {
		b.WriteByte('\n')
		b.WriteByte('|')
		for _, col := range block.Columns {
			b.WriteByte(' ')
			b.WriteString(row[col.Key])
			b.WriteString(" |")
		}
	}
	return b.String()
}
