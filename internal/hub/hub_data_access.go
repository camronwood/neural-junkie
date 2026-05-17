package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	hubDataMaxFileBytes      = 512 * 1024  // per file for agent context
	hubDataMaxTotalBytes     = 768 * 1024  // across all entries in one grant
	hubDataMaxDirFiles       = 30
	hubDataMaxDirDepth       = 4
	hubDataSkipFileBytes     = 8 * 1024 * 1024 // skip reading huge archived sessions
)

// HubDataReadTarget identifies a file or directory under ~/.neural-junkie to expose to agents.
type HubDataReadTarget struct {
	Kind         string `json:"kind"` // "file" or "directory"
	RelativePath string `json:"relative_path"`
}

// HubDataReadEntry is one file's content returned to the desktop for agent metadata.
type HubDataReadEntry struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	Bytes     int    `json:"bytes"`
	Truncated bool   `json:"truncated,omitempty"`
	Skipped   bool   `json:"skipped,omitempty"`
	Note      string `json:"note,omitempty"`
	Content   string `json:"content,omitempty"`
}

// HubDataReadResult is the response body for /api/hub-data/read.
type HubDataReadResult struct {
	Root    string             `json:"root"`
	Entries []HubDataReadEntry `json:"entries"`
}

// NeuralJunkieDataDir returns ~/.neural-junkie (or equivalent).
func NeuralJunkieDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie"), nil
}

func resolveHubDataPath(rel string) (abs, display string, err error) {
	root, err := NeuralJunkieDataDir()
	if err != nil {
		return "", "", err
	}
	rel = strings.TrimSpace(rel)
	rel = strings.TrimPrefix(rel, "~")
	rel = strings.TrimPrefix(rel, "/")
	rel = strings.TrimPrefix(rel, ".neural-junkie/")
	rel = strings.TrimPrefix(rel, ".neural-junkie")
	rel = filepath.Clean(rel)
	if rel == "." {
		rel = ""
	}
	if strings.HasPrefix(rel, "..") {
		return "", "", fmt.Errorf("path escapes hub data directory")
	}
	abs = filepath.Join(root, rel)
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	if abs != rootAbs && !strings.HasPrefix(abs, rootAbs+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path must stay under %s", rootAbs)
	}
	display = filepath.Join("~", ".neural-junkie", rel)
	if rel == "" {
		display = filepath.Join("~", ".neural-junkie")
	}
	return abs, display, nil
}

// ReadHubDataForAgent reads granted files/directories under ~/.neural-junkie with size caps.
func ReadHubDataForAgent(targets []HubDataReadTarget) (*HubDataReadResult, error) {
	root, err := NeuralJunkieDataDir()
	if err != nil {
		return nil, err
	}
	result := &HubDataReadResult{Root: root}
	total := 0

	for _, t := range targets {
		if total >= hubDataMaxTotalBytes {
			break
		}
		kind := strings.ToLower(strings.TrimSpace(t.Kind))
		switch kind {
		case "file":
			entries, n, err := readHubDataFile(t.RelativePath, hubDataMaxTotalBytes-total)
			if err != nil {
				return nil, err
			}
			result.Entries = append(result.Entries, entries...)
			total += n
		case "directory":
			entries, n, err := readHubDataDirectory(t.RelativePath, hubDataMaxTotalBytes-total)
			if err != nil {
				return nil, err
			}
			result.Entries = append(result.Entries, entries...)
			total += n
		default:
			return nil, fmt.Errorf("unknown target kind %q", t.Kind)
		}
	}
	return result, nil
}

func readHubDataFile(rel string, budget int) ([]HubDataReadEntry, int, error) {
	abs, display, err := resolveHubDataPath(rel)
	if err != nil {
		return nil, 0, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, 0, err
	}
	if fi.IsDir() {
		return nil, 0, fmt.Errorf("%s is a directory", display)
	}
	if fi.Size() > hubDataSkipFileBytes {
		entry := HubDataReadEntry{
			Path:    display,
			Kind:    "file",
			Skipped: true,
			Note:    fmt.Sprintf("file is %d MiB; use scripts/analyze-last-session.sh or grant directory with smaller files only", fi.Size()/(1024*1024)),
		}
		return []HubDataReadEntry{entry}, 0, nil
	}
	content, truncated, err := readTextFileCapped(abs, minInt(budget, hubDataMaxFileBytes))
	if err != nil {
		return nil, 0, err
	}
	entry := HubDataReadEntry{
		Path:      display,
		Kind:      "file",
		Bytes:     len(content),
		Truncated: truncated,
		Content:   content,
	}
	return []HubDataReadEntry{entry}, len(content), nil
}

