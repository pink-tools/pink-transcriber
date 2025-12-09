from __future__ import annotations

import asyncio
import time
from pathlib import Path
from dataclasses import dataclass

from pink_transcriber.config import log
from pink_transcriber.core import model

_current_state: str = "idle"


@dataclass
class TranscriptionRequest:
    audio_path: str
    result_future: asyncio.Future


def get_state() -> str:
    global _current_state
    return _current_state


async def transcription_worker(queue: asyncio.Queue[TranscriptionRequest]) -> None:
    global _current_state
    while True:
        try:
            request = await queue.get()

            if request is None:
                break

            try:
                _current_state = "transcribing"
                loop = asyncio.get_event_loop()
                text = await loop.run_in_executor(None, model.transcribe, request.audio_path)
                request.result_future.set_result(text)
            except Exception as e:
                request.result_future.set_exception(e)
            finally:
                _current_state = "idle"
                queue.task_done()

        except asyncio.CancelledError:
            break
        except Exception:
            pass


async def handle_client(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    queue: asyncio.Queue[TranscriptionRequest]
) -> None:
    start_time = time.time()

    try:
        data = await reader.readline()
        message = data.decode().strip()

        if message == "HEALTH":
            if not model.is_loaded():
                writer.write(b"LOADING\n")
            elif get_state() == "transcribing":
                writer.write(b"TRANSCRIBING\n")
            else:
                writer.write(b"OK\n")
            await writer.drain()
            writer.close()
            await writer.wait_closed()
            return

        audio_path = message

        if not audio_path:
            error_msg = f"ERROR: No audio path provided\n".encode()
            writer.write(error_msg)
            await writer.drain()
            writer.close()
            await writer.wait_closed()
            return

        filename = Path(audio_path).name
        log("transcribing", file=filename)

        result_future = asyncio.Future()
        request = TranscriptionRequest(audio_path=audio_path, result_future=result_future)
        await queue.put(request)

        text = await result_future

        elapsed = time.time() - start_time
        log("transcribed", file=filename, duration=round(elapsed, 2), chars=len(text))

        response = text.encode() + b'\n'
        writer.write(response)
        await writer.drain()

    except (BrokenPipeError, ConnectionResetError):
        pass

    except FileNotFoundError as e:
        log("file not found", severity="ERROR", error=str(e))
        try:
            error_msg = f"ERROR: {str(e)}\n".encode()
            writer.write(error_msg)
            await writer.drain()
        except (BrokenPipeError, ConnectionResetError):
            pass

    except Exception as e:
        log("transcription error", severity="ERROR", error=str(e))
        try:
            error_msg = f"ERROR: {str(e)}\n".encode()
            writer.write(error_msg)
            await writer.drain()
        except (BrokenPipeError, ConnectionResetError):
            pass

    finally:
        try:
            writer.close()
            await writer.wait_closed()
        except (BrokenPipeError, ConnectionResetError):
            pass
