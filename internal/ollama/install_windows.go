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
	"strconv"
	"strings"
	"time"
)

const ollamaWindowsSetupURL = "https://ollama.com/download/OllamaSetup.exe"

func platformInstallSupported() bool {
	return true
}

func runPlatformOllamaInstall(ctx context.Context, onProgress func(string)) error {
	if err := tryWingetInstall(ctx, onProgress); err == nil {
		if waitErr := waitForOllamaInstalled(ctx, 90*time.Second); waitErr == nil {
			if onProgress != nil {
				onProgress("Ollama installed successfully")
			}
			return nil
		}
		if onProgress != nil {
			onProgress("winget finished but Ollama not detected yet; trying official installer…")
		}
	} else if onProgress != nil {
		onProgress(fmt.Sprintf("winget unavailable (%v); downloading official installer…", err))
	}
	return installOllamaSetupExe(ctx, onProgress)
}

func tryWingetInstall(ctx context.Context, onProgress func(string)) error {
	winget, err := exec.LookPath("winget")
	if err != nil {
		return fmt.Errorf("winget not found")
	}
	if onProgress != nil {
		onProgress("Installing Ollama via winget…")
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
		"--disable-interactivity",
	)
	return runInstallCmd(cmd, onProgress)
}

func installOllamaSetupExe(ctx context.Context, onProgress func(string)) error {
	if onProgress != nil {
		onProgress("Downloading Ollama installer…")
	}

	setupPath := filepath.Join(os.TempDir(), "NeuralJunkie-OllamaSetup.exe")
	if err := downloadOllamaSetup(ctx, setupPath); err != nil {
		return err
	}
	defer os.Remove(setupPath)

	silentArgs := []string{"/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/SP-"}

	if onProgress != nil {
		onProgress("Running Ollama installer…")
	}
	cmd := exec.CommandContext(ctx, setupPath, silentArgs...)
	if err := runInstallCmd(cmd, onProgress); err != nil {
		if onProgress != nil {
			onProgress("Installer needs administrator approval — approve the Windows UAC prompt…")
		}
		if elevErr := runWindowsElevated(ctx, setupPath, silentArgs, onProgress); elevErr != nil {
			return fmt.Errorf("ollama installation failed: %w — approve the UAC prompt, or install from https://ollama.com/download", elevErr)
		}
	}

	if onProgress != nil {
		onProgress("Waiting for Ollama to finish installing…")
	}
	if err := waitForOllamaInstalled(ctx, 120*time.Second); err != nil {
		return fmt.Errorf("%w — try Check Again, or install from https://ollama.com/download", err)
	}

	if onProgress != nil {
		onProgress("Ollama installed successfully")
	}
	return nil
}

func downloadOllamaSetup(ctx context.Context, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaWindowsSetupURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "NeuralJunkie/ollama-install")
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

// runWindowsElevated shows a UAC prompt via PowerShell Start-Process -Verb RunAs.
func runWindowsElevated(ctx context.Context, exe string, args []string, onProgress func(string)) error {
	ps, err := exec.LookPath("powershell")
	if err != nil {
		ps, err = exec.LookPath("powershell.exe")
		if err != nil {
			return fmt.Errorf("powershell not found for UAC elevation")
		}
	}
	argList := make([]string, 0, len(args))
	for _, a := range args {
		argList = append(argList, strconv.Quote(a))
	}
	script := fmt.Sprintf(
		"$p = Start-Process -FilePath %s -ArgumentList @(%s) -Verb RunAs -PassThru -Wait; exit $p.ExitCode",
		strconv.Quote(exe),
		strings.Join(argList, ","),
	)
	cmd := exec.CommandContext(ctx, ps, "-NoProfile", "-NonInteractive", "-Command", script)
	return runInstallCmd(cmd, onProgress)
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
