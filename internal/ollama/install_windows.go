//go:build windows

package ollama

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const ollamaWindowsSetupURL = "https://ollama.com/download/OllamaSetup.exe"

func platformInstallSupported() bool {
	return true
}

func runPlatformOllamaInstall(ctx context.Context, onProgress func(string)) error {
	if wingetErr := tryWingetInstall(ctx, onProgress); wingetErr == nil {
		return nil
	} else if onProgress != nil {
		onProgress(fmt.Sprintf("winget unavailable (%v); downloading installer...", wingetErr))
	}
	return installOllamaSetupExe(ctx, onProgress)
}

func tryWingetInstall(ctx context.Context, onProgress func(string)) error {
	winget, err := exec.LookPath("winget")
	if err != nil {
		return fmt.Errorf("winget not found")
	}
	if onProgress != nil {
		onProgress("Installing Ollama via winget...")
	}
	cmd := exec.CommandContext(
		ctx,
		winget,
		"install",
		"--id", "Ollama.Ollama",
		"-e",
		"--accept-package-agreements",
		"--accept-source-agreements",
		"--silent",
	)
	return runInstallCmd(cmd, onProgress)
}

func installOllamaSetupExe(ctx context.Context, onProgress func(string)) error {
	if onProgress != nil {
		onProgress("Downloading Ollama installer...")
	}

	setupPath := filepath.Join(os.TempDir(), "NeuralJunkie-OllamaSetup.exe")
	if err := downloadOllamaSetup(ctx, setupPath, onProgress); err != nil {
		return err
	}
	defer os.Remove(setupPath)

	if onProgress != nil {
		onProgress("Running Ollama installer (silent)...")
	}
	cmd := exec.CommandContext(
		ctx,
		setupPath,
		"/VERYSILENT",
		"/SUPPRESSMSGBOXES",
		"/NORESTART",
		"/SP-",
	)
	if err := runInstallCmd(cmd, onProgress); err != nil {
		return err
	}

	if onProgress != nil {
		onProgress("Waiting for Ollama to finish installing...")
	}
	if err := waitForWindowsOllama(ctx, 90*time.Second); err != nil {
		return err
	}

	if onProgress != nil {
		onProgress("Ollama installed successfully")
	}
	return nil
}

func downloadOllamaSetup(ctx context.Context, dest string, onProgress func(string)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaWindowsSetupURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to save installer: %w", err)
	}
	return nil
}

func runInstallCmd(cmd *exec.Cmd, onProgress func(string)) error {
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

func waitForWindowsOllama(ctx context.Context, timeout time.Duration) error {
	m := NewManager("")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status := m.DetectInstallation()
		if status.Installed {
			checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			if m.IsServerRunning(checkCtx) {
				cancel()
				return nil
			}
			cancel()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("ollama install finished but binary was not detected (try Check Again)")
}
