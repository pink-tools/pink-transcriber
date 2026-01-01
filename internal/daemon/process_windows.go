//go:build windows

package daemon

import (
	"os"
	"os/exec"
)

func setProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't use process groups the same way
}

func killProcess(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	cmd.Process.Kill()
}

func killProcessByPid(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Windows, FindProcess always succeeds
	// Try to get process handle to check if alive
	err = proc.Signal(os.Kill)
	if err != nil {
		return false
	}
	return true
}
