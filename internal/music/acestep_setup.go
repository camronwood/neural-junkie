package music

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
	defaultMusicRoot       = "~/.neural-junkie/music"
	defaultACEStepVenv     = "~/.neural-junkie/music/acestep-venv"
	defaultACEStepProject  = "~/.neural-junkie/music/ACE-Step-1.5"
	defaultACEStepCheckpoint = "~/.neural-junkie/music/checkpoints/acestep-v15-sft"
)

var (
	installMu     sync.Mutex
	installing    atomic.Bool
	lastInstallErr atomic.Value // string
)

// ACEStepPaths holds resolved filesystem locations for ACE-Step.
type ACEStepPaths struct {
	MusicRoot   string `json:"music_root"`
	Venv        string `json:"venv"`
	Project     string `json:"project"`
	Checkpoint  string `json:"checkpoint"`
	SetupScript string `json:"setup_script,omitempty"`
}

// ACEStepStatus reports whether ACE-Step is ready for music generation.
type ACEStepStatus struct {
	Ready           bool   `json:"ready"`
	DemoMode        bool   `json:"demo_mode"`
	Installing      bool   `json:"installing"`
	PythonOK        bool   `json:"python_ok"`
	VenvReady       bool   `json:"venv_ready"`
	ProjectReady    bool   `json:"project_ready"`
	CheckpointReady bool   `json:"checkpoint_ready"`
	PythonVersion   string `json:"python_version,omitempty"`
	LastError       string `json:"last_error,omitempty"`
	Paths           ACEStepPaths `json:"paths"`
}

// DefaultACEStepPaths returns default install locations with home expanded.
func DefaultACEStepPaths() ACEStepPaths {
	return ACEStepPaths{
		MusicRoot:  ExpandHomePath(defaultMusicRoot),
		Venv:       ExpandHomePath(defaultACEStepVenv),
		Project:    ExpandHomePath(defaultACEStepProject),
		Checkpoint: ExpandHomePath(defaultACEStepCheckpoint),
	}
}

// ACEStepPathsFromSettings resolves paths from pack sidecar settings overlay.
func ACEStepPathsFromSettings(settings map[string]string, packDir string) ACEStepPaths {
	def := DefaultACEStepPaths()
	if settings == nil {
		if packDir != "" {
			def.SetupScript = filepath.Join(packDir, "scripts", "setup-acestep.sh")
		}
		return def
	}
	p := ACEStepPaths{
		MusicRoot:  firstPath(settings["music_root"], settings["ace_step_root"], def.MusicRoot),
		Venv:       firstPath(settings["ace_step_venv"], def.Venv),
		Project:    firstPath(settings["ace_step_project"], def.Project),
		Checkpoint: firstPath(settings["ace_step_checkpoint"], def.Checkpoint),
	}
	if packDir != "" {
		p.SetupScript = filepath.Join(packDir, "scripts", "setup-acestep.sh")
	}
	return p
}

func firstPath(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return ExpandHomePath(v)
		}
	}
	return ""
}

// ExpandHomePath expands leading ~ and $HOME in a path string.
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

// ACEStepStatusFromSettings inspects the local ACE-Step install.
func ACEStepStatusFromSettings(settings map[string]string, packDir string) ACEStepStatus {
	paths := ACEStepPathsFromSettings(settings, packDir)
	st := ACEStepStatus{
		DemoMode: demoModeEnabled(),
		Installing: installing.Load(),
		Paths:    paths,
	}
	if v := lastInstallErr.Load(); v != nil {
		if s, ok := v.(string); ok && s != "" {
			st.LastError = s
		}
	}
	st.VenvReady = fileExists(filepath.Join(paths.Venv, "bin", "python"))
	st.ProjectReady = dirExists(paths.Project)
	st.CheckpointReady = dirExists(paths.Checkpoint)
	if st.VenvReady {
		st.PythonOK = true
		st.PythonVersion = pythonVersion(filepath.Join(paths.Venv, "bin", "python"))
	} else {
		st.PythonOK = findACEStepPython() != ""
	}
	st.Ready = st.DemoMode || (st.VenvReady && st.ProjectReady && st.CheckpointReady)
	return st
}

// InstallACEStep runs the pack setup script (clone, venv, weights download).
func InstallACEStep(ctx context.Context, packDir string) error {
	if packDir == "" {
		return fmt.Errorf("music pack not installed")
	}
	script := filepath.Join(packDir, "scripts", "setup-acestep.sh")
	if !fileExists(script) {
		return fmt.Errorf("setup script missing: %s", script)
	}

	installMu.Lock()
	defer installMu.Unlock()
	if installing.Load() {
		return fmt.Errorf("ACE-Step install already in progress")
	}
	installing.Store(true)
	lastInstallErr.Store("")
	defer installing.Store(false)

	cmd := exec.CommandContext(ctx, "bash", script)
	cmd.Dir = packDir
	cmd.Env = append(os.Environ(), "NJ_MUSIC_ROOT="+DefaultACEStepPaths().MusicRoot)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		lastInstallErr.Store(msg)
		return fmt.Errorf("ACE-Step setup failed: %s", msg)
	}
	lastInstallErr.Store("")
	return nil
}

func demoModeEnabled() bool {
	v := strings.TrimSpace(os.Getenv("NJ_MUSIC_DEMO"))
	return strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func pythonVersion(python string) string {
	out, err := exec.Command(python, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func findACEStepPython() string {
	candidates := []string{"python3.12", "python3.11"}
	if root := os.Getenv("PYENV_ROOT"); root != "" {
		// Prefer newest 3.12/3.11 from pyenv when available.
		for _, ver := range []string{"3.12.5", "3.12.0", "3.11.0"} {
			candidates = append([]string{filepath.Join(root, "versions", ver, "bin", "python")}, candidates...)
		}
	}
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil && aceStepPythonOK(path) {
			return path
		}
		if fileExists(c) && aceStepPythonOK(c) {
			return c
		}
	}
	return ""
}

func aceStepPythonOK(python string) bool {
	err := exec.Command(python, "-c", "import sys; raise SystemExit(0 if (3,11)<=sys.version_info[:2]<(3,13) else 1)").Run()
	return err == nil
}
