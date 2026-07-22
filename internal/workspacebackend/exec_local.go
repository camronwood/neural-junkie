package workspacebackend

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const maxLocalExecOutputBytes = 512 * 1024

func runLocalCommand(ctx context.Context, cwd string, req ExecRequest) (ExecResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, req.Command, req.Args...)
	cmd.Dir = cwd
	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
	}
	// Own process group so timeout/cancel can SIGKILL the whole tree (npm/vite/tauri children).
	configureLocalExecProcessGroup(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ExecResult{ExitCode: 1}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return ExecResult{ExitCode: 1}, err
	}

	if err := cmd.Start(); err != nil {
		return ExecResult{ExitCode: 1}, err
	}
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	var outBuf, errBuf bytes.Buffer
	var mu sync.Mutex
	var wg sync.WaitGroup
	pump := func(r io.Reader, dst *bytes.Buffer) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 256*1024)
		for sc.Scan() {
			mu.Lock()
			if dst.Len() < maxLocalExecOutputBytes {
				if dst.Len() > 0 {
					dst.WriteByte('\n')
				}
				dst.Write(sc.Bytes())
			}
			mu.Unlock()
		}
	}
	wg.Add(2)
	go pump(stdout, &outBuf)
	go pump(stderr, &errBuf)

	// Watch for context cancel and kill the process group — CommandContext alone
	// only signals the shell, leaving npm/vite orphans that can stall Wait forever.
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			killLocalExecProcessGroup(pid)
		case <-done:
		}
	}()

	wg.Wait()
	waitErr := cmd.Wait()
	close(done)

	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled)
	if timedOut {
		killLocalExecProcessGroup(pid)
	}

	res := ExecResult{
		Stdout: truncateLocalExecOutput(outBuf.String()),
		Stderr: truncateLocalExecOutput(errBuf.String()),
	}
	if waitErr == nil {
		return res, nil
	}
	if exit, ok := waitErr.(*exec.ExitError); ok {
		res.ExitCode = exit.ExitCode()
		return res, nil
	}
	if timedOut {
		res.ExitCode = 1
		if strings.TrimSpace(res.Stderr) == "" {
			res.Stderr = "command timed out after " + formatTimeout(req.Timeout, ctx)
		}
		return res, nil
	}
	res.ExitCode = 1
	return res, waitErr
}

func truncateLocalExecOutput(s string) string {
	if len(s) <= maxLocalExecOutputBytes {
		return s
	}
	return s[:maxLocalExecOutputBytes] + "\n...(truncated)"
}

func formatTimeout(reqTimeout time.Duration, ctx context.Context) string {
	if reqTimeout > 0 {
		return reqTimeout.String()
	}
	if dl, ok := ctx.Deadline(); ok {
		return time.Until(dl).Abs().Round(time.Second).String()
	}
	return "deadline"
}
