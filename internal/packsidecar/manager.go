package packsidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/packs"
)

const (
	defaultPython     = "python3"
	healthPath        = "/health"
	hubServerRelPath  = "assets/hub/server.py"
	startupTimeout    = 15 * time.Second
	defaultHealthWait = 500 * time.Millisecond
)

// Instance is a running pack hub sidecar.
type Instance struct {
	PackID    string
	BaseURL   string
	Port      int
	Kind      string
	MCPAgents []string
	cmd       *exec.Cmd
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

// RestartPack stops a running sidecar for packID (if any) and starts it again with env.
func (m *Manager) RestartPack(ctx context.Context, env packs.SidecarEnv) error {
	if m == nil {
		return nil
	}
	packID := strings.TrimSpace(env.PackID)
	if packID == "" {
		return fmt.Errorf("pack id required")
	}
	m.mu.Lock()
	if inst, ok := m.instances[packID]; ok {
		m.stopLocked(inst)
		delete(m.instances, packID)
	}
	m.mu.Unlock()
	inst, err := m.startLocked(ctx, env)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.instances[packID] = inst
	m.mu.Unlock()
	return nil
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

// InstanceForPack returns the running sidecar instance for packID.
func (m *Manager) InstanceForPack(packID string) *Instance {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	inst := m.instances[packID]
	if inst == nil {
		return nil
	}
	copy := *inst
	return &copy
}

// MCPSidecarActive reports whether packID has a running mcp-sidecar.
func (m *Manager) MCPSidecarActive(packID string) bool {
	inst := m.InstanceForPack(packID)
	return inst != nil && inst.Kind == packs.SidecarKindMCP
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
	var bodyBytes []byte
	if r.Body != nil {
		var readErr error
		bodyBytes, readErr = io.ReadAll(r.Body)
		_ = r.Body.Close()
		if readErr != nil {
			return readErr
		}
		if len(bodyBytes) > 0 {
			body = bytes.NewReader(bodyBytes)
		}
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, body)
	if err != nil {
		return err
	}
	req.Header = r.Header.Clone()
	req.Header.Del("Content-Length")
	req.Header.Del("Transfer-Encoding")
	if bodyBytes != nil {
		req.ContentLength = int64(len(bodyBytes))
	}
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
	if env.Kind == packs.SidecarKindMCP {
		return m.startMCPSidecar(ctx, env)
	}
	return m.startHubSidecar(ctx, env)
}

func (m *Manager) startHubSidecar(ctx context.Context, env packs.SidecarEnv) (*Instance, error) {
	serverPath, err := packs.ResolvePackRelativePath(env.PackDir, hubServerRelPath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(serverPath); err != nil {
		return nil, fmt.Errorf("sidecar entry %s: %w", hubServerRelPath, err)
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
	cmd.Env = sidecarBaseEnv(env, settingsJSON)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := waitForHealth(baseURL); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	return &Instance{
		PackID:  env.PackID,
		BaseURL: baseURL,
		Port:    port,
		Kind:    packs.SidecarKindHub,
		cmd:     cmd,
	}, nil
}

func (m *Manager) startMCPSidecar(ctx context.Context, env packs.SidecarEnv) (*Instance, error) {
	binaryRel := strings.TrimSpace(env.BinaryRel)
	if binaryRel == "" {
		binaryRel = "assets/mcp/bin/sd-mcp-server"
	}
	binaryPath, err := resolveMCPSidecarBinary(env.PackDir, binaryRel)
	if err != nil {
		return nil, err
	}
	healthPort, err := freePort()
	if err != nil {
		return nil, err
	}
	agentsJSON, _ := json.Marshal(env.MCPAgents)
	settingsJSON, _ := json.Marshal(env.Settings)
	cmd := exec.CommandContext(ctx, binaryPath, fmt.Sprintf("--health-port=%d", healthPort))
	cmd.Dir = env.PackDir
	cmd.Env = append(sidecarBaseEnv(env, settingsJSON),
		"NJ_MCP_AGENTS_JSON="+string(agentsJSON),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", healthPort)
	if err := waitForHealth(baseURL); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	return &Instance{
		PackID:    env.PackID,
		BaseURL:   baseURL,
		Port:      healthPort,
		Kind:      packs.SidecarKindMCP,
		MCPAgents: append([]string(nil), env.MCPAgents...),
		cmd:       cmd,
	}, nil
}

func resolveMCPSidecarBinary(packDir, binaryRel string) (string, error) {
	path, err := packs.ResolvePackRelativePath(packDir, binaryRel)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	// Try platform-specific suffix: sd-mcp-server-darwin-arm64
	base := strings.TrimSuffix(path, "-"+runtime.GOOS+"-"+runtime.GOARCH)
	if base == path {
		base = strings.TrimSuffix(path, filepathBase(path)) + strings.TrimSuffix(filepathBase(path), extName(path))
	}
	alt := base + "-" + runtime.GOOS + "-" + runtime.GOARCH + extName(path)
	if altPath, err := packs.ResolvePackRelativePath(packDir, strings.TrimPrefix(alt, packDir+"/")); err == nil {
		if _, err := os.Stat(altPath); err == nil {
			return altPath, nil
		}
	}
	altRel := strings.TrimSuffix(binaryRel, extName(binaryRel)) + "-" + runtime.GOOS + "-" + runtime.GOARCH + extName(binaryRel)
	if altPath, err := packs.ResolvePackRelativePath(packDir, altRel); err == nil {
		if _, err := os.Stat(altPath); err == nil {
			return altPath, nil
		}
	}
	return path, fmt.Errorf("mcp sidecar binary not found at %s (also tried %s)", binaryRel, altRel)
}

func filepathBase(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func extName(path string) string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	if strings.HasSuffix(path, ".exe") {
		return ".exe"
	}
	return ""
}

func sidecarBaseEnv(env packs.SidecarEnv, settingsJSON []byte) []string {
	return append(os.Environ(),
		"NJ_PACK_ID="+env.PackID,
		"NJ_PACK_DIR="+env.PackDir,
		"NJ_PACK_SETTINGS_JSON="+string(settingsJSON),
	)
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

func waitForHealth(baseURL string) error {
	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		if err := healthCheck(baseURL); err == nil {
			return nil
		}
		time.Sleep(defaultHealthWait)
	}
	return fmt.Errorf("sidecar health check timed out")
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
