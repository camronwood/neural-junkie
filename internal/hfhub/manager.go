package hfhub

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

// DownloadProgress is reported during HF file downloads (SSE-friendly).
type DownloadProgress struct {
	Status    string  `json:"status"`
	RepoID    string  `json:"repo_id,omitempty"`
	Filename  string  `json:"filename,omitempty"`
	Total     int64   `json:"total,omitempty"`
	Completed int64   `json:"completed,omitempty"`
	Percent   float64 `json:"percent,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// LocalFile describes a cached GGUF on disk.
type LocalFile struct {
	RepoID   string `json:"repo_id"`
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
}

// Manager handles Hugging Face Hub downloads into a local cache.
type Manager struct {
	cacheDir   string
	httpClient *http.Client
	jobsMu     sync.Mutex
	jobs       map[string]*downloadJob
}

// NewManager creates a download manager with the given cache directory.
func NewManager(cacheDir string) (*Manager, error) {
	if cacheDir == "" {
		var err error
		cacheDir, err = defaultCacheDir()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &Manager{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 0, // downloads set per-request context
		},
		jobs: make(map[string]*downloadJob),
	}, nil
}

func defaultCacheDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv("HF_HOME")); v != "" {
		return filepath.Join(v, "hub"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "huggingface", "hub"), nil
}

// CacheDir returns the root cache path.
func (m *Manager) CacheDir() string {
	return m.cacheDir
}

func repoToCacheFolder(repoID string) string {
	safe := strings.ReplaceAll(repoID, "/", "--")
	return "models--" + safe
}

func (m *Manager) snapshotDir(repoID string) string {
	return filepath.Join(m.cacheDir, repoToCacheFolder(repoID), "snapshots", "main")
}

func (m *Manager) filePath(repoID, filename string) string {
	return filepath.Join(m.snapshotDir(repoID), filename)
}

// Download starts (if needed) and watches a hub-side background download.
func (m *Manager) Download(ctx context.Context, repoID, filename, token string, onProgress func(DownloadProgress)) error {
	if err := m.EnsureDownloadStarted(token, repoID, filename); err != nil {
		return err
	}
	err := m.WatchDownload(ctx, repoID, filename, onProgress)
	if err != nil && err == context.Canceled {
		// Client disconnected; download continues in background.
		return nil
	}
	return err
}

func catalogHasMode(entry *LibraryModel, mode string) bool {
	for _, m := range entry.Modes {
		if m == mode {
			return true
		}
	}
	return false
}

// ListLocal returns GGUF files found under the cache for catalog repos.
func (m *Manager) ListLocal() ([]LocalFile, error) {
	models, err := Library()
	if err != nil {
		return nil, err
	}
	var out []LocalFile
	for _, entry := range models {
		if !catalogHasMode(&entry, "local") {
			continue
		}
		for _, f := range entry.Files {
			p := m.filePath(entry.RepoID, f.Filename)
			st, err := os.Stat(p)
			if err != nil || st.IsDir() {
				continue
			}
			out = append(out, LocalFile{
				RepoID:   entry.RepoID,
				Filename: f.Filename,
				Path:     p,
				Size:     st.Size(),
			})
		}
	}
	return out, nil
}

// Delete removes a cached file for a catalog repo.
func (m *Manager) Delete(repoID, filename string) error {
	entry, err := FindCatalogEntry(repoID)
	if err != nil {
		return err
	}
	filename, err = ResolveDownloadFilename(entry, filename)
	if err != nil {
		return err
	}
	p := m.filePath(repoID, filename)
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Remove empty snapshot dirs
	_ = os.Remove(m.snapshotDir(repoID))
	_ = os.Remove(filepath.Dir(m.snapshotDir(repoID)))
	_ = os.Remove(filepath.Join(m.cacheDir, repoToCacheFolder(repoID)))
	return nil
}

// LocalPath returns the on-disk path if the file exists.
func (m *Manager) LocalPath(repoID, filename string) (string, error) {
	entry, err := FindCatalogEntry(repoID)
	if err != nil {
		return "", err
	}
	filename, err = ResolveDownloadFilename(entry, filename)
	if err != nil {
		return "", err
	}
	p := m.filePath(repoID, filename)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("file not downloaded: %w", err)
	}
	return p, nil
}

// RouterReachable checks HF router with a short timeout.
func RouterReachable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://router.huggingface.co/health", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// health endpoint may not exist; try HEAD on v1
		req2, err2 := http.NewRequestWithContext(ctx, http.MethodHead, "https://router.huggingface.co/v1", nil)
		if err2 != nil {
			return false
		}
		resp2, err2 := http.DefaultClient.Do(req2)
		if err2 != nil {
			return false
		}
		resp2.Body.Close()
		return resp2.StatusCode < 500
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// StatusPayload is returned by GET /api/hf/status.
type StatusPayload struct {
	TokenConfigured bool   `json:"token_configured"`
	RouterReachable bool   `json:"router_reachable"`
	CacheDir        string `json:"cache_dir"`
}

// BuildStatus builds hub status for the HF integration.
func BuildStatus(cfg *config.Config, mgr *Manager) StatusPayload {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	token := TokenFromConfig(cfg)
	out := StatusPayload{
		TokenConfigured: token != "",
		RouterReachable: RouterReachable(ctx),
	}
	if mgr != nil {
		out.CacheDir = mgr.CacheDir()
	}
	return out
}
