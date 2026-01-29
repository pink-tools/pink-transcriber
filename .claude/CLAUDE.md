# pink-transcriber

Speech-to-text via whisper.cpp (Whisper Large V3)

```bash
pink-transcriber                 # Start daemon (bootstraps on first run)
pink-transcriber transcribe FILE # Transcribe audio file
pink-transcriber status          # Check if running
pink-transcriber stop            # Stop daemon
```

**Data:** `~/pink-tools/pink-transcriber/`, `~/pink-tools/pink-whisper/` (server + model)
