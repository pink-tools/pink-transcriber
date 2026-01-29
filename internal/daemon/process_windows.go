//go:build windows

package daemon

import (
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

func setProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't use process groups the same way
}

func gracefulKill(cmd *exec.Cmd) {
	// Windows doesn't support SIGTERM, just kill
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
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == 259 // STILL_ACTIVE
}
