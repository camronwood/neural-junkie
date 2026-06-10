import { useEffect, useState } from 'react';
import type { SettingsTabProps } from './settingsShared';

type HubSecurity = {
  hub_token_configured: boolean;
  auth_required: boolean;
  listen_all: boolean;
  loopback_only: boolean;
};

export function SecuritySettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const [hubSecurity, setHubSecurity] = useState<HubSecurity | null>(null);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/system/security`);
        if (!r.ok) return;
        const data = await r.json();
        if (!cancelled) setHubSecurity(data);
      } catch {
        if (!cancelled) setHubSecurity(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp]);

  if (!isActive) return null;

  return (
<div className="space-y-6">
    <div>
      <h3 className="text-lg font-semibold text-slack-text mb-2">Hub security</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Neural Junkie is local-first. The hub binds to loopback by default. For shared machines or LAN
        access, configure environment variables before starting the hub — see{' '}
        <a
          href="https://github.com/camronwood/neural-junkie/blob/main/docs/SECURITY.md"
          className="text-indigo-400 hover:underline"
          target="_blank"
          rel="noreferrer"
        >
          SECURITY.md
        </a>
        .
      </p>
      {hubSecurity ? (
        <ul className="text-sm space-y-2 text-slack-text">
          <li>
            <span className="text-slack-textMuted">Loopback only:</span>{' '}
            {hubSecurity.loopback_only ? 'Yes' : 'No (NEURAL_JUNKIE_LISTEN_ALL=1)'}
          </li>
          <li>
            <span className="text-slack-textMuted">Hub token configured:</span>{' '}
            {hubSecurity.hub_token_configured ? 'Yes' : 'No'}
          </li>
          <li>
            <span className="text-slack-textMuted">Strict auth (NEURAL_JUNKIE_AUTH_REQUIRED):</span>{' '}
            {hubSecurity.auth_required ? 'On' : 'Off'}
          </li>
        </ul>
      ) : (
        <p className="text-sm text-slack-textMuted">Could not read hub security status.</p>
      )}
      {hubSecurity?.listen_all && !hubSecurity.hub_token_configured && (
        <p className="mt-3 text-sm text-amber-400">
          Hub is listening on all interfaces without NEURAL_JUNKIE_HUB_TOKEN — set a hub token before exposing
          the hub on a network.
        </p>
      )}
    </div>
    <div className="rounded-lg border border-slack-border bg-slack-bgHover/30 p-4 text-xs text-slack-textMuted space-y-2">
      <p>
        <strong className="text-slack-text">Shared machine:</strong> set{' '}
        <code className="font-mono">NEURAL_JUNKIE_AUTH_REQUIRED=1</code> and{' '}
        <code className="font-mono">NEURAL_JUNKIE_HUB_TOKEN</code> to random secrets.
      </p>
      <p>
        JWT/API keys and user roles are planned for post-v1.0 — see{' '}
        <a
          href="https://github.com/camronwood/neural-junkie/blob/main/docs/PLATFORM_ROADMAP.md"
          className="text-indigo-400 hover:underline"
          target="_blank"
          rel="noreferrer"
        >
          PLATFORM_ROADMAP.md
        </a>
        .
      </p>
    </div>
  </div>

  );
}
