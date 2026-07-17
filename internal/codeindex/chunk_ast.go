package codeindex

import (
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacesymbols"
)

// chunkFileAST splits source into logical chunks on function/type symbol boundaries when possible.
func chunkFileAST(relPath, content string) []Chunk {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(relPath))
	switch ext {
	case ".go", ".rs", ".py", ".js", ".ts", ".tsx", ".jsx":
		if bounds := workspacesymbols.DefinitionLines(relPath, content); len(bounds) > 0 {
			return chunkBySymbolLines(relPath, lines, bounds)
		}
		return chunkByBlankLines(relPath, lines)
	default:
		return chunkLines(relPath, lines, defaultChunkLines)
	}
}

// chunkBySymbolLines splits at definition start lines from the symbol indexer.
func chunkBySymbolLines(relPath string, lines []string, starts []int) []Chunk {
	if len(starts) == 0 {
		return chunkByBlankLines(relPath, lines)
	}
	var chunks []Chunk
	// Preamble before first symbol.
	if starts[0] > 1 {
		text := strings.Join(lines[0:starts[0]-1], "\n")
		if strings.TrimSpace(text) != "" {
			chunks = append(chunks, Chunk{
				ID: chunkID(relPath, 1, starts[0]-1), Path: relPath,
				Start: 1, End: starts[0] - 1, Content: text,
			})
		}
	}
	for i, start := range starts {
		end := len(lines)
		if i+1 < len(starts) {
			end = starts[i+1] - 1
		}
		if end < start {
			end = start
		}
		if end > len(lines) {
			end = len(lines)
		}
		text := strings.Join(lines[start-1:end], "\n")
		if strings.TrimSpace(text) == "" {
			continue
		}
		// Cap oversized symbol bodies.
		if end-start+1 > defaultChunkLines {
			for s := start; s <= end; s += defaultChunkLines {
				e := s + defaultChunkLines - 1
				if e > end {
					e = end
				}
				part := strings.Join(lines[s-1:e], "\n")
				chunks = append(chunks, Chunk{
					ID: chunkID(relPath, s, e), Path: relPath,
					Start: s, End: e, Content: part,
				})
			}
			continue
		}
		chunks = append(chunks, Chunk{
			ID: chunkID(relPath, start, end), Path: relPath,
			Start: start, End: end, Content: text,
		})
	}
	if len(chunks) == 0 {
		return chunkByBlankLines(relPath, lines)
	}
	return chunks
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
