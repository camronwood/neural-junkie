package shared

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

const maxShellOutputBytes = 512 * 1024

// ShellRunResult is the outcome of a sandboxed sh -c command.
type ShellRunResult struct {
	Output   string
	ExitCode int
	TimedOut bool
}

// RunShellCommand runs sh -c cmdStr in dir with timeout, streaming, and process-group cleanup.
func RunShellCommand(ctx context.Context, dir, cmdStr string) (ShellRunResult, error) {
	timeout := RunCommandTimeoutFromContext(ctx)
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "sh", "-c", cmdStr)
	if dir != "" {
		cmd.Dir = dir
	}
	configureCommandProcessGroup(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ShellRunResult{ExitCode: 1}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return ShellRunResult{ExitCode: 1}, err
	}

	if err := cmd.Start(); err != nil {
		return ShellRunResult{ExitCode: 1}, err
	}

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	var buf strings.Builder
	progress := RunCommandProgressFromContext(ctx)
	var mu sync.Mutex
	appendLine := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
		if progress != nil {
			progress(line)
		}
	}

	var wg sync.WaitGroup
	pump := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 256*1024)
		for sc.Scan() {
			line := sc.Text()
			appendLine(line)
			if buf.Len() >= maxShellOutputBytes {
				return
			}
		}
	}
	wg.Add(2)
	go pump(stdout)
	go pump(stderr)
	wg.Wait()

	waitErr := cmd.Wait()
	timedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)
	if timedOut && pid > 0 {
		killCommandProcessGroup(pid)
	}

	exitCode := 0
	if waitErr != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	output := buf.String()
	if len(output) > maxShellOutputBytes {
		output = output[:maxShellOutputBytes] + "\n...(truncated)"
	}
	if timedOut && IsDevServerCommand(cmdStr) {
		output = annotateDevServerTimeout(output)
	}

	return ShellRunResult{
		Output:   output,
		ExitCode: exitCode,
		TimedOut: timedOut,
	}, waitErr
}

func annotateDevServerTimeout(output string) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(output, "\n"))
	b.WriteString("\nbootfix_hint=dev_server_timeout\n")
	b.WriteString("Dev server command timed out — use npm run build or tsc --noEmit for compile diagnostics instead of long-running dev servers.\n")
	return b.String()
}

// FormatShellRunSummary formats output for MCP run_command tools.
func FormatShellRunSummary(res ShellRunResult) string {
	return fmt.Sprintf("exit_code=%d\n%s", res.ExitCode, res.Output)
}
