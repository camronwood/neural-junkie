import { useMemo, useState } from 'react';
import type { HubDataAccessOption } from '../utils/hubDataAccess';
import { HUB_DATA_ROOT_LABEL } from '../utils/hubDataAccess';

export interface HubDataAccessModalProps {
  options: HubDataAccessOption[];
  isLoading?: boolean;
  error?: string | null;
  onCancel: () => void;
  onConfirm: (selected: HubDataAccessOption[]) => void;
}

export function HubDataAccessModal({
  options,
  isLoading = false,
  error = null,
  onCancel,
  onConfirm,
}: HubDataAccessModalProps) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => {
    const s = new Set<string>();
    for (const o of options) {
      if (o.defaultSelected) s.add(o.id);
    }
    return s;
  });

  const selected = useMemo(
    () => options.filter((o) => selectedIds.has(o.id)),
    [options, selectedIds]
  );

  const toggle = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  return (
    <div className="fixed inset-0 z-[300] flex items-center justify-center p-4" role="presentation">
      <div className="absolute inset-0 bg-black/60" onClick={onCancel} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="hub-data-access-title"
        className="relative z-10 w-full max-w-md rounded-xl border border-slack-border bg-slack-bg shadow-2xl"
      >
        <div className="px-5 py-4 border-b border-slack-border">
          <h2 id="hub-data-access-title" className="text-base font-semibold text-slack-text">
            Allow hub data access?
          </h2>
          <p className="mt-2 text-sm text-slack-textMuted leading-relaxed">
            Agents normally only see your workspace and chat. Select what from{' '}
            <span className="font-mono text-slack-text">{HUB_DATA_ROOT_LABEL}</span> may be read
            for <strong className="font-normal text-slack-text">this message only</strong>.
          </p>
        </div>

        <div className="max-h-[50vh] overflow-y-auto px-5 py-3 space-y-2">
          {options.map((opt) => (
            <label
              key={opt.id}
              className="flex gap-3 items-start p-3 rounded-lg border border-slack-border bg-slack-bgHover/40 cursor-pointer hover:bg-slack-bgHover"
            >
              <input
                type="checkbox"
                className="mt-1"
                checked={selectedIds.has(opt.id)}
                onChange={() => toggle(opt.id)}
                disabled={isLoading}
              />
              <span className="min-w-0">
                <span className="block text-sm font-medium text-slack-text">
                  {opt.kind === 'directory' ? '📁 ' : '📄 '}
                  {opt.label}
                </span>
                <span className="block text-xs text-slack-textMuted mt-1">{opt.description}</span>
              </span>
            </label>
          ))}
        </div>

        {error && (
          <p className="px-5 pb-2 text-sm text-red-400">{error}</p>
        )}

        <div className="px-5 py-4 border-t border-slack-border flex justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            disabled={isLoading}
            className="px-3 py-1.5 text-sm text-slack-textMuted hover:text-slack-text rounded"
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={isLoading || selected.length === 0}
            onClick={() => onConfirm(selected)}
            className="px-4 py-1.5 text-sm bg-slack-accent hover:bg-slack-accentHover text-white rounded disabled:opacity-40"
          >
            {isLoading ? 'Loading…' : 'Allow & send'}
          </button>
        </div>
      </div>
    </div>
  );
}
