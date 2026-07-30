package ollama

import (
	"bufio"
	"context"
	"io"
	"runtime"
	"strings"
	"time"
)

// AutoInstallSupported reports whether InstallOllama can run on this OS without bundled Ollama.
func AutoInstallSupported() bool {
	if BundledBinaryPath() != "" {
		return false
	}
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		return platformInstallSupported()
	default:
		return false
	}
}

// SSESafeLine collapses whitespace so install log lines are safe for SSE payloads.
func SSESafeLine(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

func streamCmdOutput(r io.Reader, onProgress func(string)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := SSESafeLine(scanner.Text())
		if line == "" {
			continue
		}
		if onProgress != nil {
			onProgress(line)
		}
	}
}

// waitForOllamaInstalled polls until DetectInstallation reports installed or timeout.
func waitForOllamaInstalled(ctx context.Context, timeout time.Duration) error {
	m := NewManager("")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.DetectInstallation().Installed {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return errOllamaNotDetectedYet
}

// errOllamaNotDetectedYet is wrapped by platform installers with a user-facing hint.
var errOllamaNotDetectedYet = errString("ollama install finished but binary was not detected yet")

type errString string

func (e errString) Error() string { return string(e) }

