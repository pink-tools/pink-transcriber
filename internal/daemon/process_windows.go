//go:build windows

package daemon

import "os/exec"

func setProcessGroup(cmd *exec.Cmd) {}

func gracefulKill(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	cmd.Process.Kill()
}
