package ollama

import (
	"bufio"
	"io"
	"runtime"
	"strings"
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
