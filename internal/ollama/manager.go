package ollama

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type InstallStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Path      string `json:"path,omitempty"`
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
	paths := []string{}

	if p, err := exec.LookPath("ollama"); err == nil {
		paths = append(paths, p)
	}

	candidates := []string{
		"/usr/local/bin/ollama",
		"/opt/homebrew/bin/ollama",
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".ollama", "ollama"))
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
		return InstallStatus{Installed: false}
	}

	binPath := paths[0]
	version := ""
	if out, err := exec.Command(binPath, "--version").CombinedOutput(); err == nil {
		version = strings.TrimSpace(string(out))
	}

	return InstallStatus{
		Installed: true,
		Version:   version,
		Path:      binPath,
	}
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
			return m.serverCmd.Process.Kill()
		}
		m.serverCmd.Wait()
		m.serverCmd = nil
	}
	return nil
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

// PullModel pulls a model and streams progress to the provided callback.
// The callback is called for each progress line from Ollama's streaming API.
func (m *Manager) PullModel(ctx context.Context, model string, onProgress func(PullProgress)) error {
	body := fmt.Sprintf(`{"name":"%s","stream":true}`, model)
	req, err := http.NewRequestWithContext(ctx, "POST", m.endpoint+"/api/pull",
		strings.NewReader(body))
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

// InstallOllama downloads and installs Ollama. On macOS it downloads the
// CLI binary; on Linux it uses the official install script.
func (m *Manager) InstallOllama(ctx context.Context, onProgress func(string)) error {
	switch runtime.GOOS {
	case "darwin":
		return m.installOllamaDarwin(ctx, onProgress)
	case "linux":
		return m.installOllamaLinux(ctx, onProgress)
	default:
		return fmt.Errorf("automatic Ollama installation not supported on %s", runtime.GOOS)
	}
}

func (m *Manager) installOllamaDarwin(ctx context.Context, onProgress func(string)) error {
	if onProgress != nil {
		onProgress("Downloading Ollama for macOS...")
	}

	// Use the official install script
	cmd := exec.CommandContext(ctx, "bash", "-c", "curl -fsSL https://ollama.com/install.sh | sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ollama installation failed: %w", err)
	}

	if onProgress != nil {
		onProgress("Ollama installed successfully")
	}
	return nil
}

func (m *Manager) installOllamaLinux(ctx context.Context, onProgress func(string)) error {
	if onProgress != nil {
		onProgress("Downloading Ollama for Linux...")
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", "curl -fsSL https://ollama.com/install.sh | sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ollama installation failed: %w", err)
	}

	if onProgress != nil {
		onProgress("Ollama installed successfully")
	}
	return nil
}
