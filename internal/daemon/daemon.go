package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/pink-tools/pink-otel"
)

const port = 7465

func Start(baseDir, whisperDir string) error {
	pidFile := filepath.Join(baseDir, "pink-transcriber.pid")

	// Kill existing process if running
	if running, pid := Status(baseDir); running {
		otel.Info(context.Background(), "killing existing", map[string]any{"pid": pid})
		killProcessByPid(pid)
		os.Remove(pidFile)
	}

	// Start pink-whisper from its own directory
	whisperPath := filepath.Join(whisperDir, whisperBinary())
	modelPath := filepath.Join(whisperDir, "ggml-large-v3.bin")

	cmd := exec.Command(whisperPath, "-m", modelPath, "-p", strconv.Itoa(port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout // whisper.cpp writes everything to stderr

	// Set process group for cleanup
	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pink-whisper: %w", err)
	}

	// Write PID
	os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	otel.Info(context.Background(), "started", map[string]any{"pid": cmd.Process.Pid, "port": port})

	// Wait for server to be ready
	waitForServer()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for process exit in goroutine
	procDone := make(chan error, 1)
	go func() {
		procDone <- cmd.Wait()
	}()

	// Wait for either signal or process exit
	select {
	case <-sigChan:
		otel.Info(context.Background(), "shutting down")
		gracefulKill(cmd)
		os.Remove(pidFile)
		return nil
	case err := <-procDone:
		os.Remove(pidFile)
		return err
	}
}

func Stop(baseDir string) error {
	pidFile := filepath.Join(baseDir, "pink-transcriber.pid")

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("not running (no pid file)")
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("invalid pid file")
	}

	if err := killProcessByPid(pid); err != nil {
		os.Remove(pidFile)
		return err
	}

	os.Remove(pidFile)
	return nil
}

func Status(baseDir string) (bool, int) {
	pidFile := filepath.Join(baseDir, "pink-transcriber.pid")

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false, 0
	}

	if !isProcessAlive(pid) {
		os.Remove(pidFile)
		return false, 0
	}

	return true, pid
}

func whisperBinary() string {
	if runtime.GOOS == "windows" {
		return "pink-whisper.exe"
	}
	return "pink-whisper"
}

func waitForServer() {
	for i := 0; i < 100; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
