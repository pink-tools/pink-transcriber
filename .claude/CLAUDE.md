# pink-transcriber

Speech-to-text CLI via whisper.cpp (Whisper Large V3).

```bash
pink-transcriber transcribe FILE   # Transcribe audio file
```

**Environment:**
- `WHISPER_LOCAL_ADDR` — local whisper server (default: `localhost:7465`)
- `TRANSCRIBE_SERVER_URL` — HTTP fallback (default: `https://transcribe.pinkhaired.com/transcribe`)

**Data:** `/Users/pink-tools/pink-transcriber/`, `/Users/pink-tools/pink-whisper/` (server + model)