func readHubDataDirectory(rel string, budget int) ([]HubDataReadEntry, int, error) {
	abs, display, err := resolveHubDataPath(rel)
	if err != nil {
		return nil, 0, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, 0, err
	}
	if !fi.IsDir() {
		return nil, 0, fmt.Errorf("%s is not a directory", display)
	}

	var entries []HubDataReadEntry
	used := 0
	filesRead := 0

	// Directory listing summary first.
	var listing strings.Builder
	listing.WriteString(fmt.Sprintf("Directory listing for %s (max %d files in context):\n", display, hubDataMaxDirFiles))
	_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == abs {
			return nil
		}
		relPath, _ := filepath.Rel(abs, path)
		if strings.Count(relPath, string(filepath.Separator)) > hubDataMaxDirDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			listing.WriteString("  [dir]  " + relPath + "\n")
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		listing.WriteString(fmt.Sprintf("  [file] %s (%d bytes)\n", relPath, info.Size()))
		return nil
	})
	listingContent := listing.String()
	if len(listingContent) > budget {
		listingContent = listingContent[:budget] + "\n... (listing truncated)\n"
	}
	entries = append(entries, HubDataReadEntry{
		Path:    display,
		Kind:    "directory",
		Bytes:   len(listingContent),
		Content: listingContent,
		Note:    "directory index",
	})
	used += len(listingContent)

	_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || filesRead >= hubDataMaxDirFiles || used >= budget {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(abs, path)
		if strings.Count(relPath, string(filepath.Separator)) > hubDataMaxDirDepth {
			return nil
		}
		if shouldSkipHubDataFileName(d.Name()) {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		if info.Size() > hubDataSkipFileBytes {
			entries = append(entries, HubDataReadEntry{
				Path:    filepath.Join(display, relPath),
				Kind:    "file",
				Skipped: true,
				Note:    "file too large to inline",
			})
			return nil
		}
		if info.Size() > int64(hubDataMaxFileBytes) {
			entries = append(entries, HubDataReadEntry{
				Path:    filepath.Join(display, relPath),
				Kind:    "file",
				Skipped: true,
				Note:    fmt.Sprintf("over per-file cap (%d KiB)", hubDataMaxFileBytes/1024),
			})
			return nil
		}
		remaining := budget - used
		if remaining <= 0 {
			return nil
		}
		content, truncated, readErr := readTextFileCapped(path, minInt(remaining, hubDataMaxFileBytes))
		if readErr != nil {
			return nil
		}
		entries = append(entries, HubDataReadEntry{
			Path:      filepath.Join(display, relPath),
			Kind:      "file",
			Bytes:     len(content),
			Truncated: truncated,
			Content:   content,
		})
		used += len(content)
		filesRead++
		return nil
	})

	return entries, used, nil
}

func shouldSkipHubDataFileName(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "last-session-") && strings.HasSuffix(lower, ".tmp") {
		return true
	}
	if strings.HasPrefix(lower, "last-session.archived-") {
		return true
	}
	ext := filepath.Ext(lower)
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".zip", ".gguf", ".safetensors":
		return true
	}
	return false
}

func readTextFileCapped(path string, maxBytes int) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	limited := io.LimitReader(f, int64(maxBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", false, err
	}
	truncated := len(data) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	// Reject binary-ish content
	if len(data) > 0 {
		for i := 0; i < len(data) && i < 8000; i++ {
			if data[i] == 0 {
				return "", false, fmt.Errorf("binary file")
			}
		}
	}
	return string(data), truncated, nil
}

// MarshalGrantedHubDataAccess formats read result for message metadata.
func MarshalGrantedHubDataAccess(result *HubDataReadResult) (map[string]interface{}, error) {
	if result == nil {
		return nil, nil
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
