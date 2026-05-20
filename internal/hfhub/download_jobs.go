package hfhub

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const hfDownloadTimeout = 4 * time.Hour

type downloadJob struct {
	repoID   string
	filename string
	mu       sync.RWMutex
	progress DownloadProgress
	done     bool
	err      error
}

func downloadJobKey(repoID, filename string) string {
	return repoID + "\x00" + filename
}

// FileReady reports whether the GGUF is fully on disk (not a partial).
func (m *Manager) FileReady(repoID, filename string) (bool, error) {
	entry, err := FindCatalogEntry(repoID)
	if err != nil {
		return false, err
	}
	filename, err = ResolveDownloadFilename(entry, filename)
	if err != nil {
		return false, err
	}
	st, err := os.Stat(m.filePath(repoID, filename))
	if err != nil {
		return false, nil
	}
	return st.Mode().IsRegular() && st.Size() > 0, nil
}

// DownloadStatus returns the latest progress for an active or recently finished job.
func (m *Manager) DownloadStatus(repoID, filename string) (DownloadProgress, bool) {
	key := downloadJobKey(repoID, filename)
	m.jobsMu.Lock()
	job := m.jobs[key]
	m.jobsMu.Unlock()
	if job == nil {
		return DownloadProgress{}, false
	}
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.progress, true
}

// ActiveDownloads lists in-progress hub-side downloads.
func (m *Manager) ActiveDownloads() []DownloadProgress {
	m.jobsMu.Lock()
	defer m.jobsMu.Unlock()
	var out []DownloadProgress
	for _, job := range m.jobs {
		job.mu.RLock()
		if !job.done {
			out = append(out, job.progress)
		}
		job.mu.RUnlock()
	}
	return out
}

// EnsureDownloadStarted kicks off a background download if the file is not ready.
func (m *Manager) EnsureDownloadStarted(token, repoID, filename string) error {
	ready, err := m.FileReady(repoID, filename)
	if err != nil {
		return err
	}
	if ready {
		return nil
	}

	entry, err := FindCatalogEntry(repoID)
	if err != nil {
		return err
	}
	if !catalogHasMode(entry, "local") {
		return fmt.Errorf("repo_id %q is not enabled for local download in the catalog", repoID)
	}
	filename, err = ResolveDownloadFilename(entry, filename)
	if err != nil {
		return err
	}

	key := downloadJobKey(repoID, filename)
	m.jobsMu.Lock()
	job := m.jobs[key]
	if job != nil {
		job.mu.RLock()
		running := !job.done
		job.mu.RUnlock()
		if running {
			m.jobsMu.Unlock()
			return nil
		}
	}
	job = &downloadJob{
		repoID:   repoID,
		filename: filename,
		progress: DownloadProgress{Status: "queued", RepoID: repoID, Filename: filename},
	}
	m.jobs[key] = job
	m.jobsMu.Unlock()

	go m.runDownloadJob(job, token)
	return nil
}

func (m *Manager) runDownloadJob(job *downloadJob, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), hfDownloadTimeout)
	defer cancel()

	err := m.downloadOnce(ctx, job.repoID, job.filename, token, func(p DownloadProgress) {
		job.mu.Lock()
		job.progress = p
		job.mu.Unlock()
	})

	job.mu.Lock()
	job.done = true
	job.err = err
	if err != nil {
		job.progress = DownloadProgress{
			Status:   "error",
			RepoID:   job.repoID,
			Filename: job.filename,
			Error:    err.Error(),
		}
	} else if job.progress.Status != "success" {
		job.progress = DownloadProgress{
			Status:   "success",
			RepoID:   job.repoID,
			Filename: job.filename,
			Percent:  100,
		}
	}
	job.mu.Unlock()

	// Drop finished jobs after a while so retries can start fresh.
	time.AfterFunc(30*time.Minute, func() {
		m.jobsMu.Lock()
		defer m.jobsMu.Unlock()
		if cur, ok := m.jobs[downloadJobKey(job.repoID, job.filename)]; ok && cur == job && cur.done {
			delete(m.jobs, downloadJobKey(job.repoID, job.filename))
		}
	})
}

// WatchDownload streams progress until the download completes or the client context ends.
// The hub keeps downloading after the client disconnects.
func (m *Manager) WatchDownload(ctx context.Context, repoID, filename string, onProgress func(DownloadProgress)) error {
	if ready, err := m.FileReady(repoID, filename); err != nil {
		return err
	} else if ready {
		if onProgress != nil {
			onProgress(DownloadProgress{Status: "success", RepoID: repoID, Filename: filename, Percent: 100})
		}
		return nil
	}

	key := downloadJobKey(repoID, filename)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var last string
	for {
		m.jobsMu.Lock()
		job := m.jobs[key]
		m.jobsMu.Unlock()

		if job != nil {
			job.mu.RLock()
			p := job.progress
			done := job.done
			err := job.err
			job.mu.RUnlock()
			snap := fmt.Sprintf("%s|%0.2f|%d|%d", p.Status, p.Percent, p.Completed, p.Total)
			if onProgress != nil && snap != last {
				last = snap
				onProgress(p)
			}
			if done {
				if err != nil {
					return err
				}
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// downloadOnce performs the HTTP download (caller must not hold downloadMu globally).
func (m *Manager) downloadOnce(ctx context.Context, repoID, filename, token string, onProgress func(DownloadProgress)) error {
	entry, err := FindCatalogEntry(repoID)
	if err != nil {
		return err
	}
	filename, err = ResolveDownloadFilename(entry, filename)
	if err != nil {
		return err
	}

	dest := m.filePath(repoID, filename)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	report := func(p DownloadProgress) {
		if onProgress != nil {
			onProgress(p)
		}
	}
	report(DownloadProgress{Status: "starting", RepoID: repoID, Filename: filename})

	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repoID, filename)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("access denied (%d): accept the model license on Hugging Face and set HF_TOKEN for gated models", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	total := resp.ContentLength
	tmp := dest + ".partial"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	var written int64
	buf := make([]byte, 32*1024)
	for {
		if ctx.Err() != nil {
			out.Close()
			os.Remove(tmp)
			return ctx.Err()
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := out.Write(buf[:n]); wErr != nil {
				out.Close()
				os.Remove(tmp)
				return wErr
			}
			written += int64(n)
			p := DownloadProgress{Status: "downloading", RepoID: repoID, Filename: filename, Completed: written, Total: total}
			if total > 0 {
				p.Percent = float64(written) / float64(total) * 100
			}
			report(p)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			out.Close()
			os.Remove(tmp)
			return readErr
		}
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	report(DownloadProgress{Status: "success", RepoID: repoID, Filename: filename, Completed: written, Total: total, Percent: 100})
	return nil
}
