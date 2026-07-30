//go:build darwin || linux

package ollama

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const ollamaInstallScriptURL = "https://ollama.com/install.sh"

func platformInstallSupported() bool {
	if _, err := exec.LookPath("bash"); err != nil {
		return false
	}
	return true
}

func runPlatformOllamaInstall(ctx context.Context, onProgress func(string)) error {
	return runOllamaInstallScript(ctx, onProgress)
}

func runOllamaInstallScript(ctx context.Context, onProgress func(string)) error {
	if onProgress != nil {
		onProgress("Downloading Ollama install script...")
	}
	scriptPath, err := downloadOllamaInstallScript(ctx)
	if err != nil {
		return err
	}
	defer os.Remove(scriptPath)

	cmd, method, err := elevatedInstallCmd(ctx, scriptPath)
	if err != nil {
		return err
	}
	if onProgress != nil {
		switch method {
		case "pkexec":
			onProgress("Approve the system password dialog to install Ollama…")
		case "osascript":
			onProgress("Approve the macOS administrator dialog to install Ollama…")
		default:
			onProgress("Installing Ollama…")
		}
	}

	if err := runStreamingCmd(cmd, onProgress); err != nil {
		return fmt.Errorf("%w — %s", err, installFailureHint(method))
	}

	if onProgress != nil {
		onProgress("Waiting for Ollama to appear on PATH…")
	}
	if err := waitForOllamaInstalled(ctx, 90*time.Second); err != nil {
		return fmt.Errorf("%w — %s", err, installFailureHint(method))
	}
	if onProgress != nil {
		onProgress("Ollama installed successfully")
	}
	return nil
}

func installFailureHint(method string) string {
	switch runtime.GOOS {
	case "linux":
		if method == "pkexec" {
			return "approve the password dialog, or install from a terminal: curl -fsSL https://ollama.com/install.sh | sh"
		}
		return "install policykit-1 (pkexec) for a GUI password dialog, or run: curl -fsSL https://ollama.com/install.sh | sh"
	case "darwin":
		return "approve the administrator dialog, or install from https://ollama.com/download"
	default:
		return "install from https://ollama.com/download"
	}
}

func downloadOllamaInstallScript(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaInstallScriptURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "NeuralJunkie/ollama-install")

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download install script: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download install script: HTTP %d", resp.StatusCode)
	}

	f, err := os.CreateTemp("", "nj-ollama-install-*.sh")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(path)
		return "", fmt.Errorf("write install script: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	if err := os.Chmod(path, 0o755); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

// elevatedInstallCmd picks a GUI-capable elevation path when possible.
// method is "pkexec", "osascript", or "bash".
func elevatedInstallCmd(ctx context.Context, scriptPath string) (*exec.Cmd, string, error) {
	abs, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, "", err
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		return nil, "", fmt.Errorf("bash not found: %w", err)
	}

	switch runtime.GOOS {
	case "linux":
		if linuxElevator() == "pkexec" {
			pkexec, _ := exec.LookPath("pkexec")
			cmd := exec.CommandContext(ctx, pkexec, bash, abs)
			return cmd, "pkexec", nil
		}
		cmd := exec.CommandContext(ctx, bash, abs)
		return cmd, "bash", nil
	case "darwin":
		if _, err := exec.LookPath("osascript"); err == nil {
			return darwinAdminBashCmd(ctx, bash, abs), "osascript", nil
		}
		cmd := exec.CommandContext(ctx, bash, abs)
		return cmd, "bash", nil
	default:
		cmd := exec.CommandContext(ctx, bash, abs)
		return cmd, "bash", nil
	}
}

func linuxElevator() string {
	if _, err := exec.LookPath("pkexec"); err == nil {
		return "pkexec"
	}
	return "bash"
}

func darwinAdminBashCmd(ctx context.Context, bash, scriptPath string) *exec.Cmd {
	esc := strings.ReplaceAll(scriptPath, `\`, `\\`)
	esc = strings.ReplaceAll(esc, `"`, `\"`)
	expr := fmt.Sprintf(`do shell script "%s \"%s\"" with administrator privileges`, bash, esc)
	return exec.CommandContext(ctx, "osascript", "-e", expr)
}

func runStreamingCmd(cmd *exec.Cmd, onProgress func(string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start install: %w", err)
	}
	done := make(chan struct{}, 2)
	go func() {
		streamCmdOutput(stdout, onProgress)
		done <- struct{}{}
	}()
	go func() {
		streamCmdOutput(stderr, onProgress)
		done <- struct{}{}
	}()
	<-done
	<-done
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ollama installation failed: %w", err)
	}
	return nil
}
