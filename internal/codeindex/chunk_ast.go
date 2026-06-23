package codeindex

import (
	"strings"

	"path/filepath"
)

// chunkFileAST splits source into logical chunks on function/type boundaries when possible.
func chunkFileAST(relPath, content string) []Chunk {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(relPath))
	switch ext {
	case ".go", ".rs", ".py", ".js", ".ts", ".tsx", ".jsx":
		return chunkByBlankLines(relPath, lines)
	default:
		return chunkLines(relPath, lines, defaultChunkLines)
	}
}

func chunkByBlankLines(relPath string, lines []string) []Chunk {
	var chunks []Chunk
	var buf []string
	start := 1
	flush := func(end int) {
		if len(buf) == 0 {
			return
		}
		text := strings.Join(buf, "\n")
		if strings.TrimSpace(text) == "" {
			buf = nil
			return
		}
		chunks = append(chunks, Chunk{
			ID:      chunkID(relPath, start, end),
			Path:    relPath,
			Start:   start,
			End:     end,
			Content: text,
		})
		buf = nil
	}
	for i, line := range lines {
		ln := i + 1
		if strings.TrimSpace(line) == "" && len(buf) > 0 {
			flush(ln - 1)
			start = ln + 1
			continue
		}
		if len(buf) == 0 {
			start = ln
		}
		buf = append(buf, line)
		if len(buf) >= defaultChunkLines {
			flush(ln)
			start = ln + 1
		}
	}
	if len(buf) > 0 {
		flush(len(lines))
	}
	return chunks
}

func chunkLines(relPath string, lines []string, size int) []Chunk {
	var out []Chunk
	for i := 0; i < len(lines); i += size {
		end := i + size
		if end > len(lines) {
			end = len(lines)
		}
		text := strings.Join(lines[i:end], "\n")
		start := i + 1
		out = append(out, Chunk{
			ID:      chunkID(relPath, start, end),
			Path:    relPath,
			Start:   start,
			End:     end,
			Content: text,
		})
	}
	return out
}

func chunkID(relPath string, start, end int) string {
	return relPath + ":" + itoa(start) + "-" + itoa(end)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
