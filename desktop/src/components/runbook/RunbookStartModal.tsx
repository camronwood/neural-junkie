import { useMemo, useState } from 'react';
import type { RunInputSpec } from '../../types/protocol';

interface RunbookStartModalProps {
  isOpen: boolean;
  inputs: RunInputSpec[];
  busy?: boolean;
  onClose: () => void;
  onStart: (values: Record<string, string>) => void;
}

export function RunbookStartModal({ isOpen, inputs, busy = false, onClose, onStart }: RunbookStartModalProps) {
  const initial = useMemo(() => {
    const m: Record<string, string> = {};
    for (const spec of inputs) {
      if (spec.default) m[spec.key] = spec.default;
    }
    return m;
  }, [inputs]);
  const [values, setValues] = useState<Record<string, string>>(initial);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/55 p-4" onClick={onClose}>
      <div className="bg-slack-bg border border-slack-border rounded-lg p-5 w-full max-w-md" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-lg font-bold text-slack-text mb-3">Run inputs</h3>
        {inputs.length === 0 ? (
          <p className="text-sm text-slack-textMuted mb-4">No inputs required for this runbook.</p>
        ) : (
          <div className="space-y-3 mb-4">
            {inputs.map((spec) => (
              <label key={spec.key} className="block text-xs text-slack-textMuted">
                {spec.label || spec.key}
                {spec.required ? ' *' : ''}
                <input
                  className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text"
                  value={values[spec.key] ?? ''}
                  onChange={(e) => setValues((v) => ({ ...v, [spec.key]: e.target.value }))}
                />
              </label>
            ))}
          </div>
        )}
        <div className="flex justify-end gap-2">
          <button type="button" className="text-sm text-slack-textMuted" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button
            type="button"
            className="px-3 py-1.5 rounded bg-[#8b5cf6] text-white text-sm disabled:opacity-50"
            disabled={busy}
            onClick={() => onStart(values)}
          >
            {busy ? 'Starting…' : 'Start execution'}
          </button>
        </div>
      </div>
    </div>
  );
}
