package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/pathutil"
)

// AuthState describes CLI authentication status.
type AuthState string

const (
	AuthNotApplicable AuthState = "not_applicable"
	AuthUnknown       AuthState = "unknown"
	AuthNeedsAuth     AuthState = "needs_auth"
	AuthAuthed        AuthState = "authed"
)

// AgentStatus is the full status for one CLI type.
type AgentStatus struct {
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	ProviderName string            `json:"provider_name"`
	Featured     bool              `json:"featured"`
	Installed    bool              `json:"installed"`
	Binary       string            `json:"binary,omitempty"`
	BinaryPath   string            `json:"binary_path,omitempty"`
	Version      string            `json:"version,omitempty"`
	AuthState    AuthState         `json:"auth_state"`
	AuthMethod   string            `json:"auth_method,omitempty"`
	LoginCommand string            `json:"login_command,omitempty"`
	InstallHint  string            `json:"install_hint,omitempty"`
	CanInstall   bool              `json:"can_install"`
	MissingPrereqs []string        `json:"missing_prereqs,omitempty"`
	Install      *agent.CLIInstallSpec `json:"install,omitempty"`
	Auth         *agent.CLIAuthSpec  `json:"auth,omitempty"`
}

// Manager handles CLI install, auth probing, and status.
type Manager struct {
	mu sync.Mutex
}

func NewManager() *Manager {
	return &Manager{}
}

// ListStatus returns status for all registered CLI types.
func (m *Manager) ListStatus(getAPIKey func(providerName string) string) []AgentStatus {
	types := agent.ListCLIAgentTypes()
	out := make([]AgentStatus, 0, len(types))
	pathEnv := pathutil.EnhancedPATH()
	for _, t := range types {
		cfg, ok := agent.GetCLIAgentConfig(t)
		if !ok {
			continue
		}
		out = append(out, m.statusFor(cfg, pathEnv, getAPIKey))
	}
	return out
}

// StatusFor returns status for a single CLI type.
func (m *Manager) StatusFor(cliType string, getAPIKey func(providerName string) string) (AgentStatus, error) {
	cfg, ok := agent.GetCLIAgentConfig(cliType)
	if !ok {
		return AgentStatus{}, fmt.Errorf("unknown CLI type %q", cliType)
	}
	return m.statusFor(cfg, pathutil.EnhancedPATH(), getAPIKey), nil
}

func (m *Manager) statusFor(cfg agent.CLIAgentConfig, pathEnv string, getAPIKey func(providerName string) string) AgentStatus {
	st := AgentStatus{
		Type:         cfg.Type,
		Name:         cfg.DefaultName,
		ProviderName: cfg.ProviderName,
		Featured:     agent.IsFeaturedCLI(cfg.Type),
		InstallHint:  cfg.InstallHint,
		CanInstall:   cfg.HasInstallSpec(),
		Install:      cfg.Install,
		Auth:         cfg.Auth,
		AuthState:    AuthNotApplicable,
	}

	if cfg.Auth != nil {
		st.AuthMethod = cfg.Auth.Method
		if len(cfg.Auth.LoginCommand) > 0 {
			st.LoginCommand = strings.Join(cfg.Auth.LoginCommand, " ")
		}
	}

	if cfg.HasInstallSpec() {
		st.MissingPrereqs = missingPrereqs(cfg.Install.Prereqs, pathEnv)
	}

	resolved, ok := agent.ResolveCLIWithPATH(cfg, pathEnv)
	if !ok {
		st.Installed = false
		if cfg.Auth != nil {
			st.AuthState = AuthNeedsAuth
		}
		return st
	}

	st.Installed = true
	st.Binary = filepath.Base(resolved.Command)
	if filepath.IsAbs(resolved.Command) {
		st.BinaryPath = resolved.Command
		st.Version = probeVersion(resolved.Command)
	} else if fullPath, err := pathutil.LookPathIn(resolved.Command, pathEnv); err == nil {
		st.BinaryPath = fullPath
		st.Version = probeVersion(fullPath)
	}

	if cfg.Auth == nil {
		st.AuthState = AuthNotApplicable
		return st
	}

	st.AuthState = m.probeAuth(cfg, pathEnv, getAPIKey)
	return st
}

func missingPrereqs(prereqs []string, pathEnv string) []string {
	var missing []string
	for _, p := range prereqs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, err := pathutil.LookPathIn(p, pathEnv); err != nil {
			missing = append(missing, p)
		}
	}
	return missing
}

func probeVersion(binaryPath string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binaryPath, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	if len(line) > 120 {
		return line[:120] + "..."
	}
	return line
}

