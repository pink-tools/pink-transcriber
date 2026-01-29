# pink-transcriber

Go CLI wrapper for pink-whisper. Manages daemon lifecycle, auto-downloads dependencies, transcribes audio files.

**Repository:** https://github.com/pink-tools/pink-transcriber

**Language:** Go 1.25

## File Structure

```
pink-transcriber/
├── cmd/
│   └── pink-transcriber/
│       └── main.go              # CLI entry point
├── internal/
│   ├── bootstrap/
│   │   └── bootstrap.go         # Auto-download pink-whisper + model
│   ├── daemon/
│   │   ├── daemon.go            # Daemon start/stop/status
│   │   ├── process_unix.go      # Unix process management
│   │   └── process_windows.go   # Windows process management
│   └── transcriber/
│       └── transcriber.go       # Audio conversion + TCP client
├── .github/
│   └── workflows/
│       └── build.yml
├── ai-docs/
│   └── CLAUDE.md                # This file
├── go.mod
├── go.sum
├── README.md
├── RELEASE_NOTES.md
└── .gitignore
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     pink-transcriber CLI                     │
├─────────────────────────────────────────────────────────────┤
│  main.go                                                     │
│  ├── (no args) → bootstrap.EnsureReady() → daemon.Start()  │
│  ├── stop      → daemon.Stop()                              │
│  ├── status    → daemon.Status()                            │
│  ├── transcribe FILE → transcriber.Transcribe()             │
│  └── help      → printUsage()                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    bootstrap package                         │
│  1. Check pink-whisper binary exists                        │
│  2. Download from GitHub releases if missing                │
│  3. Check model exists                                      │
│  4. Download from HuggingFace if missing (~3GB)             │
│  5. Set permissions, remove macOS quarantine                │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     daemon package                           │
│  1. Exec: pink-whisper -m <model> -p 7465                   │
│  2. Write PID to ~/pink-tools/pink-transcriber/*.pid        │
│  3. Wait for TCP port 7465 to accept connections            │
│  4. Handle SIGINT for clean shutdown                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   transcriber package                        │
│  1. ffmpeg converts audio → PCM (16kHz, mono, s16le)        │
│  2. TCP connect to 127.0.0.1:7465                           │
│  3. Send: [4B size][PCM data]                               │
│  4. Receive: [4B size][UTF-8 text]                          │
└─────────────────────────────────────────────────────────────┘
```

## CLI Commands

```bash
pink-transcriber                 # Start daemon (downloads deps on first run)
pink-transcriber stop            # Stop daemon
pink-transcriber status          # Check if running
pink-transcriber transcribe FILE # Transcribe audio file
pink-transcriber help            # Show help
```

## File Locations

```
~/pink-tools/
├── pink-transcriber/
│   └── pink-transcriber.pid    # Daemon PID file
└── pink-whisper/
    ├── pink-whisper            # Server binary
    └── ggml-large-v3.bin       # Whisper model (~3GB)
```

## Bootstrap Mechanism

**getArtifactName()** selects platform-specific archive:

| Platform | CUDA | Artifact |
|----------|------|----------|
| darwin/arm64 | N/A | darwin-arm64-coreml.tar.gz |
| windows/amd64 | Yes | windows-amd64-cuda.zip |
| windows/amd64 | No | windows-amd64-cpu.zip |
| linux/amd64 | Yes | linux-amd64-cuda.tar.gz |
| linux/amd64 | No | linux-amd64-cpu.tar.gz |

**CUDA detection:** checks for `nvidia-smi` in PATH

**Download sources:**
- Binary: `https://github.com/pink-tools/pink-whisper/releases/latest/download/<artifact>`
- Model: `https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin`

**macOS:** removes quarantine with `xattr -d com.apple.quarantine`

## Daemon Management

### Start
1. Check not already running (via PID file)
2. Bootstrap dependencies
3. Exec: `pink-whisper -m ggml-large-v3.bin -p 7465`
4. Set process group (Unix)
5. Write PID file
6. Wait for TCP port (max 10s, 100ms intervals)
7. Handle SIGINT

### Stop
1. Read PID from file
2. Kill process group (Unix) or process (Windows)
3. Remove PID file

### Platform-Specific Process Handling

**Unix (process_unix.go):**
- `setProcessGroup()` - Sets Setpgid for process isolation
- `killProcess()` - Kills process group with SIGKILL (negative PID)
- `isProcessAlive()` - Signal 0 check

**Windows (process_windows.go):**
- `killProcess()` - Direct Process.Kill()
- `isProcessAlive()` - Attempts signal

## Transcription

**ffmpeg conversion:**
```bash
ffmpeg -i <input> -ar 16000 -ac 1 -f s16le -loglevel error -
```

**TCP Protocol (pink-whisper):**
```
Request:  [4B uint32 LE: size][N bytes: PCM data]
Response: [4B uint32 LE: size][M bytes: UTF-8 text]
```

**Server address:** 127.0.0.1:7465

## Dependencies

### Go
```
github.com/pink-tools/pink-otel  # JSON structured logging
```

### External
| Dependency | Purpose |
|------------|---------|
| ffmpeg | Audio format conversion |
| pink-whisper | whisper.cpp TCP server (auto-downloaded) |
| ggml-large-v3.bin | Whisper model (auto-downloaded) |

## Build Process

**Trigger:** Manual (workflow_dispatch)

**Build matrix:**
| GOOS | GOARCH | Output |
|------|--------|--------|
| darwin | arm64 | pink-transcriber-darwin-arm64 |
| linux | amd64 | pink-transcriber-linux-amd64 |
| windows | amd64 | pink-transcriber-windows-amd64.exe |

```bash
GOOS=<os> GOARCH=<arch> go build -o <output> ./cmd/pink-transcriber
```

## Logging

Uses `github.com/pink-tools/pink-otel` for JSON structured logging:
- `started` - Daemon started with PID and port
- `stopped` - Daemon stopped
- `bootstrap failed` - Download/setup error
- `start failed` / `already running` - Process issues
- `file not found` - Invalid transcription target
- `ffmpeg not found` - Missing dependency
- `server not running` - Daemon not started
- `transcribe failed` - Transcription error

## Related Projects

- **pink-whisper** - Whisper.cpp TCP server (github.com/pink-tools/pink-whisper)
- **pink-voice** - Voice input daemon (github.com/pink-tools/pink-voice)
- **pink-otel** - OTEL JSON logging (github.com/pink-tools/pink-otel)
