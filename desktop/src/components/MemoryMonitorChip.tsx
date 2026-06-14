import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type { ChatToolbarActionsLayout } from './ChatToolbarActions';
import type { SettingsTab } from './SettingsModal';
import {
  fetchSystemMemory,
  formatMemoryBytes,
  memoryPressureClass,
  memoryPressureLevel,
  memoryPressureTextClass,
  MEMORY_MONITOR_POLL_MS,
  type SystemMemorySnapshot,
} from '../utils/memoryMonitor';

const iconBtn =
  'w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2';

const POPOVER_WIDTH = 288;

interface MemoryMonitorChipProps {
  layout: ChatToolbarActionsLayout;
  serverAddr: string;
  onOpenModelLibrary: () => void;
  onOpenSettings?: (tab?: SettingsTab) => void;
}

export function MemoryMonitorChip({
  layout,
  serverAddr,
  onOpenModelLibrary,
  onOpenSettings,
}: MemoryMonitorChipProps) {
  const [open, setOpen] = useState(false);
  const [snapshot, setSnapshot] = useState<SystemMemorySnapshot | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [popoverPos, setPopoverPos] = useState({ top: 0, left: 0 });
  const anchorRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const isVertical = layout === 'vertical';

  const refresh = useCallback(async () => {
    try {
      const next = await fetchSystemMemory(serverAddr);
      if (next) {
        setSnapshot(next);
        setError(null);
      } else {
        setError('Could not read memory stats');
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to read memory stats');
    }
  }, [serverAddr]);

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), MEMORY_MONITOR_POLL_MS);
    return () => window.clearInterval(id);
  }, [refresh]);

  useLayoutEffect(() => {
    if (!open || !anchorRef.current) return;

    const pad = 8;
    const rect = anchorRef.current.getBoundingClientRect();
    const popoverH = popoverRef.current?.offsetHeight ?? 320;

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
  }, [open, isVertical, snapshot, error]);

  const usedPercent = snapshot?.used_percent ?? 0;
  const level = memoryPressureLevel(usedPercent);
  const loadedCount = snapshot?.ollama.loaded_models.length ?? 0;
  const primaryModel = snapshot?.ollama.loaded_models[0]?.name;

  const chipTitle = snapshot
    ? `System RAM ${Math.round(usedPercent)}% used${
        primaryModel ? ` — ${primaryModel} loaded` : loadedCount === 0 ? ' — no models loaded' : ''
      }`
    : 'System memory monitor';

  const popover = open
    ? createPortal(
        <>
          <div className="fixed inset-0 z-[250]" aria-hidden onMouseDown={() => setOpen(false)} />
          <div
            ref={popoverRef}
            className="fixed z-[251] w-72 rounded-lg border border-slack-border bg-slack-bg shadow-xl p-3 space-y-3"
            style={{ top: popoverPos.top, left: popoverPos.left }}
            role="dialog"
            aria-label="System memory monitor"
            onMouseDown={(e) => e.stopPropagation()}
          >
            <div>
              <p className="text-xs font-semibold text-slack-text">System memory</p>
              {snapshot ? (
                <>
                  <div className="mt-2 flex items-center justify-between text-xs">
                    <span className={`font-semibold ${memoryPressureTextClass(level)}`}>
                      {Math.round(usedPercent)}% used
                    </span>
                    <span className="text-slack-textMuted">
                      {formatMemoryBytes(snapshot.available_bytes)} free
                    </span>
                  </div>
                  <div className="mt-2 h-2 rounded-full bg-slack-bgHover overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all ${memoryPressureClass(level)}`}
                      style={{ width: `${Math.min(100, Math.max(0, usedPercent))}%` }}
                    />
                  </div>
                  <p className="mt-2 text-[11px] text-slack-textMuted">
                    {formatMemoryBytes(snapshot.used_bytes)} used of {formatMemoryBytes(snapshot.total_bytes)}
                    {' · '}
                    {snapshot.tier} tier
                  </p>
                  {(snapshot.app_memory_bytes ?? 0) > 0 && (
                    <div className="mt-2 space-y-0.5 text-[11px] text-slack-textMuted">
                      <p className="text-slack-textMuted/80">Activity Monitor breakdown</p>
                      <p>App {formatMemoryBytes(snapshot.app_memory_bytes ?? 0)}</p>
                      <p>Wired {formatMemoryBytes(snapshot.wired_memory_bytes ?? 0)}</p>
                      <p>Compressed {formatMemoryBytes(snapshot.compressed_memory_bytes ?? 0)}</p>
                      <p className="pt-1 text-slack-textMuted/70">
                        Matches App + Wired + Compressed in Activity Monitor. Its top-line
                        &quot;Memory Used&quot; can read slightly higher.
                      </p>
                    </div>
                  )}
                </>
              ) : (
                <p className="mt-1 text-xs text-slack-textMuted">Reading memory stats…</p>
              )}
            </div>

            {snapshot && (
              <div className="border-t border-slack-border pt-2">
                <p className="text-xs font-semibold text-slack-text">Ollama loaded models</p>
                {!snapshot.ollama.running ? (
                  <p className="mt-1 text-xs text-slack-textMuted">Ollama is not running</p>
                ) : snapshot.ollama.loaded_models.length === 0 ? (
                  <p className="mt-1 text-xs text-slack-textMuted">No models loaded in RAM</p>
                ) : (
                  <ul className="mt-1 space-y-1">
                    {snapshot.ollama.loaded_models.map((model) => (
                      <li
                        key={model.name}
                        className="flex items-start justify-between gap-2 text-[11px]"
                      >
                        <span className="font-mono text-slack-text truncate" title={model.name}>
                          {model.name}
                        </span>
                        <span className="text-slack-textMuted shrink-0">
                          {formatMemoryBytes(model.size_bytes)}
                          {model.vram_bytes > 0 ? ` · VRAM ${formatMemoryBytes(model.vram_bytes)}` : ''}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
                {snapshot.ollama.loaded_bytes_total > 0 && (
                  <p className="mt-2 text-[11px] text-slack-textMuted">
                    Total loaded: {formatMemoryBytes(snapshot.ollama.loaded_bytes_total)}
                  </p>
                )}
              </div>
            )}

            {error && <p className="text-xs text-red-400">{error}</p>}

            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => void refresh()}
                className="px-2.5 py-1 text-xs rounded bg-slack-bgHover text-slack-textMuted hover:text-slack-text"
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
                    onOpenSettings('models-performance');
                  }}
                  className="w-full text-left text-xs text-slack-textMuted hover:text-slack-text"
                >
                  Memory monitor settings…
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
        className={`${iconBtn} bg-violet-700 hover:bg-violet-600 text-white text-[10px] font-bold focus-visible:outline-violet-400 relative min-w-[2rem] px-1`}
        title={chipTitle}
        aria-label="System memory monitor"
        aria-expanded={open}
        data-testid="memory-monitor-chip"
      >
        {snapshot ? `${Math.round(usedPercent)}%` : 'RAM'}
        <span
          className={`absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full border border-slack-bgHover ${memoryPressureClass(level)}`}
          aria-hidden
        />
      </button>
      {popover}
    </>
  );
}
