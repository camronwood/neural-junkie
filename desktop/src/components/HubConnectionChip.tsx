import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type { ChatToolbarActionsLayout } from './ChatToolbarActions';
import type { SettingsTab } from './SettingsModal';
import type { ConnectionStatus } from '../hooks/useWebSocket';
import { getHubBaseURL, hubAuthHeaders, hubSessionHeaders } from '../config/hubUrl';

const iconBtn =
  'w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2';

const POPOVER_WIDTH = 256;

interface HubConnectionChipProps {
  layout: ChatToolbarActionsLayout;
  connectionStatus: ConnectionStatus;
  serverAddr: string;
  onReconnect: () => void;
  onOpenSettings?: (tab?: SettingsTab) => void;
}

type HubHealth = {
  status?: string;
  version?: string;
  uptime_secs?: number;
  agent_count?: number;
};

function statusLabel(ws: ConnectionStatus, healthOk: boolean | null): string {
  if (ws === 'connected' && healthOk === true) return 'Hub connected';
  if (ws === 'connected' && healthOk === false) return 'WS up, HTTP failing';
  if (ws === 'connecting') return 'Connecting to hub…';
  if (ws === 'error') return 'Hub connection error';
  return 'Hub disconnected';
}

function statusDotClass(ws: ConnectionStatus, healthOk: boolean | null): string {
  if (ws === 'connected' && healthOk !== false) return 'bg-green-400';
  if (ws === 'connecting') return 'bg-amber-400';
  if (ws === 'error' || healthOk === false) return 'bg-red-400';
  return 'bg-slack-textMuted';
}

function formatUptime(secs: number | undefined): string {
  if (secs == null || !Number.isFinite(secs) || secs < 0) return '—';
  const s = Math.floor(secs);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ${s % 60}s`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m`;
}

export function HubConnectionChip({
  layout,
  connectionStatus,
  serverAddr,
  onReconnect,
  onOpenSettings,
}: HubConnectionChipProps) {
  const [open, setOpen] = useState(false);
  const [health, setHealth] = useState<HubHealth | null>(null);
  const [healthOk, setHealthOk] = useState<boolean | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);
  const [popoverPos, setPopoverPos] = useState({ top: 0, left: 0 });
  const anchorRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const isVertical = layout === 'vertical';

  const refreshHealth = useCallback(async () => {
    const base = (serverAddr.startsWith('http') ? serverAddr : `http://${serverAddr}`).replace(
      /\/$/,
      '',
    );
    try {
      const res = await fetch(`${base}/api/health`, {
        headers: { ...hubAuthHeaders(), ...hubSessionHeaders() },
      });
      if (!res.ok) {
        setHealthOk(false);
        setHealthError(`HTTP ${res.status}`);
        return;
      }
      const data = (await res.json()) as HubHealth;
      setHealth(data);
      setHealthOk(true);
      setHealthError(null);
    } catch (e) {
      setHealthOk(false);
      setHealthError(e instanceof Error ? e.message : 'Health check failed');
    }
  }, [serverAddr]);

  useEffect(() => {
    void refreshHealth();
    const id = window.setInterval(() => void refreshHealth(), 5000);
    return () => window.clearInterval(id);
  }, [refreshHealth]);

  useLayoutEffect(() => {
    if (!open || !anchorRef.current) return;

    const pad = 8;
    const rect = anchorRef.current.getBoundingClientRect();
    const popoverH = popoverRef.current?.offsetHeight ?? 240;

    let left = isVertical ? rect.left - POPOVER_WIDTH - 8 : rect.right - POPOVER_WIDTH;
    let top = isVertical ? rect.top : rect.bottom + 4;

    if (left < pad) left = pad;
    if (left + POPOVER_WIDTH > window.innerWidth - pad) {
      left = Math.max(pad, window.innerWidth - POPOVER_WIDTH - pad);
    }
    if (top + popoverH > window.innerHeight - pad) {
      top = Math.max(pad, rect.top - popoverH - 4);
    }
    if (top < pad) top = pad;

    setPopoverPos({ top, left });
  }, [open, isVertical, connectionStatus, health, healthError]);

  const label = statusLabel(connectionStatus, healthOk);
  const hubOrigin = (() => {
    try {
      return getHubBaseURL();
    } catch {
      return serverAddr;
    }
  })();

  const popover = open
    ? createPortal(
        <>
          <div className="fixed inset-0 z-[250]" aria-hidden onMouseDown={() => setOpen(false)} />
          <div
            ref={popoverRef}
            className="fixed z-[251] w-64 rounded-lg border border-slack-border bg-slack-bg shadow-xl p-3 space-y-2"
            style={{ top: popoverPos.top, left: popoverPos.left }}
            role="dialog"
            aria-label="Hub connection status"
          >
            <div className="flex items-center gap-2">
              <span className={`h-2 w-2 rounded-full shrink-0 ${statusDotClass(connectionStatus, healthOk)}`} />
              <div className="min-w-0">
                <div className="text-sm font-semibold text-slack-text truncate">{label}</div>
                <div className="text-[11px] text-slack-textMuted truncate">{hubOrigin}</div>
              </div>
            </div>

            <div className="text-[11px] text-slack-textMuted space-y-0.5 border-t border-slack-border pt-2">
              <div>
                WebSocket: <span className="text-slack-text">{connectionStatus}</span>
              </div>
              <div>
                HTTP health:{' '}
                <span className="text-slack-text">
                  {healthOk == null ? 'checking…' : healthOk ? health?.status || 'ok' : healthError || 'fail'}
                </span>
              </div>
              {healthOk && (
                <>
                  <div>
                    Version: <span className="text-slack-text">{health?.version || '—'}</span>
                  </div>
                  <div>
                    Uptime: <span className="text-slack-text">{formatUptime(health?.uptime_secs)}</span>
                  </div>
                  <div>
                    Agents: <span className="text-slack-text">{health?.agent_count ?? '—'}</span>
                  </div>
                </>
              )}
            </div>

            <div className="flex flex-wrap gap-1.5 pt-1">
              <button
                type="button"
                onClick={() => {
                  onReconnect();
                  void refreshHealth();
                }}
                className="px-2.5 py-1 text-xs rounded bg-teal-700/50 text-teal-100 hover:bg-teal-700/70"
              >
                Reconnect
              </button>
              <button
                type="button"
                onClick={() => void refreshHealth()}
                className="px-2.5 py-1 text-xs rounded bg-slack-bgHover text-slack-textMuted hover:text-slack-text"
              >
                Refresh
              </button>
            </div>

            {onOpenSettings && (
              <button
                type="button"
                onClick={() => {
                  setOpen(false);
                  onOpenSettings('connection');
                }}
                className="w-full text-left text-xs text-teal-300 hover:text-teal-200"
              >
                Connection settings…
              </button>
            )}
          </div>
        </>,
        document.body,
      )
    : null;

  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={`${iconBtn} bg-slate-700 hover:bg-slate-600 text-white relative focus-visible:outline-slate-300`}
        title={label}
        aria-label={label}
        aria-expanded={open}
        data-testid="hub-connection-chip"
      >
        <span className="text-[10px] font-bold tracking-tight">HUB</span>
        <span
          className={`absolute -bottom-0.5 -right-0.5 h-2 w-2 rounded-full ring-1 ring-slack-bg ${statusDotClass(connectionStatus, healthOk)}`}
        />
      </button>
      {popover}
    </>
  );
}
