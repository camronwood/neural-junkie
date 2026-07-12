package arenasidecar

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	defaultArenaVenv = "~/.neural-junkie/arena/venv"
	requirementsFile = "requirements-sidecar.txt"
)

var (
	installMu      sync.Mutex
	installing     atomic.Bool
	lastInstallErr atomic.Value // string
)

// SidecarPaths holds resolved filesystem locations for the Model Arena sidecar venv.
type SidecarPaths struct {
	Venv         string `json:"venv"`
	Python       string `json:"python"`
	Requirements string `json:"requirements,omitempty"`
}

// SidecarStatus reports whether chess dependencies are installed for the pack sidecar.
type SidecarStatus struct {
	ChessAvailable bool         `json:"chess_available"`
	VenvReady      bool         `json:"venv_ready"`
	Installing     bool         `json:"installing"`
	PythonOK       bool         `json:"python_ok"`
	PythonVersion  string       `json:"python_version,omitempty"`
	LastError      string       `json:"last_error,omitempty"`
	Paths          SidecarPaths `json:"paths"`
}

// DefaultSidecarPaths returns default venv locations with home expanded.
func DefaultSidecarPaths() SidecarPaths {
	venv := ExpandHomePath(defaultArenaVenv)
	return SidecarPaths{
		Venv:   venv,
		Python: filepath.Join(venv, "bin", "python"),
	}
}

// SidecarPathsFromSettings resolves paths from pack settings overlay.
func SidecarPathsFromSettings(settings map[string]string, packDir string) SidecarPaths {
	def := DefaultSidecarPaths()
	venv := def.Venv
	if settings != nil {
		if v := strings.TrimSpace(settings["arena_venv"]); v != "" {
			venv = ExpandHomePath(v)
		}
	}
	p := SidecarPaths{
		Venv:   venv,
		Python: filepath.Join(venv, "bin", "python"),
	}
	if packDir != "" {
		p.Requirements = filepath.Join(packDir, requirementsFile)
	}
	return p
}

// ExpandHomePath expands leading ~ in a path string.
func ExpandHomePath(val string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return ""
	}
	if strings.HasPrefix(val, "~") {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return val
		}
		if val == "~" {
			return home
		}
		if strings.HasPrefix(val, "~/") || strings.HasPrefix(val, "~\\") {
			return filepath.Join(home, val[2:])
		}
	}
	return os.ExpandEnv(val)
}

// SidecarStatusFromSettings inspects the local arena sidecar Python environment.
func SidecarStatusFromSettings(settings map[string]string, packDir string) SidecarStatus {
	paths := SidecarPathsFromSettings(settings, packDir)
	st := SidecarStatus{
		Installing: installing.Load(),
		Paths:      paths,
	}
	if v := lastInstallErr.Load(); v != nil {
		if s, ok := v.(string); ok && s != "" {
			st.LastError = s
		}
	}
	st.VenvReady = fileExists(paths.Python)
	if st.VenvReady {
		st.PythonOK = true
		st.PythonVersion = pythonVersion(paths.Python)
		st.ChessAvailable = pythonImports(paths.Python, "chess")
	} else {
		if py := findSystemPython(); py != "" {
			st.PythonOK = true
			st.PythonVersion = pythonVersion(py)
			st.ChessAvailable = pythonImports(py, "chess")
		}
	}
	return st
}

// InstallSidecarDeps creates the arena venv and installs requirements-sidecar.txt.
func InstallSidecarDeps(ctx context.Context, packDir string, settings map[string]string) error {
	if strings.TrimSpace(packDir) == "" {
		return fmt.Errorf("model arena pack not installed")
	}
	reqPath := filepath.Join(packDir, requirementsFile)
	if !fileExists(reqPath) {
		return fmt.Errorf("requirements file missing: %s", reqPath)
	}
	paths := SidecarPathsFromSettings(settings, packDir)
	systemPython := findSystemPython()
	if systemPython == "" {
		return fmt.Errorf("python3 not found on PATH")
	}

	installMu.Lock()
	defer installMu.Unlock()
	if installing.Load() {
		return fmt.Errorf("arena sidecar install already in progress")
	}
	installing.Store(true)
	lastInstallErr.Store("")
	defer installing.Store(false)

	if err := os.MkdirAll(filepath.Dir(paths.Venv), 0o755); err != nil {
		lastInstallErr.Store(err.Error())
		return err
	}
	if !fileExists(paths.Python) {
		if err := runCmd(ctx, systemPython, "-m", "venv", paths.Venv); err != nil {
			msg := fmt.Sprintf("create venv: %v", err)
			lastInstallErr.Store(msg)
			return fmt.Errorf("%s", msg)
		}
	}
	pip := filepath.Join(paths.Venv, "bin", "pip")
	if err := runCmd(ctx, pip, "install", "-r", reqPath); err != nil {
		msg := fmt.Sprintf("pip install: %v", err)
		lastInstallErr.Store(msg)
		return fmt.Errorf("%s", msg)
	}
	if !pythonImports(paths.Python, "chess") {
		msg := "python-chess import failed after install"
		lastInstallErr.Store(msg)
		return fmt.Errorf("%s", msg)
	}
	lastInstallErr.Store("")
	return nil
}

func runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func pythonVersion(python string) string {
	out, err := exec.Command(python, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func pythonImports(python, module string) bool {
	err := exec.Command(python, "-c", "import "+module).Run()
	return err == nil
}

func findSystemPython() string {
	for _, c := range []string{"python3", "python"} {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
	}
	return ""
}
