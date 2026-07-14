import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type { ChatToolbarActionsLayout } from './ChatToolbarActions';
import type { SettingsTab } from './SettingsModal';
import { useSettingsStore } from '../stores/settingsStore';
import {
  fetchOllamaRuntimeStatus,
  restartOllamaRuntime,
  startOllamaRuntime,
  stopOllamaRuntime,
  updateOllamaRuntime,
  type OllamaRuntimeStatus,
} from '../utils/ollamaRuntime';

const iconBtn =
  'w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2';

const POPOVER_WIDTH = 256;

interface OllamaRuntimeChipProps {
  layout: ChatToolbarActionsLayout;
  serverAddr: string;
  onOpenModelLibrary: () => void;
  onOpenSettings?: (tab?: SettingsTab) => void;
}

function statusLabel(status: OllamaRuntimeStatus | null): string {
  if (!status) return 'Checking Ollama…';
  if (status.running) {
    return status.bundled ? 'Running (bundled)' : 'Running';
  }
  if (status.installed) {
    return status.bundled ? 'Bundled (stopped)' : 'Installed (stopped)';
  }
  return 'Not installed';
}

function statusDotClass(status: OllamaRuntimeStatus | null): string {
  if (!status) return 'bg-slack-textMuted';
  if (status.running) return 'bg-green-400';
  if (status.installed) return 'bg-amber-400';
  return 'bg-red-400';
}

