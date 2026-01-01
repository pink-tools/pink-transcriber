package daemon

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/pink-tools/pink-otel"
)

const port = 7465

func Start(baseDir, whisperDir string) error {
	pidFile := filepath.Join(baseDir, "pink-transcriber.pid")

	// Check if already running
	if running, _ := Status(baseDir); running {
		return fmt.Errorf("already running")
	}

	// Start pink-whisper from its own directory
	whisperPath := filepath.Join(whisperDir, whisperBinary())
	modelPath := filepath.Join(whisperDir, "ggml-large-v3.bin")

	cmd := exec.Command(whisperPath, "-m", modelPath, "-p", strconv.Itoa(port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set process group for cleanup
	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pink-whisper: %w", err)
	}

	// Write PID
	os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	otel.Info("started", map[string]any{"pid": cmd.Process.Pid, "port": port})

	// Wait for server to be ready
	waitForServer()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		otel.Info("shutting down")
		killProcess(cmd)
		os.Remove(pidFile)
		os.Exit(0)
	}()

	// Wait for process
	err := cmd.Wait()
	os.Remove(pidFile)
	return err
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
