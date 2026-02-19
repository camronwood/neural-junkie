# Neural Junkie - Desktop App

A modern Tauri-based desktop application for the Neural Junkie with React and TypeScript.

## Features

- 🎨 Slack-inspired dark theme
- 💬 Real-time chat with AI agents via WebSocket
- 🤖 Agent status and expertise display
- 📝 Rich text message composition
- 🔄 Auto-reconnection
- 🖥️ Native desktop experience with small bundle size (~10-15MB)

## Prerequisites

- **Node.js** (v18 or higher)
- **npm** or **yarn**
- **Rust** (for Tauri builds)
  - Install via: `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`
- **Go** (for running the backend server)

## Installation

```bash
# Install dependencies
npm install
```

## Development

### 1. Start the Go backend server

In the project root:

```bash
make server
```

### 2. Start the desktop app

From the project root:

```bash
make gui
```

Or manually from desktop directory:

```bash
cd desktop
npm run tauri:dev
```

This will:
- Start Vite dev server on port 1420
- Launch the Tauri window
- Enable hot module reloading

## Building for Production

From project root:

```bash
make gui-build
```

Or manually from desktop directory:

```bash
npm run tauri:build
```

This creates platform-specific installers in `src-tauri/target/release/bundle/`:
- **macOS**: `.dmg` and `.app` bundle
- **Windows**: `.msi` installer
- **Linux**: `.deb` and `.AppImage`

## Project Structure

```
desktop/
├── src/
│   ├── api/              # HTTP API client
│   ├── components/       # React components
│   ├── hooks/            # Custom React hooks
│   ├── stores/           # Zustand state management
│   ├── types/            # TypeScript type definitions
│   ├── App.tsx           # Main app component
│   ├── main.tsx          # Entry point
│   └── styles.css        # Global styles
├── src-tauri/            # Tauri Rust backend
│   ├── src/
│   │   └── main.rs       # Minimal Rust code
│   ├── icons/            # App icons
│   ├── Cargo.toml        # Rust dependencies
│   └── tauri.conf.json   # Tauri configuration
├── public/               # Static assets
├── index.html            # HTML template
├── package.json          # Node dependencies
├── vite.config.ts        # Vite configuration
├── tailwind.config.js    # Tailwind CSS config
└── tsconfig.json         # TypeScript config
```

## Tech Stack

- **Tauri**: Rust-based desktop framework
- **React 18**: UI framework
- **TypeScript**: Type safety
- **Vite**: Build tool and dev server
- **Tailwind CSS**: Utility-first CSS
- **Zustand**: State management
- **WebSocket**: Real-time communication

## Configuration

### Server Address

Default: `localhost:8080`

You can change this in the login screen or update the default in `src/stores/chatStore.ts`.

### Window Size

Configured in `src-tauri/tauri.conf.json`:
- Default: 1000x700
- Minimum: 800x600

## Troubleshooting

### Rust not found
```bash
# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

### WebSocket connection fails
Make sure the Go backend server is running on `localhost:8080`:
```bash
make server
```

### Port 1420 already in use
Kill the process using port 1420 or change the port in `vite.config.ts`.

## Comparison: Tauri vs Fyne

| Feature | Tauri (This) | Fyne (Previous) |
|---------|--------------|-----------------|
| Bundle Size | ~10-15MB | ~30MB |
| Tech Stack | Rust + Web | Pure Go |
| Styling | Full CSS/Tailwind | Limited theming |
| UI Libraries | Entire npm ecosystem | Fyne widgets only |
| Rich Text | Native support | Limited |
| Development | Hot reload | Rebuild required |
| Performance | Fast (system webview) | Fast (native) |

## License

Same as parent project.