export function OllamaRuntimeChip({
  layout,
  serverAddr,
  onOpenModelLibrary,
  onOpenSettings,
}: OllamaRuntimeChipProps) {
  const defaultModel = useSettingsStore((s) => s.integrations.ollama.defaultModel);
  const [open, setOpen] = useState(false);
  const [status, setStatus] = useState<OllamaRuntimeStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [updateProgress, setUpdateProgress] = useState('');
  const [popoverPos, setPopoverPos] = useState({ top: 0, left: 0 });
  const anchorRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const isVertical = layout === 'vertical';

  const refresh = useCallback(async () => {
    try {
      const next = await fetchOllamaRuntimeStatus(serverAddr);
      setStatus(next);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to read Ollama status');
    }
  }, [serverAddr]);

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 8000);
    return () => window.clearInterval(id);
  }, [refresh]);

  useLayoutEffect(() => {
    if (!open || !anchorRef.current) return;

    const pad = 8;
    const rect = anchorRef.current.getBoundingClientRect();
    const popoverH = popoverRef.current?.offsetHeight ?? 280;

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
  }, [open, isVertical, status, error, defaultModel, updateProgress]);

  const runAction = async (action: 'start' | 'stop' | 'restart') => {
    setBusy(true);
    setError(null);
    try {
      if (action === 'start') await startOllamaRuntime(serverAddr);
      else if (action === 'stop') await stopOllamaRuntime(serverAddr);
      else await restartOllamaRuntime(serverAddr);
      await new Promise((r) => setTimeout(r, action === 'stop' ? 400 : 1200));
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ollama action failed');
    } finally {
      setBusy(false);
    }
  };

  const runUpdate = async () => {
    setBusy(true);
    setError(null);
    setUpdateProgress('Starting update…');
    try {
      await updateOllamaRuntime(serverAddr, (msg) => setUpdateProgress(msg));
      setUpdateProgress('');
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ollama update failed');
    } finally {
      setBusy(false);
    }
  };

  const chipTitle = status?.running
    ? `Ollama running${defaultModel ? ` — default ${defaultModel}` : ''}${status.updateAvailable ? ' — update available' : ''}`
    : 'Ollama local runtime';

  const popover = open
    ? createPortal(
        <>
          <div
            className="fixed inset-0 z-[250]"
            aria-hidden
            onMouseDown={() => setOpen(false)}
          />
          <div
            ref={popoverRef}
            className="fixed z-[251] w-64 rounded-lg border border-slack-border bg-slack-bg shadow-xl p-3 space-y-3"
            style={{ top: popoverPos.top, left: popoverPos.left }}
            role="dialog"
            aria-label="Ollama runtime"
            onMouseDown={(e) => e.stopPropagation()}
          >
            <div>
              <p className="text-xs font-semibold text-slack-text">Ollama runtime</p>
              <div className="mt-1 flex items-center gap-2">
                <span className={`w-2 h-2 rounded-full shrink-0 ${statusDotClass(status)}`} />
                <span className="text-xs text-slack-textMuted">{statusLabel(status)}</span>
              </div>
              {status?.version && (
                <p className="mt-1 text-[11px] text-slack-textMuted font-mono truncate">
                  {status.version}
                  {status.recommendedVersion ? ` → want ${status.recommendedVersion}` : ''}
                </p>
              )}
              {status?.updateAvailable && (
                <p className="mt-1 text-[11px] text-amber-300">
                  Update available{status.recommendedVersion ? ` (${status.recommendedVersion})` : ''}.
                </p>
              )}
              {defaultModel && (
                <p className="mt-1 text-[11px] text-slack-textMuted">
                  Default model: <span className="font-mono text-slack-text">{defaultModel}</span>
                </p>
              )}
            </div>

            {error && <p className="text-xs text-red-400">{error}</p>}
            {updateProgress && <p className="text-[11px] text-slack-textMuted">{updateProgress}</p>}

            <div className="flex flex-wrap gap-2">
              {status?.updateAvailable && status.updateSupported !== false && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void runUpdate()}
                  className="px-2.5 py-1 text-xs rounded bg-amber-700/50 text-amber-100 hover:bg-amber-700/70 disabled:opacity-50"
                >
                  Update Ollama
                </button>
              )}
              {status?.running ? (                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void runAction('stop')}
                  className="px-2.5 py-1 text-xs rounded bg-red-700/40 text-red-200 hover:bg-red-700/60 disabled:opacity-50"
                >
                  Stop
                </button>
              ) : (
                <button
                  type="button"
                disabled={busy}
                onClick={() => void runAction('start')}
                  className="px-2.5 py-1 text-xs rounded bg-green-700/40 text-green-200 hover:bg-green-700/60 disabled:opacity-50"
                >
                  Start
                </button>
              )}
              <button
                type="button"
              disabled={busy}
              onClick={() => void runAction('restart')}
                className="px-2.5 py-1 text-xs rounded bg-slack-bgHover text-slack-text hover:bg-slack-border disabled:opacity-50"
              >
                Restart
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => void refresh()}
                className="px-2.5 py-1 text-xs rounded bg-slack-bgHover text-slack-textMuted hover:text-slack-text disabled:opacity-50"
              >
                Refresh
              </button>
            </div>

            <div className="border-t border-slack-border pt-2 space-y-1">
              <button
                type="button"
                onClick={() => {
                  setOpen(false);
                  onOpenModelLibrary();
                }}
                className="w-full text-left text-xs text-teal-300 hover:text-teal-200"
              >
                Open model library…
              </button>
              {onOpenSettings && (
                <button
                  type="button"
                  onClick={() => {
                    setOpen(false);
                    onOpenSettings('providers');
                  }}
                  className="w-full text-left text-xs text-slack-textMuted hover:text-slack-text"
                >
                  Ollama settings…
                </button>
              )}
            </div>
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
        onClick={() => setOpen((v) => !v)}
        className={`${iconBtn} bg-teal-700 hover:bg-teal-600 text-white text-[10px] font-bold focus-visible:outline-teal-400 relative`}
        title={chipTitle}
        aria-label="Ollama runtime status and controls"
        aria-expanded={open}
        data-testid="ollama-runtime-chip"
      >
        OLL
        <span
          className={`absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full border border-slack-bgHover ${
            status?.updateAvailable ? 'bg-amber-400' : statusDotClass(status)
          }`}
          aria-hidden
        />
      </button>
      {popover}
    </>
  );
}
