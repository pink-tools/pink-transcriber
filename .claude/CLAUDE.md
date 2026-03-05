# pink-transcriber

Speech-to-text CLI via whisper.cpp (Whisper Large V3).

```bash
{{PINK_TOOLS}}/pink-transcriber/pink-transcriber transcribe FILE
```

**Environment:**
- `WHISPER_LOCAL_ADDR` — local whisper server (default: `localhost:7465`)
- `TRANSCRIBE_SERVER_URL` — HTTP fallback (default: `https://transcribe.pinkhaired.com/transcribe`)
