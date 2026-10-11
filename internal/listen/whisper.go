package listen

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultModelName = "ggml-base.en.bin"
	envWhisperCmd    = "NJ_WHISPER_CMD"
	envWhisperModel  = "NJ_WHISPER_MODEL"
)

// WhisperConfig locates the local whisper.cpp binary and model.
type WhisperConfig struct {
	// Command is the whisper CLI path or a full command override (NJ_WHISPER_CMD).
	// When set via NJ_WHISPER_CMD, the binary receives: <cmd> <audioPath> and should
	// print transcript text to stdout (CI stub mode).
	Command string
	// ModelPath is the ggml model file.
	ModelPath string
	// Timeout per transcription call.
	Timeout time.Duration
}

// DefaultWhisperConfig resolves paths under ~/.neural-junkie and env overrides.
func DefaultWhisperConfig() WhisperConfig {
	home, _ := os.UserHomeDir()
	model := filepath.Join(home, ".neural-junkie", "models", "whisper", defaultModelName)
	if v := strings.TrimSpace(os.Getenv(envWhisperModel)); v != "" {
		model = v
	}
	cmd := strings.TrimSpace(os.Getenv(envWhisperCmd))
	if cmd == "" {
		cmd = resolveWhisperBinary(home)
	}
	return WhisperConfig{
		Command:   cmd,
		ModelPath: model,
		Timeout:   2 * time.Minute,
	}
}

func resolveWhisperBinary(home string) string {
	candidates := []string{
		filepath.Join(home, ".neural-junkie", "bin", "whisper-cli"),
		filepath.Join(home, ".neural-junkie", "bin", "whisper"),
		filepath.Join(home, ".neural-junkie", "bin", "main"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	if p, err := exec.LookPath("whisper-cli"); err == nil {
		return p
	}
	if p, err := exec.LookPath("whisper"); err == nil {
		return p
	}
	return filepath.Join(home, ".neural-junkie", "bin", "whisper-cli")
}

// WhisperTranscriber runs whisper.cpp locally. No cloud fallback.
type WhisperTranscriber struct {
	Cfg WhisperConfig
}

// NewWhisperTranscriber builds a transcriber from defaults / env.
func NewWhisperTranscriber() *WhisperTranscriber {
	return &WhisperTranscriber{Cfg: DefaultWhisperConfig()}
}

// Ready reports whether binary + model (or stub cmd) are usable.
func (w *WhisperTranscriber) Ready() (bool, string) {
	if w == nil {
		return false, "transcriber not configured"
	}
	cmd := strings.TrimSpace(w.Cfg.Command)
	if cmd == "" {
		return false, "whisper binary not found; set NJ_WHISPER_CMD or install whisper-cli under ~/.neural-junkie/bin/"
	}
	// Stub / override mode: NJ_WHISPER_CMD alone is enough (no model required).
	if strings.TrimSpace(os.Getenv(envWhisperCmd)) != "" {
		if _, err := exec.LookPath(firstToken(cmd)); err != nil {
			// Allow absolute or relative path that exists
			if st, err2 := os.Stat(firstToken(cmd)); err2 != nil || st.IsDir() {
				return false, fmt.Sprintf("NJ_WHISPER_CMD not executable: %s", cmd)
			}
		}
		return true, ""
	}
	if st, err := os.Stat(cmd); err != nil || st.IsDir() {
		if _, err := exec.LookPath(cmd); err != nil {
			return false, fmt.Sprintf("whisper binary not found at %s; install whisper.cpp and set NJ_WHISPER_CMD or place whisper-cli in ~/.neural-junkie/bin/", cmd)
		}
	}
	model := strings.TrimSpace(w.Cfg.ModelPath)
	if model == "" {
		return false, "whisper model path empty; set NJ_WHISPER_MODEL"
	}
	if st, err := os.Stat(model); err != nil || st.IsDir() {
		return false, fmt.Sprintf("whisper model missing at %s (download ggml-base.en.bin into ~/.neural-junkie/models/whisper/)", model)
	}
	return true, ""
}

func firstToken(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return cmd
	}
	return fields[0]
}

// Transcribe runs the local whisper CLI on audioPath.
func (w *WhisperTranscriber) Transcribe(ctx context.Context, audioPath string) (string, error) {
	if ready, hint := w.Ready(); !ready {
		return "", fmt.Errorf("%w: %s", ErrSTTUnavailable, hint)
	}
	timeout := w.Cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var argv []string
	stub := strings.TrimSpace(os.Getenv(envWhisperCmd)) != ""
	if stub {
		// NJ_WHISPER_CMD may be "python3 /path/stub.py" or a single binary.
		argv = append(strings.Fields(w.Cfg.Command), audioPath)
	} else {
		// whisper.cpp whisper-cli: -m model -f file -nt (no timestamps) -np (no prints)
		argv = []string{w.Cfg.Command, "-m", w.Cfg.ModelPath, "-f", audioPath, "-nt", "-np"}
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("whisper failed: %s", msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}
