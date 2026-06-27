# Music creation pack

Official domain pack: **MusicExpert**, Ollama lyrics/style tags, and **ACE-Step 1.5** via a hub sidecar.

## Install

1. Desktop **Settings → Domain packs → Pack store** → **Music creation**
2. Enable the pack
3. One-time ACE-Step setup:

```bash
cd /Users/camronwood/development/projects/neural-junkie-pack-music-creation
make setup-acestep
```

## Use in chat

```text
/create-expert music MusicExpert
@MusicExpert write a 30s lo-fi instrumental
/generate-music lo-fi chill, 90 bpm | [Instrumental]
```

The agent calls `generate_music`; audio posts to the channel with an inline player.

## Demo mode (no ACE-Step)

```bash
export NJ_MUSIC_DEMO=1
# restart hub, then /generate-music test tone
```

## Architecture

| Layer | Role |
|-------|------|
| Ollama (`qwen2.5:7b`, `qwen3.5:9b`) | Lyrics and style tags |
| Pack sidecar | `POST /api/music/generate` → ACE-Step |
| Hub | `generate_music` tool, `/generate-music`, `generated_audio` metadata |

Pack repo: [camronwood/neural-junkie-pack-music-creation](https://github.com/camronwood/neural-junkie-pack-music-creation)

See also [PACKS.md](./PACKS.md).
