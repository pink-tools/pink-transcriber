import os
import sys
import json
from datetime import datetime, timezone
from pathlib import Path

IS_WINDOWS = sys.platform == 'win32'

if IS_WINDOWS:
    TCP_HOST = '127.0.0.1'
    TCP_PORT = 19876
    SOCKET_PATH = None
else:
    SOCKET_PATH = Path("/tmp/pink-transcriber.sock")
    TCP_HOST = None
    TCP_PORT = None

SUPPORTED_AUDIO_FORMATS = frozenset({
    '.aiff', '.flac', '.m4a', '.mp3', '.ogg', '.opus', '.wav'
})

SINGLETON_IDENTIFIERS = ['pink-transcriber', 'pink_transcriber', 'Pink Transcriber']


def log(body: str, severity: str = "INFO", **attributes) -> None:
    severity_map = {"INFO": 9, "WARN": 13, "ERROR": 17}
    entry = {
        "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%f000Z"),
        "severityNumber": severity_map.get(severity, 9),
        "severityText": severity,
        "body": body,
        "resource": {"service.name": "pink-transcriber"},
    }
    if attributes:
        entry["attributes"] = attributes
    print(json.dumps(entry), flush=True)


