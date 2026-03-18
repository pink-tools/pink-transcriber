# pink-transcriber

Speech-to-text CLI. Transcribes audio files using Whisper Large V3. Tries local [pink-whisper](https://github.com/pink-tools/pink-whisper) first, falls back to remote server.

## Install

Download binary from [Releases](https://github.com/pink-tools/pink-transcriber/releases), or via pink-orchestrator:

```bash
pink-orchestrator --service-download pink-transcriber
```

## Requirements

- ffmpeg

## Usage

```bash
pink-transcriber transcribe audio.ogg    # Transcribe file → text to stdout
```

Supports any audio format (converted via ffmpeg). Long audio (>2 min) is automatically chunked.

## How It Works

1. Checks if whisper.cpp TCP server is running on `localhost:7465`
2. If available — transcribes locally (fast, no network)
3. If unavailable — sends to remote HTTP endpoint

## Configuration

Optional environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `WHISPER_LOCAL_ADDR` | `localhost:7465` | Local whisper.cpp TCP address |
| `TRANSCRIBE_SERVER_URL` | `https://transcribe.pinkhaired.com/transcribe` | Remote fallback URL |

## Build from Source

```bash
git clone https://github.com/pink-tools/pink-transcriber.git
cd pink-transcriber
go build ./cmd/pink-transcriber
```
