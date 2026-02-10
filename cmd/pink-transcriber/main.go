package main

import (
	"fmt"
	"os"

	"github.com/pink-tools/pink-core"
	"github.com/pink-tools/pink-transcriber/internal/transcriber"
)

var version = "dev"

const serviceName = "pink-transcriber"

func main() {
	core.Run(core.Config{
		Name:    serviceName,
		Version: version,
		Usage: `pink-transcriber - speech to text

Usage:
  pink-transcriber transcribe FILE  Transcribe audio file

Environment:
  WHISPER_LOCAL_ADDR      Local whisper (default: localhost:7465)
  TRANSCRIBE_SERVER_URL   HTTP fallback (default: https://transcribe.pinkhaired.com/transcribe)`,
		Commands: map[string]core.Command{
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
	}, nil)
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
