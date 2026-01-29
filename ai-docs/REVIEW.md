# Code Review: pink-transcriber-private

**Date:** 2026-01-16
**Status:** CRITICAL BUG

## Architecture

CLI wrapper around whisper.cpp for speech-to-text.

**Components:**
- `cmd/pink-transcriber/main.go` — CLI entry
- `internal/daemon/` — Lifecycle management
- `internal/transcriber/` — Audio conversion + TCP protocol
- `internal/bootstrap/` — Auto-download dependencies

## Critical Bug (FIXED)

### isProcessAlive() Kills Process on Windows
**File:** `internal/daemon/process_windows.go:30-42`

**Problem:** Was sending SIGKILL to check if process exists.

**Fix applied:** Now uses Windows API `OpenProcess` + `GetExitCodeProcess` to check if process is alive without killing it.

## Legitimate Patterns (Not Bugs)

- Server readiness polling (necessary for external C++ process)
- Graceful SIGTERM → SIGKILL with 5s timeout (standard pattern)
- Process group signals (-pid) for child cleanup

## Code Quality

- Clean 566-line codebase
- Good platform separation
- No retry logic or hacks
- Only issue is Windows process detection

## Actions Required

~~1. Fix `isProcessAlive()` on Windows (CRITICAL)~~ DONE
