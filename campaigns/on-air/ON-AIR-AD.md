# On Air — hero ads

Station ID, not a feature sheet. Neural Junkie as a late-night **callsign**: brand first, one carrier line, tight copy. No UI chrome.

**Campaign:** On Air  
**Style:** Station ID / callsign  
**Format:** 1080×1080 heroes (LinkedIn / X feed)  
**CTA:** https://github.com/camronwood/neural-junkie/releases/latest

**Regenerate:**

```bash
./scripts/compose-on-air-ad.sh all          # baseline + team
./scripts/compose-on-air-ad.sh baseline
./scripts/compose-on-air-ad.sh team
```

| Variant | Asset | Job |
|---------|-------|-----|
| **baseline** | `creatives/on-air-hero-1080.png` | Brand callsign |
| **team** | `creatives/on-air-team-1080.png` | Differentiates beyond “runs local AI” |

---

## Style bible

| Token | Value | Role |
|-------|-------|------|
| Field | `#0d0f14` | Full-bleed ground (site `--bg-deep`) |
| Signal | `#e94560` | One thin horizontal carrier + center mark |
| Ink | `#e8eaef` | Brand wordmark + primary line |
| Quiet | `#8b93a7` | Sub-line + footer |
| Display | Avenir Next Condensed Heavy | Brand |
| Readout | Menlo | Support lines + footer |

**Layout**

- Brand wordmark dominates the upper half — edge-to-edge presence
- Middle: one rose carrier line with a small diamond at center
- Below: one hard line (team adds one quiet sub-line)
- Bottom micro + release URL
- Subtle procedural grain only

**This is not**

- Product screenshots or fake UI
- Habit cream / Georgia letters
- Walk Into a Room dark joke cards
- Feature grids, agent chips, teal/amber/pink accent stacks, purple glow, pills

**Positioning note (team)**

Lots of apps run local models. Neural Junkie’s wedge is a **multi-agent desktop hub**: specialist agents from domain packs, human approval on file/tool changes, BYOM (local *or* cloud, per agent) — not a single local chatbot.

---

## Copy on image

### Baseline

- **Brand:** NEURAL JUNKIE  
- **Line:** Any model. On yours.  
- **Footer:** OPEN SOURCE · LOCAL FIRST  

### Team (differentiator)

- **Brand:** NEURAL JUNKIE  
- **Line:** Not another local chat.  
- **Sub:** Specialists. Approvals. Your models.  
- **Footer:** MULTI-AGENT · HUMAN-IN-THE-LOOP · BYOM  

---

## LinkedIn / X paste — baseline

Do not paste `---` into LinkedIn. Copy **PASTE START** through **PASTE END** only.

### PASTE START

Most AI tools sound like a lobby. Always on. Always someone else's room.

I wanted a callsign — a place that identifies as yours. Any model. Local when you want it. One workspace that doesn't wipe the desk every morning.

Neural Junkie. On air from your machine.

https://github.com/camronwood/neural-junkie/releases/latest

### PASTE END

**Image:** `on-air-hero-1080.png`  
**First comment (optional):** “Station ID, not a feature list.”

---

## LinkedIn / X paste — team

### PASTE START

Running a model on your laptop is table stakes now. That isn't the product.

Neural Junkie is a multi-agent desktop hub: specialist agents from domain packs, collaborate sessions, and you approve every file change and tool call before it lands. Bring your own model — Ollama, Claude, OpenAI-compatible, LM Studio — different brains per agent if you want.

Not another local chat. A team you keep the key for.

https://github.com/camronwood/neural-junkie/releases/latest

### PASTE END

**Image:** `on-air-team-1080.png`  
**First comment (optional):** “Local is the floor. Specialists + approvals are the product.”
