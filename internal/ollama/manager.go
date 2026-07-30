package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type InstallStatus struct {
	Installed          bool   `json:"installed"`
	Bundled            bool   `json:"bundled,omitempty"`
	Version            string `json:"version,omitempty"`
	Path               string `json:"path,omitempty"`
	RecommendedVersion string `json:"recommended_version,omitempty"`
	MinVersion         string `json:"min_version,omitempty"`
	UpdateAvailable    bool   `json:"update_available,omitempty"`
	MeetsMinimum       bool   `json:"meets_minimum,omitempty"`
	UpdateSupported    bool   `json:"update_supported,omitempty"`
	EffectiveVersion   string `json:"effective_version,omitempty"`
}

// BundledBinaryPath returns the app-shipped Ollama binary when NJ_BUNDLED_OLLAMA is set.
func BundledBinaryPath() string {
	p := strings.TrimSpace(os.Getenv("NJ_BUNDLED_OLLAMA"))
	if p == "" {
		return ""
	}
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

type PullProgress struct {
	Status    string  `json:"status"`
	Digest    string  `json:"digest,omitempty"`
	Total     int64   `json:"total,omitempty"`
	Completed int64   `json:"completed,omitempty"`
	Percent   float64 `json:"percent,omitempty"`
}

type Manager struct {
	endpoint   string
	mu         sync.Mutex
	pullMu     sync.Mutex
	serverCmd  *exec.Cmd
	httpClient *http.Client
}

func NewManager(endpoint string) *Manager {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	return &Manager{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (m *Manager) DetectInstallation() InstallStatus {
	fill := func(st InstallStatus) InstallStatus {
		raw := st.Version
		if parsed, ok := ParseOllamaVersion(raw); ok {
			st.Version = parsed
		}
		st.RecommendedVersion = RecommendedOllamaVersion
		st.MinVersion = MinOllamaVersion
		st.UpdateSupported = UpdateSupported(st)
		effective := st.Version
		// Prefer live server version for feature gates when reachable.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if m.IsServerRunning(ctx) {
			if ver, ok := m.fetchAPIVersion(ctx); ok {
				effective = ver
			}
		}
		st.EffectiveVersion = effective
		st.UpdateAvailable = st.Installed && NeedsUpdate(st.Version)
		st.MeetsMinimum = MeetsMinimum(effective)
		return st
	}

	if bundled, isBundled := resolveBundledBinary(); bundled != "" {
		version := ""
		if out, err := exec.Command(bundled, "--version").Output(); err == nil {
			version = strings.TrimSpace(string(out))
		}
		return fill(InstallStatus{
			Installed: true,
			Bundled:   isBundled,
			Version:   version,
			Path:      bundled,
		})
	}

	paths := []string{}

	if p, err := exec.LookPath("ollama"); err == nil {
		paths = append(paths, p)
	}

	candidates := []string{}
	switch runtime.GOOS {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			candidates = append(candidates, filepath.Join(localAppData, "Programs", "Ollama", "ollama.exe"))
		}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, "AppData", "Local", "Programs", "Ollama", "ollama.exe"))
		}
	default:
		candidates = []string{
			"/usr/local/bin/ollama",
			"/usr/bin/ollama",
			"/opt/homebrew/bin/ollama",
		}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, ".ollama", "ollama"))
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			found := false
			for _, existing := range paths {
				if existing == c {
					found = true
					break
				}
			}
			if !found {
				paths = append(paths, c)
			}
		}
	}

	if len(paths) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if m.IsServerRunning(ctx) {
			st := fill(InstallStatus{Installed: true, Version: "", Path: ""})
			if st.EffectiveVersion != "" {
				st.Version = st.EffectiveVersion
			}
			return st
		}
		return fill(InstallStatus{Installed: false})
	}

	binPath := paths[0]
	version := ""
	if out, err := exec.Command(binPath, "--version").Output(); err == nil {
		version = strings.TrimSpace(string(out))
	}

	return fill(InstallStatus{
		Installed: true,
		Version:   version,
		Path:      binPath,
	})
}

