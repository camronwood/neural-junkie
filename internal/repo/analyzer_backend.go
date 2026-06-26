package repo

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// AnalyzeViaBackend builds a repository index by reading files through a workspace backend.
func (a *Analyzer) AnalyzeViaBackend(ctx context.Context, logicalPath string, backend workspacebackend.Backend) (*RepositoryIndex, error) {
	if backend == nil {
		return nil, fmt.Errorf("backend is required")
	}
	logicalPath = filepath.Clean(strings.TrimSpace(logicalPath))
	if logicalPath == "" {
		logicalPath = backend.Root()
	}

	a.updateProgress(0, "Starting remote repository analysis...")

	index := &RepositoryIndex{
		Path:         logicalPath,
		Name:         filepath.Base(strings.TrimRight(logicalPath, "/")),
		LastIndexed:  time.Now(),
		KeyFiles:     make(map[string]string),
		Dependencies: make(map[string][]string),
		CodePatterns: []string{},
		FileModTimes: make(map[string]time.Time),
		SourceFiles:  make(map[string]*SourceFile),
	}
	if index.Name == "" || index.Name == "." {
		index.Name = "workspace"
	}

	a.updateProgress(10, "Listing files...")
	files, err := workspacebackend.ListFilesRecursive(ctx, backend, ".", MaxFiles)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	index.FileCount = len(files)

	a.updateProgress(20, "Extracting key files...")
	for _, rel := range files {
		base := filepath.Base(rel)
		for _, keyFile := range KeyFileTypes {
			if base != keyFile && rel != keyFile {
				continue
			}
			data, err := backend.ReadFile(ctx, rel)
			if err != nil || len(data) > MaxFileSize {
				continue
			}
			index.KeyFiles[keyFile] = string(data)
			break
		}
	}

	a.updateProgress(30, "Indexing source code files...")
	stats := &CompressionStats{}
	cappedLogged := false
	for i, rel := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		ext := strings.ToLower(filepath.Ext(rel))
		lang, ok := LanguageExtensions[ext]
		if !ok {
			continue
		}
		data, err := backend.ReadFile(ctx, rel)
		if err != nil || int64(len(data)) > MaxFileSize {
			continue
		}
		index.TotalSize += int64(len(data))
		content := string(data)
		compressed, compressedSize, err := CompressContent(content)
		if err != nil {
			continue
		}
		if len(index.SourceFiles) >= MaxIndexedSourceFilesInMemory {
			if !cappedLogged {
				log.Printf("[repo] backend source index capped at %d files", MaxIndexedSourceFilesInMemory)
				cappedLogged = true
			}
			break
		}
		relSlash := filepath.ToSlash(rel)
		index.SourceFiles[relSlash] = &SourceFile{
			Path:           relSlash,
			Language:       lang,
			Size:           int64(len(data)),
			CompressedSize: compressedSize,
			Content:        compressed,
			ModTime:        time.Now(),
		}
		stats.OriginalSize += int64(len(data))
		stats.CompressedSize += compressedSize
		if i%10 == 0 {
			a.updateProgress(30+int(float64(i)/float64(len(files))*30), fmt.Sprintf("Indexed %d source files...", len(index.SourceFiles)))
		}
	}

	a.updateProgress(75, "Reading git history...")
	if res, err := backend.Exec(ctx, workspacebackend.ExecRequest{
		Command: "git",
		Args:    []string{"log", "-5", "--format=%H|%an|%aI|%s"},
		RelCwd:  ".",
		Timeout: 15 * time.Second,
	}); err == nil && res.ExitCode == 0 {
		index.GitInfo = parseGitLogOutput(res.Stdout)
	}

	a.updateProgress(85, "Identifying code patterns...")
	index.CodePatterns = a.identifyCodePatterns(index)

	a.updateProgress(95, "Generating architecture overview...")
	index.ArchitectureDoc = a.generateArchitectureDoc(index)
	a.updateProgress(100, "Analysis complete!")

	TrimRepositoryIndexFootprint(index)
	return index, nil
}

func parseGitLogOutput(stdout string) *GitInfo {
	info := &GitInfo{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		date, _ := time.Parse(time.RFC3339, parts[2])
		ci := CommitInfo{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    date,
			Message: parts[3],
		}
		info.RecentCommits = append(info.RecentCommits, ci)
		if info.LastCommit == "" {
			info.LastCommit = ci.Hash
			info.LastCommitMsg = ci.Message
			info.LastCommitDate = ci.Date
		}
	}
	return info
}
