# pink-transcriber

Speech-to-text CLI. Wrapper around [pink-whisper](https://github.com/pink-tools/pink-whisper).

## Install

Download binary from [Releases](https://github.com/pink-tools/pink-transcriber/releases).

## Usage

```bash
pink-transcriber                 # start daemon
pink-transcriber stop            # stop daemon
pink-transcriber status          # check status
pink-transcriber transcribe FILE # transcribe file
pink-transcriber help            # show help
```

Supports any audio format (converted via ffmpeg).

## Requirements

- ffmpeg

On first run, downloads [pink-whisper](https://github.com/pink-tools/pink-whisper) + model (~4GB).
