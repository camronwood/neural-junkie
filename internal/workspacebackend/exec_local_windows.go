//go:build windows

package workspacebackend

import "os/exec"

func configureLocalExecProcessGroup(cmd *exec.Cmd) {}

func killLocalExecProcessGroup(pid int) {}
