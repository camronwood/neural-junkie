# Keyboard shortcuts

Neural Junkie desktop app shortcuts (fixed defaults). `mod` = **⌘** on macOS, **Ctrl** on Windows/Linux.

See also **Settings → Keyboard** in the app for the live list from the shortcut registry.

## Global

| Shortcut | Action |
|----------|--------|
| `mod+,` | Open Settings |
| `mod+shift+p` | Command palette |
| `mod+shift+m` | Model library |
| `mod+f` | Find in chat (Monaco keeps its own find when the editor is focused) |
| `escape` | Close top overlay, or stop agents when none open |
| `mod+j` | Toggle terminal |

## Layout and panels

| Shortcut | Action |
|----------|--------|
| `mod+b` | Toggle channel sidebar |
| `mod+shift+e` | Toggle file explorer |
| `mod+shift+t` | Toggle task management |
| `mod+shift+g` | Toggle Git panel (dev pack) |
| `mod+shift+d` | Toggle problems panel (dev pack) |
| `mod+shift+u` | Jump to the oldest pending change card (dev pack) |
| `mod+shift+a` | Toggle My Agents panel |
| `mod+shift+c` | Toggle chat panel |
| `mod+shift+\` | Toggle toolbar sidebar (compact layout) |
| `mod+shift+i` | Toggle IDE vs team layout (dev pack) |
| `mod+shift+r` | New runbook |

## Navigation

| Shortcut | Action |
|----------|--------|
| `alt+up` / `alt+down` | Previous / next channel or DM |
| `mod+0` | Focus channel sidebar search |
| `mod+n` | Create channel |
| `mod+shift+n` | New DM wizard |
| `mod+shift+w` | Workspace switcher (dev pack) |

## IDE / dev pack

| Shortcut | Action |
|----------|--------|
| `mod+p` | Quick open file |
| `mod+shift+o` | Go to symbol |
| `mod+k` | Fast edit (when code editor open; terminal clear when terminal focused) |
| `mod+l` | Focus composer (IDE layout) |
| `mod+s` | Save active editor tab |
| `mod+shift+s` | Save all editor tabs |
| `mod+w` | Close active editor tab |
| `mod+shift+f` | Open code editor panel |
| `mod+tab` | Next editor tab |
| `mod+shift+tab` | Previous editor tab |

## Agent approvals

| Shortcut | Action |
|----------|--------|
| `mod+enter` | Approve / run first pending item |
| `mod+backspace` | Reject / dismiss first pending item |

## Thread

| Shortcut | Action |
|----------|--------|
| `mod+shift+]` | Close thread panel |

## Slash commands and telemetry

- **Slash command history:** When you send a `/command`, your typed line stays in channel history (styled as a command) and the hub posts the system response separately. Agents do not reply to slash lines.
- **Turn telemetry drawer:** Settings → Models & performance → **Turn telemetry drawer** shows live routing, tool, and activity events above the composer. Routing badges on completed messages remain available via **Routing badges on messages**.

## Manual QA matrix

| Area | macOS | Windows/Linux |
|------|-------|---------------|
| Channel sidebar `mod+b` | Toggle | Toggle (Ctrl+B) |
| File explorer `mod+shift+e` | Toggle | Toggle |
| Terminal `mod+j` | Toggle | Toggle |
| Settings `mod+,` | Opens | Opens |
| Escape stack | Closes modals top-first | Same |
| IDE quick open `mod+p` | Opens (dev pack) | Opens |
| Channel nav `alt+arrows` | Cycles channels | Cycles channels |
