package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pink-tools/pink-otel"
	"github.com/pink-tools/pink-transcriber/internal/bootstrap"
	"github.com/pink-tools/pink-transcriber/internal/daemon"
	"github.com/pink-tools/pink-transcriber/internal/transcriber"
)

const version = "1.0.0"

func main() {
	otel.Init("pink-transcriber", version)

	baseDir := getBaseDir()
	whisperDir := bootstrap.GetWhisperDir()

	if len(os.Args) < 2 {
		runDaemon(baseDir, whisperDir)
		return
	}

	switch os.Args[1] {
	case "stop":
		if err := daemon.Stop(baseDir); err != nil {
			otel.Error(context.Background(), "stop failed", map[string]any{"error": err.Error()})
			os.Exit(1)
		}
		otel.Info(context.Background(), "stopped")
	case "status":
		running, pid := daemon.Status(baseDir)
		if running {
			fmt.Printf("running (pid %d)\n", pid)
		} else {
			fmt.Println("not running")
		}
	case "transcribe":
		if len(os.Args) < 3 {
			otel.Error(context.Background(), "missing file argument")
			fmt.Println("Usage: pink-transcriber transcribe <file>")
			os.Exit(1)
		}
		runTranscribe(os.Args[2])
	case "help", "-h":
		printUsage()
	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runDaemon(baseDir, whisperDir string) {
	if err := bootstrap.EnsureReady(); err != nil {
		otel.Error(context.Background(), "bootstrap failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	if err := daemon.Start(baseDir, whisperDir); err != nil {
		otel.Error(context.Background(), "start failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
}

func runTranscribe(filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		otel.Error(context.Background(), "file not found", map[string]any{"path": filePath})
		os.Exit(1)
	}

	if err := transcriber.CheckFFmpeg(); err != nil {
		otel.Error(context.Background(), "ffmpeg not found")
		os.Exit(1)
	}

	if !transcriber.IsServerRunning() {
		otel.Error(context.Background(), "server not running")
		os.Exit(1)
	}

	text, err := transcriber.Transcribe(filePath)
	if err != nil {
		otel.Error(context.Background(), "transcribe failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	fmt.Println(text)
}

func printUsage() {
	fmt.Println(`pink-transcriber - speech to text

Usage:
  pink-transcriber                 Start daemon (bootstrap + server)
  pink-transcriber stop            Stop daemon
  pink-transcriber status          Check if daemon is running
  pink-transcriber transcribe FILE Transcribe audio file
  pink-transcriber help            Show this help

Supported formats: any audio format (converted via ffmpeg)`)
}

func getBaseDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "pink-tools", "pink-transcriber")
	os.MkdirAll(dir, 0755)
	return dir
}
