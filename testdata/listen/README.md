# Desktop Listen fixtures

CI uses `scripts/whisper-stub.sh` via `NJ_WHISPER_CMD` so tests do not download Whisper models.

For a real local smoke:

1. Install whisper.cpp `whisper-cli` to `~/.neural-junkie/bin/whisper-cli`
2. Download `ggml-base.en.bin` to `~/.neural-junkie/models/whisper/`
3. Start the hub, then run `scripts/smoke-desktop-listen.sh` (or use the desktop toolbar mic)
