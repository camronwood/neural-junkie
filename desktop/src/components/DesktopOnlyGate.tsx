const RELEASES_URL = 'https://github.com/camronwood/neural-junkie/releases/latest';

export function DesktopOnlyGate() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-slack-bg text-slack-text p-8">
      <div className="max-w-lg text-center space-y-6">
        <h1 className="text-3xl font-bold text-white">Neural Junkie</h1>
        <p className="text-slack-textMuted leading-relaxed">
          The desktop app is required for terminal access, workspace file operations, and the
          integrated hub sidecar. The browser build is not supported.
        </p>
        <p className="text-sm text-slack-textMuted">
          Run <code className="font-mono text-slack-text bg-slack-bgHover px-1.5 py-0.5 rounded">npm run tauri:dev</code>{' '}
          for local development, or install a release build below.
        </p>
        <a
          href={RELEASES_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-block px-6 py-3 rounded-lg bg-slack-accent text-white font-medium hover:opacity-90 transition-opacity"
        >
          Download desktop app
        </a>
      </div>
    </div>
  );
}
