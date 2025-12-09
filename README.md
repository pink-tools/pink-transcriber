# Pink Transcriber

Speech-to-text daemon. Supports MPS (Apple Silicon), CUDA (NVIDIA), CPU backends.

## Install

```bash
git clone https://github.com/pink-tools/pink-transcriber.git
cd pink-transcriber

# Detect backend and install
if [[ $(uname -m) == "arm64" && $(uname) == "Darwin" ]]; then
  backend="mps"; python_version="3.12"
elif nvidia-smi &>/dev/null; then
  backend="cuda"; python_version="3.11"
else
  backend="cpu"; python_version="3.11"
fi
echo "$backend" > .backend
uv venv --python $python_version --clear
uv pip install -e ".[$backend]"
sudo ln -sf $(pwd)/.venv/bin/pink-transcriber /usr/local/bin/
sudo ln -sf $(pwd)/.venv/bin/pink-transcriber-server /usr/local/bin/
```

## Usage

```bash
pink-transcriber-server   # Start daemon
pink-transcriber audio.ogg # Transcribe file
pink-transcriber --health  # Check status
```
