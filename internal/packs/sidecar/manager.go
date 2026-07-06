package sidecar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/packs"
)

const (
	defaultPython     = "python3"
	healthPath        = "/health"
	serverRelPath     = "assets/hub/server.py"
	startupTimeout    = 15 * time.Second
	defaultHealthWait = 500 * time.Millisecond
)

// Instance is a running pack hub sidecar.
type Instance struct {
	PackID  string
	BaseURL string
	Port    int
	cmd     *exec.Cmd
}

// Manager starts and stops pack sidecars for enabled customer packs.
type Manager struct {
	mu        sync.Mutex
	instances map[string]*Instance
}

// NewManager creates a sidecar manager.
func NewManager() *Manager {
	return &Manager{instances: make(map[string]*Instance)}
}

// Sync ensures sidecars match the given enabled pack manifests.
func (m *Manager) Sync(ctx context.Context, enabled []packs.SidecarEnv) error {
	if m == nil {
		return nil
	}
	want := make(map[string]packs.SidecarEnv)
	for _, e := range enabled {
		if strings.TrimSpace(e.PackID) == "" {
			continue
		}
		want[e.PackID] = e
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, inst := range m.instances {
		if _, ok := want[id]; !ok {
			m.stopLocked(inst)
			delete(m.instances, id)
		}
	}
	var firstErr error
	for id, env := range want {
		if _, ok := m.instances[id]; ok {
			continue
		}
		inst, err := m.startLocked(ctx, env)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("pack %q sidecar: %w", id, err)
			}
			continue
		}
		m.instances[id] = inst
	}
	return firstErr
}

// StopAll stops every running sidecar.
func (m *Manager) StopAll() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, inst := range m.instances {
		m.stopLocked(inst)
		delete(m.instances, id)
	}
}

// BaseURL returns the sidecar base URL for packID, or "" if not running.
func (m *Manager) BaseURL(packID string) string {
	if m == nil {
		return ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[packID]; ok {
		return inst.BaseURL
	}
	return ""
}

// ProxyHTTP forwards a hub request to the pack sidecar.
func (m *Manager) ProxyHTTP(w http.ResponseWriter, r *http.Request, packID, sidecarPath string) error {
	base := m.BaseURL(packID)
	if base == "" {
		return fmt.Errorf("sidecar not running for pack %q", packID)
	}
	target := strings.TrimRight(base, "/") + sidecarPath
	if q := r.URL.RawQuery; q != "" {
		target += "?" + q
	}
	var body io.Reader
	if r.Body != nil {
		body = r.Body
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, body)
	if err != nil {
		return err
	}
	req.Header = r.Header.Clone()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	return err
}

func (m *Manager) startLocked(ctx context.Context, env packs.SidecarEnv) (*Instance, error) {
	serverPath, err := packs.ResolvePackRelativePath(env.PackDir, serverRelPath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(serverPath); err != nil {
		return nil, fmt.Errorf("sidecar entry %s: %w", serverRelPath, err)
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	python := defaultPython
	if p := strings.TrimSpace(env.Settings["python_executable"]); p != "" {
		python = p
	}
	settingsJSON, _ := json.Marshal(env.Settings)
	cmd := exec.CommandContext(ctx, python, serverPath, fmt.Sprintf("--port=%d", port))
	cmd.Dir = env.PackDir
	cmd.Env = append(os.Environ(),
		"NJ_PACK_ID="+env.PackID,
		"NJ_PACK_DIR="+env.PackDir,
		"NJ_PACK_SETTINGS_JSON="+string(settingsJSON),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		if err := healthCheck(baseURL); err == nil {
			return &Instance{
				PackID:  env.PackID,
				BaseURL: baseURL,
				Port:    port,
				cmd:     cmd,
			}, nil
		}
		time.Sleep(defaultHealthWait)
	}
	_ = cmd.Process.Kill()
	return nil, fmt.Errorf("sidecar health check timed out")
}

func (m *Manager) stopLocked(inst *Instance) {
	if inst == nil || inst.cmd == nil || inst.cmd.Process == nil {
		return
	}
	_ = inst.cmd.Process.Kill()
	_, _ = inst.cmd.Process.Wait()
}

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port, nil
}

func healthCheck(baseURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+healthPath, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health status %d", resp.StatusCode)
	}
	return nil
}

