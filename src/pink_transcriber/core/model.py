from __future__ import annotations

import os
import sys
from typing import Any, Optional

from pink_transcriber.config import log, IS_WINDOWS

_model: Optional[Any] = None
_device: Optional[str] = None
_backend: Optional[str] = None


def is_model_cached() -> bool:
    from pathlib import Path
    backend = detect_backend()
    hf_cache = Path.home() / ".cache" / "huggingface" / "hub"

    if backend == 'mps':
        return (hf_cache / "models--nvidia--parakeet-tdt-0.6b-v3").exists()
    elif backend == 'cuda':
        return (hf_cache / "models--Systran--faster-whisper-large-v3").exists()
    elif backend == 'cpu':
        return (hf_cache / "models--deepdml--faster-whisper-large-v3-turbo-ct2").exists()
    return False


def detect_backend() -> str:
    from pathlib import Path

    backend_file = Path(__file__).resolve().parent.parent.parent.parent / ".backend"
    if backend_file.exists():
        return backend_file.read_text().strip()

    return 'none'


def download_model() -> None:
    backend = detect_backend()
    os.environ['HF_HUB_DISABLE_PROGRESS_BARS'] = '1'

    if backend == 'mps':
        from huggingface_hub import snapshot_download
        snapshot_download("nvidia/parakeet-tdt-0.6b-v3")
    elif backend == 'cuda':
        from faster_whisper.utils import download_model as fw_download
        fw_download("large-v3")
    elif backend == 'cpu':
        from faster_whisper.utils import download_model as fw_download
        fw_download("deepdml/faster-whisper-large-v3-turbo-ct2")


def load_model() -> None:
    global _model, _device, _backend

    _backend = detect_backend()

    if _backend == 'mps':
        _load_mps()
    elif _backend == 'cuda':
        _load_cuda()
    elif _backend == 'cpu':
        _load_cpu()
    else:
        log("no backend available", severity="ERROR")
        sys.exit(1)


def _load_mps() -> None:
    global _model, _device

    os.environ['NEMO_LOG_LEVEL'] = 'CRITICAL'
    os.environ['HYDRA_FULL_ERROR'] = '0'

    import nemo.collections.asr as nemo_asr
    import torch
    import logging

    logging.getLogger('nemo_logger').setLevel(logging.CRITICAL)
    logging.getLogger('nemo').setLevel(logging.CRITICAL)
    logging.getLogger('pytorch_lightning').setLevel(logging.CRITICAL)
    logging.getLogger('lightning').setLevel(logging.CRITICAL)
    logging.getLogger('lightning.pytorch').setLevel(logging.CRITICAL)

    _model = nemo_asr.models.ASRModel.from_pretrained("nvidia/parakeet-tdt-0.6b-v3")

    if torch.backends.mps.is_available():
        try:
            _model = _model.to('mps')
            _device = 'MPS'
        except Exception:
            _model = _model.to('cpu')
            _device = 'CPU'
    else:
        _model = _model.to('cpu')
        _device = 'CPU'


def _load_cuda() -> None:
    global _model, _device

    if not IS_WINDOWS:
        try:
            import ctypes
            import nvidia.cudnn, nvidia.cublas
            for m in [nvidia.cudnn, nvidia.cublas]:
                lib_dir = os.path.join(m.__path__[0], "lib")
                for f in os.listdir(lib_dir):
                    if '.so' in f:
                        try:
                            ctypes.CDLL(os.path.join(lib_dir, f), mode=ctypes.RTLD_GLOBAL)
                        except:
                            pass
        except:
            pass

    from faster_whisper import WhisperModel
    import torch

    if torch.cuda.is_available():
        device = "cuda"
        compute_type = "float16"
    else:
        device = "cpu"
        compute_type = "int8"

    _model = WhisperModel(
        "large-v3",
        device=device,
        compute_type=compute_type,
    )
    _device = f"{device.upper()} ({compute_type.upper()})"


def _load_cpu() -> None:
    global _model, _device

    from faster_whisper import WhisperModel

    _model = WhisperModel(
        "deepdml/faster-whisper-large-v3-turbo-ct2",
        device="cpu",
        compute_type="int8",
    )
    _device = "CPU (INT8)"


def transcribe(audio_path: str) -> str:
    if _model is None:
        raise RuntimeError("Model not loaded")

    if not os.path.exists(audio_path):
        raise FileNotFoundError(f"Audio file not found: {audio_path}")

    if _backend == 'mps':
        return _transcribe_mps(audio_path)
    else:
        return _transcribe_whisper(audio_path)


def _transcribe_mps(audio_path: str) -> str:
    old_stdout_fd = os.dup(1)
    old_stderr_fd = os.dup(2)
    devnull_fd = os.open(os.devnull, os.O_WRONLY)

    try:
        os.dup2(devnull_fd, 1)
        os.dup2(devnull_fd, 2)
        result = _model.transcribe([audio_path], verbose=False, batch_size=1)
    finally:
        os.dup2(old_stdout_fd, 1)
        os.dup2(old_stderr_fd, 2)
        os.close(devnull_fd)
        os.close(old_stdout_fd)
        os.close(old_stderr_fd)

    if isinstance(result, list) and len(result) > 0:
        first_result = result[0]
        if hasattr(first_result, 'text'):
            return first_result.text
        else:
            return str(first_result)
    else:
        return str(result) if result else ""


def _transcribe_whisper(audio_path: str) -> str:
    segments, info = _model.transcribe(
        audio_path,
        beam_size=5,
        vad_filter=_backend == 'cpu',
        language=None,
    )

    text_segments = [segment.text.strip() for segment in segments]
    result = " ".join(text_segments)
    return result if result else ""


def get_device() -> str:
    return _device or "Unknown"


def get_model_name() -> str:
    backend = detect_backend()
    if backend == 'mps':
        return "nvidia/parakeet-tdt-0.6b-v3"
    elif backend == 'cuda':
        return "Systran/faster-whisper-large-v3"
    elif backend == 'cpu':
        return "deepdml/faster-whisper-large-v3-turbo-ct2"
    return "unknown"


def is_loaded() -> bool:
    return _model is not None
