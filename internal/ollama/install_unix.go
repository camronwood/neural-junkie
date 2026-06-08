//go:build darwin || linux

package ollama

import (
	"context"
	"fmt"
	"os/exec"
)

const ollamaInstallScriptURL = "https://ollama.com/install.sh"

func platformInstallSupported() bool {
	return pickInstallRunner() != ""
}

func runPlatformOllamaInstall(ctx context.Context, onProgress func(string)) error {
	return runOllamaInstallScript(ctx, onProgress)
}

func pickInstallRunner() string {
	if _, err := exec.LookPath("curl"); err == nil {
		return "curl"
	}
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}
	if _, err := exec.LookPath("python"); err == nil {
		return "python"
	}
	return ""
}

func runOllamaInstallScript(ctx context.Context, onProgress func(string)) error {
	runner := pickInstallRunner()
	if runner == "" {
		return fmt.Errorf("need curl or python3 to download the Ollama install script")
	}

	var cmd *exec.Cmd
	switch runner {
	case "curl":
		if onProgress != nil {
			onProgress("Downloading Ollama install script...")
		}
		cmd = exec.CommandContext(ctx, "bash", "-c", "curl -fsSL "+ollamaInstallScriptURL+" | sh")
	default:
		if onProgress != nil {
			onProgress("Downloading Ollama install script (python)...")
		}
		pyScript := fmt.Sprintf(`
import sys, os, tempfile, subprocess
try:
    from urllib.request import urlopen
except ImportError:
    from urllib2 import urlopen
data = urlopen(%q).read()
fd, path = tempfile.mkstemp(suffix='.sh')
os.write(fd, data)
os.close(fd)
os.chmod(path, 0o755)
rc = subprocess.call(['bash', path])
os.unlink(path)
sys.exit(rc)
`, ollamaInstallScriptURL)
		cmd = exec.CommandContext(ctx, runner, "-c", pyScript)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if onProgress != nil {
		onProgress("Installing Ollama (may prompt for your password)...")
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

	if onProgress != nil {
		onProgress("Ollama installed successfully")
	}
	return nil
}
