# pink-transcriber

Speech-to-text daemon using Whisper Large V3. Auto-bootstraps on first run.

## Binaries

- `pink-transcriber-darwin-arm64` - macOS ARM64
- `pink-transcriber-linux-amd64` - Linux x64
- `pink-transcriber-windows-amd64.exe` - Windows x64

## Features

- Whisper Large V3 model (~4GB, downloaded automatically)
- Daemon mode with TCP server
- File transcription command
- Platform-specific acceleration (CoreML on macOS, CUDA on Linux/Windows)

## Requirements

- ffmpeg
