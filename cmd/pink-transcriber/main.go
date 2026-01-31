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

	port := os.Getenv("WHISPER_PORT")
	if port == "" {
		port = "7465"
	}
	transcriber.SetPort(port)

	whisperDir := bootstrap.GetWhisperDir()

	core.Run(core.Config{
		Name:    serviceName,
		Version: version,
		Usage: `pink-transcriber - speech to text

Usage:
  pink-transcriber                 Start daemon
  pink-transcriber stop            Stop daemon
  pink-transcriber status          Check if running
  pink-transcriber transcribe FILE Transcribe audio file`,
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
				Desc: "Check if running",
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
		if err := bootstrap.EnsureReady(); err != nil {
			otel.Error(ctx, "bootstrap failed", otel.Attr{"error", err.Error()})
			return err
		}
		return daemon.Run(ctx, whisperDir, port)
	})
}

func runTranscribe(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	if err := transcriber.CheckFFmpeg(); err != nil {
		return fmt.Errorf("ffmpeg not found")
	}

	text, err := transcriber.Transcribe(filePath)
	if err != nil {
		return err
	}
	fmt.Println(text)
	return nil
}
