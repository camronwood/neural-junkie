import { useCallback, useRef, useState, type ReactNode } from 'react';
import { isTauriRuntime } from '../utils/promptAttachments';

const PLAN_PLACEHOLDER = `## Plan
- Task 1: @RustExpert - Build feature
- Task 2: @SecurityExpert - Review
  - depends: 1`;

type ImportTab = 'paste' | 'file';

interface RunbookImportModalProps {
  isOpen: boolean;
  busy?: boolean;
  onClose: () => void;
  onImport: (markdown: string) => void | Promise<void>;
}

export function RunbookImportModal({ isOpen, busy = false, onClose, onImport }: RunbookImportModalProps) {
  const [tab, setTab] = useState<ImportTab>('paste');
  const [markdown, setMarkdown] = useState('');
  const [fileLabel, setFileLabel] = useState<string | null>(null);
  const [localError, setLocalError] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const reset = useCallback(() => {
    setTab('paste');
    setMarkdown('');
    setFileLabel(null);
    setLocalError('');
  }, []);

  const handleClose = () => {
    if (busy) return;
    reset();
    onClose();
  };

  const loadFromBrowserFile = async (file: File) => {
    setLocalError('');
    if (file.size > 512_000) {
      setLocalError('File is too large (max 512 KB).');
      return;
    }
    try {
      const text = await file.text();
      setMarkdown(text);
      setFileLabel(file.name);
      setTab('paste');
    } catch (e) {
      setLocalError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleChooseFileTauri = async () => {
    try {
      const { open } = await import('@tauri-apps/api/dialog');
      const { readTextFile } = await import('@tauri-apps/api/fs');
      const selected = await open({
        multiple: false,
        title: 'Import runbook plan',
        filters: [{ name: 'Markdown', extensions: ['md', 'markdown', 'txt'] }],
      });
      if (!selected || typeof selected !== 'string') return;
      const text = await readTextFile(selected);
      const name = selected.split(/[/\\]/).pop() ?? selected;
      setMarkdown(text);
      setFileLabel(name);
      setTab('paste');
    } catch (e) {
      setLocalError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleChooseFile = () => {
    setLocalError('');
    if (isTauriRuntime()) {
      void handleChooseFileTauri();
    } else {
      fileInputRef.current?.click();
    }
  };

  const handleImport = async () => {
    setLocalError('');
    const body = markdown.trim();
    if (!body) {
      setLocalError('Add plan markdown or choose a file first.');
      return;
    }
    try {
      await onImport(body);
      reset();
    } catch (e) {
      setLocalError(e instanceof Error ? e.message : String(e));
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/55 p-4"
      onClick={handleClose}
      role="presentation"
    >
      <div
        className="bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-lg flex flex-col max-h-[min(90vh,640px)]"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="runbook-import-title"
      >
        <div className="px-5 py-4 border-b border-slack-border shrink-0">
          <h2 id="runbook-import-title" className="text-lg font-bold text-slack-text">
            Import from markdown
          </h2>
          <p className="text-xs text-slack-textMuted mt-1">
            Paste a collaborate-style plan or load a <code className="text-slack-text">.md</code> file. Lines with{' '}
            <code className="text-slack-text">depends: 1, 2</code> become task dependencies.
          </p>
        </div>

        <div className="px-5 pt-3 flex gap-2 shrink-0">
          <TabButton active={tab === 'paste'} onClick={() => setTab('paste')} disabled={busy}>
            Paste text
          </TabButton>
          <TabButton active={tab === 'file'} onClick={() => setTab('file')} disabled={busy}>
            Choose file
          </TabButton>
        </div>

        <div className="px-5 py-3 flex-1 min-h-0 overflow-y-auto">
          {localError ? (
            <p className="text-xs text-red-400 mb-2" role="alert">
              {localError}
            </p>
          ) : null}

          {tab === 'paste' ? (
            <div>
              {fileLabel ? (
                <p className="text-xs text-slack-textMuted mb-2">
                  Loaded from <span className="text-slack-text font-medium">{fileLabel}</span>
                </p>
              ) : null}
              <label className="block text-xs font-medium text-slack-textMuted mb-1">Plan markdown</label>
              <textarea
                value={markdown}
                onChange={(e) => setMarkdown(e.target.value)}
                placeholder={PLAN_PLACEHOLDER}
                rows={12}
                disabled={busy}
                className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted font-mono focus:outline-none focus:ring-1 focus:ring-slack-accent resize-y min-h-[200px]"
              />
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-8 gap-4 border border-dashed border-slack-border rounded-lg bg-slack-bgHover">
              <p className="text-sm text-slack-textMuted text-center px-4">
                Select a markdown or text file from your machine.
              </p>
              <button
                type="button"
                onClick={handleChooseFile}
                disabled={busy}
                className="px-4 py-2 rounded bg-slack-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50"
              >
                Choose file…
              </button>
              {!isTauriRuntime() ? (
                <p className="text-xs text-slack-textMuted">Uses your browser file picker in web-only dev mode.</p>
              ) : null}
              <input
                ref={fileInputRef}
                type="file"
                accept=".md,.markdown,.txt,text/markdown,text/plain"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  e.target.value = '';
                  if (file) void loadFromBrowserFile(file);
                }}
              />
            </div>
          )}
        </div>

        <div className="px-5 py-4 border-t border-slack-border flex justify-end gap-2 shrink-0">
          <button
            type="button"
            onClick={handleClose}
            disabled={busy}
            className="px-4 py-2 rounded text-sm text-slack-textMuted hover:text-slack-text disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void handleImport()}
            disabled={busy || !markdown.trim()}
            className="px-4 py-2 rounded bg-[#8b5cf6] text-white text-sm font-medium hover:opacity-90 disabled:opacity-50"
          >
            {busy ? 'Parsing…' : 'Import tasks'}
          </button>
        </div>
      </div>
    </div>
  );
}

function TabButton({
  active,
  onClick,
  disabled,
  children,
}: {
  active: boolean;
  onClick: () => void;
  disabled?: boolean;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`px-3 py-1.5 rounded text-xs font-medium transition-colors ${
        active
          ? 'bg-slack-accent/20 text-slack-accent border border-slack-accent/40'
          : 'bg-slack-bgHover text-slack-textMuted border border-slack-border hover:text-slack-text'
      } disabled:opacity-50`}
    >
      {children}
    </button>
  );
}
