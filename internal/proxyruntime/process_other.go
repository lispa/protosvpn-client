//go:build !windows

package proxyruntime

import "os/exec"

func configureCommand(cmd *exec.Cmd) {}
