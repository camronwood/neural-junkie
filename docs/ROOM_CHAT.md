# Room chat

Room chat is an optional domain pack that adds “same room” LAN chat to Neural Junkie.

## What it is

- One person **hosts** an ephemeral room on their local hub.
- Others **join** from the desktop app using a host hub URL + join code.
- Everyone chats in a room channel (threads, @mentions, markdown).
- Guests can @mention the **host’s** agents, which respond on the host hub.

## Security model (v1)

- No cloud hub and no multi-hop relay.
- Guests are issued a hub session, but **channel ACLs restrict** them to room channels only.
- Room join is gated behind the `room-chat` capability; if the pack is disabled, room APIs return **404**.

## Troubleshooting

- Corporate WiFi “client isolation” may prevent devices from reaching the host hub. Use a different network or wired LAN.
- The host must enable LAN binding (Settings → Security → `listen_all`) and have a hub token configured.