func (m *Manager) probeAuth(cfg agent.CLIAgentConfig, pathEnv string, getAPIKey func(providerName string) string) AuthState {
	auth := cfg.Auth
	if auth == nil {
		return AuthNotApplicable
	}

	for _, envKey := range auth.EnvVars {
		if strings.TrimSpace(os.Getenv(envKey)) != "" {
			return AuthAuthed
		}
		if getAPIKey != nil {
			if key := strings.TrimSpace(getAPIKey(cfg.ProviderName)); key != "" {
				return AuthAuthed
			}
		}
	}

	for _, credPath := range auth.CredentialPaths {
		expanded := pathutil.ExpandHome(credPath)
		if fi, err := os.Stat(expanded); err == nil && !fi.IsDir() {
			return AuthAuthed
		}
	}

	if len(auth.ProbeCommand) == 0 {
		return AuthUnknown
	}

	binary := auth.ProbeCommand[0]
	args := auth.ProbeCommand[1:]
	fullPath, err := pathutil.LookPathIn(binary, pathEnv)
	if err != nil {
		return AuthNeedsAuth
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, fullPath, args...).CombinedOutput()
	combined := strings.ToLower(string(out))

	if err == nil {
		if strings.Contains(combined, `"loggedin": false`) ||
			strings.Contains(combined, `"logged_in": false`) ||
			strings.Contains(combined, "not logged in") ||
			strings.Contains(combined, "not authenticated") {
			return AuthNeedsAuth
		}
		return AuthAuthed
	}

	if strings.Contains(combined, `"loggedin": true`) ||
		strings.Contains(combined, `"logged_in": true`) ||
		strings.Contains(combined, "logged in") ||
		strings.Contains(combined, "login successful") {
		return AuthAuthed
	}

	return AuthNeedsAuth
}

// Install runs the configured install command with progress callbacks.
func (m *Manager) Install(ctx context.Context, cliType string, onProgress func(string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, ok := agent.GetCLIAgentConfig(cliType)
	if !ok {
		return fmt.Errorf("unknown CLI type %q", cliType)
	}
	if !cfg.HasInstallSpec() {
		return fmt.Errorf("%s does not support one-click install", cfg.DefaultName)
	}

	pathEnv := pathutil.EnhancedPATH()
	if missing := missingPrereqs(cfg.Install.Prereqs, pathEnv); len(missing) > 0 {
		return fmt.Errorf("missing prerequisites: %s", strings.Join(missing, ", "))
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("Installing %s...", cfg.DefaultName))
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.Install.Command)
	cmd.Env = append(os.Environ(), "PATH="+pathEnv)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail != "" && onProgress != nil {
			onProgress(detail)
		}
		return fmt.Errorf("install failed: %w", err)
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("%s installed successfully", cfg.DefaultName))
	}

	// Verify binary is discoverable after install.
	pathEnv = pathutil.EnhancedPATH()
	if _, ok := agent.ResolveCLIWithPATH(cfg, pathEnv); !ok {
		return fmt.Errorf("%s install finished but binary not found on PATH — restart the app or open a new shell", cfg.DefaultName)
	}

	return nil
}

// AuthLoginInfo returns instructions for interactive CLI login.
func (m *Manager) AuthLoginInfo(cliType string) (map[string]string, error) {
	cfg, ok := agent.GetCLIAgentConfig(cliType)
	if !ok {
		return nil, fmt.Errorf("unknown CLI type %q", cliType)
	}
	if cfg.Auth == nil {
		return nil, fmt.Errorf("%s does not require authentication", cfg.DefaultName)
	}
	if len(cfg.Auth.LoginCommand) == 0 && cfg.Auth.Method != "api_key" {
		return nil, fmt.Errorf("no login command configured for %s", cfg.DefaultName)
	}

	out := map[string]string{
		"mode": cfg.Auth.Method,
	}
	if len(cfg.Auth.LoginCommand) > 0 {
		out["command"] = strings.Join(cfg.Auth.LoginCommand, " ")
	}
	if len(cfg.Auth.EnvVars) > 0 {
		out["env_var"] = cfg.Auth.EnvVars[0]
	}
	return out, nil
}

// ApplyAPIKey stores an API key in the process environment for CLI subprocesses.
func ApplyAPIKey(cfg agent.CLIAgentConfig, apiKey string) error {
	if cfg.Auth == nil || len(cfg.Auth.EnvVars) == 0 {
		return fmt.Errorf("%s does not support API key auth", cfg.DefaultName)
	}
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return fmt.Errorf("api_key is required")
	}
	for _, envKey := range cfg.Auth.EnvVars {
		_ = os.Setenv(envKey, key)
	}
	return nil
}

// ResolveBinaryPath returns the full path to a CLI binary using enhanced PATH.
func ResolveBinaryPath(cfg agent.CLIAgentConfig) (string, bool) {
	pathEnv := pathutil.EnhancedPATH()
	resolved, ok := agent.ResolveCLIWithPATH(cfg, pathEnv)
	if !ok {
		return "", false
	}
	if filepath.IsAbs(resolved.Command) {
		return filepath.Clean(resolved.Command), true
	}
	full, err := pathutil.LookPathIn(resolved.Command, pathEnv)
	if err != nil {
		return resolved.Command, true
	}
	return filepath.Clean(full), true
}
