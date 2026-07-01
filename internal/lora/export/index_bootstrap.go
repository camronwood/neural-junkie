package export

import (
	"fmt"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/repo"
)

const (
	bootstrapMaxRows      = 15
	archChunkSize         = 800
	archMaxChunks         = 3
	sourceFileMaxRows     = 5
	sourceFileExcerptSize = 600
)

// BootstrapFromIndex builds deterministic instruction/output rows from a repo index.
func BootstrapFromIndex(index *repo.RepositoryIndex) []Row {
	if index == nil {
		return nil
	}
	repoName := strings.TrimSpace(index.Name)
	if repoName == "" {
		repoName = "this repository"
	}
	var rows []Row
	rows = append(rows, bootstrapArchitectureRows(repoName, index.ArchitectureDoc)...)
	rows = append(rows, bootstrapKeyFileRows(index.KeyFiles)...)
	rows = append(rows, bootstrapPatternRow(repoName, index.CodePatterns)...)
	rows = append(rows, bootstrapSourceFileRows(index.SourceFiles)...)
	return dedupeImportRows(rows, bootstrapMaxRows)
}

func bootstrapArchitectureRows(repoName, doc string) []Row {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return nil
	}
	chunks := splitChunks(doc, archChunkSize, archMaxChunks)
	var out []Row
	for i, chunk := range chunks {
		ref := fmt.Sprintf("architecture:%d", i+1)
		out = append(out, Row{
			Instruction: fmt.Sprintf("Describe the architecture of %s.", repoName),
			Output:      chunk,
			SourceKind:  "index",
			SourceRef:   ref,
		})
	}
	return out
}

func bootstrapKeyFileRows(keyFiles map[string]string) []Row {
	if len(keyFiles) == 0 {
		return nil
	}
	names := make([]string, 0, len(keyFiles))
	for name := range keyFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Row
	for _, name := range names {
		content := strings.TrimSpace(keyFiles[name])
		if content == "" {
			continue
		}
		instruction := fmt.Sprintf("What does %s say about this project?", name)
		if strings.EqualFold(name, "README.md") || strings.EqualFold(name, "readme.md") {
			instruction = "What does the README say about this project?"
		}
		out = append(out, Row{
			Instruction: instruction,
			Output:      truncateText(content, sourceFileExcerptSize),
			SourceKind:  "index",
			SourceRef:   name,
		})
	}
	return out
}

func bootstrapPatternRow(repoName string, patterns []string) []Row {
	if len(patterns) == 0 {
		return nil
	}
	joined := strings.Join(patterns, ", ")
	return []Row{{
		Instruction: fmt.Sprintf("What frameworks or patterns does %s use?", repoName),
		Output:      joined,
		SourceKind:  "index",
		SourceRef:   "code_patterns",
	}}
}

func bootstrapSourceFileRows(files map[string]*repo.SourceFile) []Row {
	if len(files) == 0 {
		return nil
	}
	type ranked struct {
		path string
		file *repo.SourceFile
	}
	rankedFiles := make([]ranked, 0, len(files))
	for path, file := range files {
		if file == nil {
			continue
		}
		rankedFiles = append(rankedFiles, ranked{path: path, file: file})
	}
	sort.Slice(rankedFiles, func(i, j int) bool {
		return rankedFiles[i].file.Size > rankedFiles[j].file.Size
	})
	if len(rankedFiles) > sourceFileMaxRows {
		rankedFiles = rankedFiles[:sourceFileMaxRows]
	}
	var out []Row
	for _, rf := range rankedFiles {
		output := strings.TrimSpace(rf.file.Summary)
		if output == "" && strings.TrimSpace(rf.file.Content) != "" {
			if content, err := repo.DecompressContent(rf.file.Content); err == nil {
				output = truncateText(content, sourceFileExcerptSize)
			}
		}
		if output == "" {
			continue
		}
		out = append(out, Row{
			Instruction: fmt.Sprintf("What is the purpose of `%s`?", rf.path),
			Output:      output,
			SourceKind:  "index",
			SourceRef:   rf.path,
		})
	}
	return out
}

func splitChunks(text string, size, max int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) <= size {
		return []string{text}
	}
	var chunks []string
	for len(text) > 0 && len(chunks) < max {
		if len(text) <= size {
			chunks = append(chunks, text)
			break
		}
		cut := size
		if idx := strings.LastIndex(text[:size], "\n\n"); idx > size/2 {
			cut = idx
		}
		chunks = append(chunks, strings.TrimSpace(text[:cut]))
		text = strings.TrimSpace(text[cut:])
	}
	return chunks
}

func truncateText(text string, max int) string {
	text = strings.TrimSpace(text)
	if len(text) <= max {
		return text
	}
	return strings.TrimSpace(text[:max-1]) + "…"
}

func dedupeImportRows(rows []Row, max int) []Row {
	seen := make(map[string]struct{})
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		h := ContentHash(r.Instruction, r.Input, r.Output)
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		if strings.TrimSpace(r.SourceKind) == "" {
			r.SourceKind = "index"
		}
		out = append(out, r)
		if len(out) >= max {
			break
		}
	}
	return out
}
