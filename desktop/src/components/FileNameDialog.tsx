import { useEffect, useRef, useState } from 'react';

interface FileNameDialogProps {
  title: string;
  label: string;
  initialValue?: string;
  confirmLabel?: string;
  onConfirm: (value: string) => void;
  onCancel: () => void;
}

/** Modal prompt for rename / new file / new folder (replaces window.prompt in Tauri). */
export function FileNameDialog({
  title,
  label,
  initialValue = '',
  confirmLabel = 'OK',
  onConfirm,
  onCancel,
}: FileNameDialogProps) {
  const [value, setValue] = useState(initialValue);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  const submit = () => {
    const trimmed = value.trim();
    if (!trimmed) return;
    onConfirm(trimmed);
  };

  return (
    <>
      <div className="fixed inset-0 z-[260] bg-black/50" onClick={onCancel} aria-hidden />
      <div
        className="fixed z-[261] top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-slack-bg border border-slack-border rounded-lg shadow-xl p-5 min-w-[320px]"
        role="dialog"
        aria-labelledby="file-name-dialog-title"
      >
        <h3 id="file-name-dialog-title" className="text-sm font-semibold text-slack-text mb-3">
          {title}
        </h3>
        <label className="block text-xs text-slack-textMuted mb-1">{label}</label>
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') submit();
            if (e.key === 'Escape') onCancel();
          }}
          className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text focus:outline-none focus:border-slack-accent font-mono"
        />
        <div className="flex justify-end gap-2 mt-4">
          <button
            type="button"
            onClick={onCancel}
            className="px-3 py-1.5 text-xs rounded bg-slack-bgHover text-slack-text hover:bg-slack-border"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={submit}
            disabled={!value.trim()}
            className="px-3 py-1.5 text-xs rounded bg-slack-accent text-white hover:opacity-90 disabled:opacity-50"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </>
  );
}
