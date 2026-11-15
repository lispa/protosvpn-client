//go:build !windows

package proxyengine

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
