//go:build windows

package shared

import "os/exec"

func configureCommandProcessGroup(cmd *exec.Cmd) {}

func killCommandProcessGroup(pid int) {}
