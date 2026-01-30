package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pink-tools/pink-core"
	"github.com/pink-tools/pink-otel"
	"github.com/pink-tools/pink-transcriber/internal/bootstrap"
	"github.com/pink-tools/pink-transcriber/internal/daemon"
	"github.com/pink-tools/pink-transcriber/internal/transcriber"
)

var version = "dev"

const serviceName = "pink-transcriber"

func main() {
	core.LoadEnv(serviceName)
	whisperDir := bootstrap.GetWhisperDir()

	core.Run(core.Config{
		Name:    serviceName,
		Version: version,
		Usage: `pink-transcriber - speech to text

Usage:
  pink-transcriber                 Start daemon (bootstrap + server)
  pink-transcriber stop            Stop daemon
  pink-transcriber status          Check if daemon is running
  pink-transcriber transcribe FILE Transcribe audio file
  pink-transcriber help            Show this help

Supported formats: any audio format (converted via ffmpeg)`,
		Commands: map[string]core.Command{
			"stop": {
				Desc: "Stop daemon",
				Run: func(args []string) error {
					if !core.IsRunning(serviceName) {
						fmt.Println("not running")
						return nil
					}
					return core.SendStop(serviceName)
				},
			},
			"status": {
				Desc: "Check if daemon is running",
				Run: func(args []string) error {
					if core.IsRunning(serviceName) {
						fmt.Println("running")
					} else {
						fmt.Println("not running")
					}
					return nil
				},
			},
			"transcribe": {
				Desc: "Transcribe audio file",
				Run: func(args []string) error {
					if len(args) < 1 {
						return fmt.Errorf("missing file argument")
					}
					return runTranscribe(args[0])
				},
			},
		},
	}, func(ctx context.Context) error {
		// Bootstrap whisper binary and model
		if err := bootstrap.EnsureReady(); err != nil {
			otel.Error(ctx, "bootstrap failed", otel.Attr{"error", err.Error()})
			return err
		}

		// Run whisper server and wait for shutdown
		return daemon.Run(ctx, whisperDir)
	})
}

func runTranscribe(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	if err := transcriber.CheckFFmpeg(); err != nil {
		return fmt.Errorf("ffmpeg not found")
	}

	if !transcriber.IsServerRunning() {
		return fmt.Errorf("server not running")
	}

	text, err := transcriber.Transcribe(filePath)
	if err != nil {
		return err
	}
	fmt.Println(text)
	return nil
}
