package daemon

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pink-tools/pink-otel"
)

func Run(ctx context.Context, whisperDir, port string) error {
	whisperPath := filepath.Join(whisperDir, whisperBinary())
	modelPath := filepath.Join(whisperDir, "ggml-large-v3.bin")
	serverAddr := "127.0.0.1:" + port

	cmd := exec.Command(whisperPath, "-m", modelPath, "-p", port)
	cmd.Stdout = io.Discard
	setProcessGroup(cmd)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pink-whisper: %w", err)
	}

	otel.Info(ctx, "whisper started", otel.Attr{"pid", cmd.Process.Pid}, otel.Attr{"port", port})

	ready := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "listening on port") {
				close(ready)
				io.Copy(io.Discard, stderr)
				return
			}
		}
	}()

	<-ready

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("whisper not accepting connections: %w", err)
	}
	conn.Close()

	otel.Info(ctx, "whisper ready")

	procDone := make(chan error, 1)
	go func() {
		procDone <- cmd.Wait()
	}()

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
