#!/usr/bin/env python3
from __future__ import annotations

import os
os.environ['PYTHONWARNINGS'] = 'ignore'
os.environ['NEMO_LOG_LEVEL'] = 'CRITICAL'
os.environ['HYDRA_FULL_ERROR'] = '0'
os.environ['HF_HUB_DISABLE_PROGRESS_BARS'] = '1'
os.environ['HF_HUB_DISABLE_TELEMETRY'] = '1'

import warnings
warnings.filterwarnings('ignore')

import logging
logging.disable(logging.WARNING)

try:
    import setproctitle
    setproctitle.setproctitle('Pink Transcriber')
except ImportError:
    pass

import asyncio
import signal

from pink_transcriber.config import log, SOCKET_PATH, IS_WINDOWS, TCP_HOST, TCP_PORT
from pink_transcriber.core import model
from pink_transcriber.daemon import worker
from pink_transcriber.daemon.singleton import ensure_single_instance


def run() -> None:
    log("starting")
    ensure_single_instance('pink-transcriber')

    backend = model.detect_backend()
    log("detected", backend=backend)

    model_name = model.get_model_name()

    cached = model.is_model_cached()
    if not cached:
        log("downloading", model=model_name)
        model.download_model()
        log("downloaded", model=model_name)

    log("loading", model=model_name)
    model.load_model()
    log("loaded", model=model_name, device=model.get_device())

    asyncio.run(serve())


async def serve() -> None:
    queue: asyncio.Queue = asyncio.Queue()
    worker_task = asyncio.create_task(worker.transcription_worker(queue))

    async def client_handler(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
        await worker.handle_client(reader, writer, queue)

    if IS_WINDOWS:
        server = await asyncio.start_server(client_handler, TCP_HOST, TCP_PORT)
        log("ready", transport="tcp", address=f"{TCP_HOST}:{TCP_PORT}")
    else:
        socket_path = SOCKET_PATH
        if socket_path.exists():
            socket_path.unlink()
        server = await asyncio.start_unix_server(client_handler, path=str(socket_path))
        log("ready", transport="unix", socket=str(socket_path))

    shutdown_event = asyncio.Event()
    loop = asyncio.get_running_loop()

    def handle_signal():
        shutdown_event.set()

    loop.add_signal_handler(signal.SIGINT, handle_signal)
    loop.add_signal_handler(signal.SIGTERM, handle_signal)

    await shutdown_event.wait()

    log("stopping")

    await queue.put(None)
    worker_task.cancel()

    server.close()
    await server.wait_closed()

    if not IS_WINDOWS and SOCKET_PATH and SOCKET_PATH.exists():
        SOCKET_PATH.unlink()

    log("stopped")


def cli_main() -> None:
    run()


if __name__ == "__main__":
    cli_main()
