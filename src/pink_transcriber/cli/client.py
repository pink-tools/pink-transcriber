#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import sys
import socket
from pathlib import Path

from pink_transcriber import __version__
from pink_transcriber.config import (
    SUPPORTED_AUDIO_FORMATS, SOCKET_PATH, IS_WINDOWS, TCP_HOST, TCP_PORT
)


def validate_audio_file(file_path: str) -> None:
    if not os.path.exists(file_path):
        print(f"ERROR: File not found: {file_path}", file=sys.stderr)
        sys.exit(1)

    if not os.path.isfile(file_path):
        print(f"ERROR: Not a file: {file_path}", file=sys.stderr)
        sys.exit(1)

    ext = os.path.splitext(file_path)[1].lower()
    if ext not in SUPPORTED_AUDIO_FORMATS:
        print(f"ERROR: Unsupported format: {ext}", file=sys.stderr)
        supported_list = ', '.join(sorted(SUPPORTED_AUDIO_FORMATS))
        print(f"Supported formats: {supported_list}", file=sys.stderr)
        sys.exit(1)


def connect_to_server() -> socket.socket:
    if IS_WINDOWS:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect((TCP_HOST, TCP_PORT))
    else:
        sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        sock.connect(str(SOCKET_PATH))
    return sock


def transcribe(audio_path: str) -> str:
    sock = connect_to_server()

    try:
        sock.sendall(audio_path.encode() + b'\n')

        response = b''
        while True:
            chunk = sock.recv(4096)
            if not chunk:
                break
            response += chunk
            if response.endswith(b'\n'):
                break

        text = response.decode().strip()

        if text.startswith("ERROR:"):
            raise RuntimeError(text[7:])

        return text

    finally:
        sock.close()


def main() -> None:
    parser = argparse.ArgumentParser(
        prog='pink-transcriber',
        description='Voice transcription',
        epilog='Supported formats: ' + ', '.join(sorted(SUPPORTED_AUDIO_FORMATS))
    )

    parser.add_argument(
        'audio_file',
        nargs='?',
        help='Audio file to transcribe'
    )
    parser.add_argument(
        '--version',
        action='version',
        version=f'%(prog)s {__version__}'
    )
    parser.add_argument(
        '--health',
        action='store_true',
        help='Check if transcription server is running'
    )

    args = parser.parse_args()

    if args.health:
        if not IS_WINDOWS and SOCKET_PATH and not SOCKET_PATH.exists():
            print("not running", file=sys.stderr)
            sys.exit(1)

        try:
            sock = connect_to_server()
            sock.settimeout(2)
            sock.sendall(b"HEALTH\n")
            response = sock.recv(1024).decode().strip()
            sock.close()

            if response == "OK":
                print("idle")
                sys.exit(0)
            elif response == "LOADING":
                print("loading")
                sys.exit(0)
            elif response == "TRANSCRIBING":
                print("transcribing")
                sys.exit(0)
            else:
                print(f"unknown: {response}", file=sys.stderr)
                sys.exit(1)

        except socket.timeout:
            print("timeout", file=sys.stderr)
            sys.exit(1)
        except ConnectionRefusedError:
            print("not running", file=sys.stderr)
            sys.exit(1)
        except Exception as e:
            print(f"error: {e}", file=sys.stderr)
            sys.exit(1)

    if not args.audio_file:
        parser.print_help()
        sys.exit(1)

    audio_path = os.path.abspath(args.audio_file)
    validate_audio_file(audio_path)

    if not IS_WINDOWS and SOCKET_PATH and not SOCKET_PATH.exists():
        print("ERROR: Server not running", file=sys.stderr)
        sys.exit(1)

    try:
        text = transcribe(audio_path)
        print(text)
    except ConnectionRefusedError:
        print("ERROR: Server not running", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