func (m *Manager) fetchAPIVersion(ctx context.Context) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.endpoint+"/api/version", nil)
	if err != nil {
		return "", false
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", false
	}
	return ParseAPIVersion(string(body))
}

func (m *Manager) IsServerRunning(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", m.endpoint+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (m *Manager) StartServer(ctx context.Context) error {
	if m.IsServerRunning(ctx) {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.serverCmd != nil && m.serverCmd.Process != nil {
		if m.IsServerRunning(ctx) {
			return nil
		}
	}

	status := m.DetectInstallation()
	if !status.Installed {
		return fmt.Errorf("ollama not installed")
	}

	cmd := exec.CommandContext(ctx, status.Path, "serve")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if status.Bundled {
		modelsDir := ResolveOllamaModelsDir()
		_ = os.MkdirAll(modelsDir, 0o755)
		cmd.Env = append(os.Environ(),
			"OLLAMA_HOST=127.0.0.1:11434",
			"OLLAMA_MODELS="+modelsDir,
		)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama serve: %w", err)
	}
	m.serverCmd = cmd

	// Wait for server to be ready
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if m.IsServerRunning(ctx) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("ollama server started but health check timed out")
}

func (m *Manager) StopServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.serverCmd != nil && m.serverCmd.Process != nil {
		if err := m.serverCmd.Process.Signal(os.Interrupt); err != nil {
			_ = m.serverCmd.Process.Kill()
		}
		_, _ = m.serverCmd.Process.Wait()
		m.serverCmd = nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if m.IsServerRunning(ctx) {
		return killProcessOnEndpointPort(m.endpoint)
	}
	return nil
}

func killProcessOnEndpointPort(endpoint string) error {
	port, err := portFromHTTPEndpoint(endpoint)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin", "linux":
		out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
		if err != nil {
			return fmt.Errorf("could not find process on port %d: %w", port, err)
		}
		for _, line := range strings.Fields(strings.TrimSpace(string(out))) {
			if err := exec.Command("kill", line).Run(); err != nil {
				_ = exec.Command("kill", "-9", line).Run()
			}
		}
		return nil
	case "windows":
		out, err := exec.Command("cmd", "/c", fmt.Sprintf("for /f \"tokens=5\" %%a in ('netstat -ano ^| findstr :%d') do taskkill /PID %%a /F", port)).Output()
		if err != nil {
			return fmt.Errorf("could not stop process on port %d: %w (%s)", port, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("stop unmanaged ollama not supported on %s", runtime.GOOS)
	}
}

func portFromHTTPEndpoint(endpoint string) (int, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return 0, err
	}
	p := u.Port()
	if p == "" {
		if u.Scheme == "https" {
			return 443, nil
		}
		return 80, nil
	}
	return strconv.Atoi(p)
}

func (m *Manager) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	names := make([]string, len(result.Models))
	for i, m := range result.Models {
		names[i] = m.Name
	}
	return names, nil
}

// RunningModel describes a model currently loaded in Ollama memory.
type RunningModel struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	VRAMBytes int64  `json:"vram_bytes"`
}

