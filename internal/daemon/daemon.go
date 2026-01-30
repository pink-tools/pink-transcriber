package daemon

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/pink-tools/pink-otel"
)

const port = 7465

// Run starts pink-whisper server and blocks until ctx is cancelled
func Run(ctx context.Context, whisperDir string) error {
	whisperPath := filepath.Join(whisperDir, whisperBinary())
	modelPath := filepath.Join(whisperDir, "ggml-large-v3.bin")

	cmd := exec.Command(whisperPath, "-m", modelPath, "-p", strconv.Itoa(port))
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard // suppress whisper internal logs

	// Set process group for cleanup
	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pink-whisper: %w", err)
	}

	otel.Info(ctx, "whisper started", otel.Attr{"pid", cmd.Process.Pid}, otel.Attr{"port", port})

	// Wait for server to be ready
	waitForServer()
	otel.Info(ctx, "whisper ready")

	// Wait for process exit in goroutine
	procDone := make(chan error, 1)
	go func() {
		procDone <- cmd.Wait()
	}()

	// Wait for either context cancellation or process exit
	select {
	case <-ctx.Done():
		otel.Info(ctx, "stopping whisper")
		gracefulKill(cmd)
		return nil
	case err := <-procDone:
		if err != nil {
			return fmt.Errorf("whisper exited: %w", err)
		}
		return nil
	}
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
