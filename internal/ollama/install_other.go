//go:build !windows && !darwin && !linux

package ollama

import (
	"context"
	"fmt"
	"runtime"
)

func platformInstallSupported() bool {
	return false
}

func runPlatformOllamaInstall(ctx context.Context, onProgress func(string)) error {
	return fmt.Errorf("automatic Ollama installation not supported on %s", runtime.GOOS)
}