// RunningModels returns models currently resident in Ollama (GET /api/ps).
func (m *Manager) RunningModels(ctx context.Context) ([]RunningModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.endpoint+"/api/ps", nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama /api/ps status %d: %s", resp.StatusCode, string(data))
	}

	var result struct {
		Models []struct {
			Name     string `json:"name"`
			Model    string `json:"model"`
			Size     int64  `json:"size"`
			SizeVRAM int64  `json:"size_vram"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	out := make([]RunningModel, 0, len(result.Models))
	for _, row := range result.Models {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			name = strings.TrimSpace(row.Model)
		}
		if name == "" {
			continue
		}
		out = append(out, RunningModel{
			Name:      name,
			SizeBytes: row.Size,
			VRAMBytes: row.SizeVRAM,
		})
	}
	return out, nil
}

// HasModel reports whether an Ollama tag is installed locally.
func (m *Manager) HasModel(ctx context.Context, tag string) (bool, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false, fmt.Errorf("model tag is required")
	}
	names, err := m.ListModels(ctx)
	if err != nil {
		return false, err
	}
	want := tag
	wantWithLatest := tag
	if !strings.Contains(tag, ":") {
		wantWithLatest = tag + ":latest"
	}
	for _, name := range names {
		if name == want || name == wantWithLatest {
			return true, nil
		}
		// Match unversioned request against tagged names (qwen2.5-coder:14b vs qwen2.5-coder).
		if strings.HasPrefix(name, want+":") {
			return true, nil
		}
	}
	return false, nil
}

// TagInstalled reports whether tag is satisfied by any name returned from ListModels.
func TagInstalled(installed []string, tag string) bool {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false
	}
	want := tag
	wantWithLatest := tag
	if !strings.Contains(tag, ":") {
		wantWithLatest = tag + ":latest"
	}
	for _, name := range installed {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if name == want || name == wantWithLatest {
			return true
		}
		if strings.HasPrefix(name, want+":") {
			return true
		}
		if strings.HasSuffix(want, ":latest") && name == strings.TrimSuffix(want, ":latest") {
			return true
		}
	}
	return false
}

// PullModel pulls a model and streams progress to the provided callback.
// The callback is called for each progress line from Ollama's streaming API.
// Concurrent pulls are serialized so SSE clients do not interleave the same daemon state.
func (m *Manager) PullModel(ctx context.Context, model string, onProgress func(PullProgress)) error {
	m.pullMu.Lock()
	defer m.pullMu.Unlock()

	bodyBytes, err := json.Marshal(map[string]any{"name": model, "stream": true})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", m.endpoint+"/api/pull",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{} // no timeout for pulls
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start pull: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed with status %d: %s", resp.StatusCode, string(data))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var progress PullProgress
		if err := json.Unmarshal(scanner.Bytes(), &progress); err != nil {
			continue
		}
		if progress.Total > 0 {
			progress.Percent = float64(progress.Completed) / float64(progress.Total) * 100
		}
		if onProgress != nil {
			onProgress(progress)
		}
	}
	return scanner.Err()
}

// DeleteModel removes a model from the local Ollama store (DELETE /api/delete).
// Ollama 0.17+ rejects POST on this route (405); older docs used {"name"}; current API uses {"model"}.
func (m *Manager) DeleteModel(ctx context.Context, model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model name is required")
	}
	payload, err := json.Marshal(map[string]string{"model": model, "name": model})
	if err != nil {
		return err
	}

	doDelete := func(method string) (int, []byte, error) {
		req, err := http.NewRequestWithContext(ctx, method, m.endpoint+"/api/delete", bytes.NewReader(payload))
		if err != nil {
			return 0, nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil, fmt.Errorf("delete request failed: %w", err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, data, nil
	}

	status, data, err := doDelete(http.MethodDelete)
	if err != nil {
		return err
	}
	// Some older Ollama builds accepted POST instead of DELETE.
	if status == http.StatusMethodNotAllowed {
		status, data, err = doDelete(http.MethodPost)
		if err != nil {
			return err
		}
	}
	if status != http.StatusOK {
		return fmt.Errorf("delete failed with status %d: %s", status, string(data))
	}
	return nil
}

// InstallOllama downloads and installs Ollama using the platform installer.
// macOS: official install.sh via osascript admin dialog (when not bundled).
// Linux: official install.sh via pkexec password dialog.
// Windows: winget or OllamaSetup.exe (UAC elevation when needed).
func (m *Manager) InstallOllama(ctx context.Context, onProgress func(string)) error {
	if BundledBinaryPath() != "" {
		if onProgress != nil {
			onProgress("Ollama is bundled with Neural Junkie")
		}
		return nil
	}
	return runPlatformOllamaInstall(ctx, onProgress)
}
