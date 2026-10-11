# Desktop Listen — local/dev checklist

Branch/worktree: `feat/assistant-desktop-listen` / `neural-junkie-wt-desktop-listen`.

Do **not** promote to staging/release until this checklist is green.

## Automated

- `go test ./internal/listen/ ./internal/agent/ ./internal/hub/ ./cmd/server/ -count=1`
- `cd desktop && npx vitest run src/listen src/api/chatAPI.test.ts`
- Hub with stub: `NJ_WHISPER_CMD=./scripts/whisper-stub.sh` then `./scripts/smoke-desktop-listen.sh`

## Paths to exercise

| Path | Expect |
|------|--------|
| Happy path (default mic) | Start → speak → Stop → note in `meeting_notes.json` with `source=desktop_listen` |
| Mic denied | Error in toolbar; no hub session left active |
| Missing whisper | `/api/assistant/listen/start` → `stt_unavailable`; toolbar Start disabled/hint |
| Cancel mid-listen | No new note; temp audio removed |
| Double start | HTTP 409 / already listening |
| Empty transcript stop | Error; no note |
| Device picker / missing device | Clear error if selected virtual device gone |
| Virtual device (optional) | BlackHole selected; non-silence while tone/Zoom routed in |
| Q&A | Ask Assistant about the recorded meeting; stays in chat with note context |

## Env

```text
NJ_WHISPER_CMD=./scripts/whisper-stub.sh   # CI / no model
# or real whisper-cli + ~/.neural-junkie/models/whisper/ggml-base.en.bin
```
