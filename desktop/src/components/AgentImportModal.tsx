import { useCallback, useRef, useState } from 'react';
import { isTauriRuntime } from '../utils/promptAttachments';
import type { ChatAPI } from '../api/chatAPI';

interface AgentImportModalProps {
  isOpen: boolean;
  api: ChatAPI;
  onClose: () => void;
  onImported: (message: string) => void;
}

/**
 * Share Agent import — file picker + optional path remap + hydrate-from-bundle.
 * Uses POST /api/import (hub needs a filesystem path; Tauri open() supplies one).
 */
export function AgentImportModal({ isOpen, api, onClose, onImported }: AgentImportModalProps) {
  const [filePath, setFilePath] = useState('');
  const [fileLabel, setFileLabel] = useState<string | null>(null);
  const [hydrate, setHydrate] = useState(true);
  const [repositoryPath, setRepositoryPath] = useState('');
  const [busy, setBusy] = useState(false);
  const [localError, setLocalError] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const reset = useCallback(() => {
    setFilePath('');
    setFileLabel(null);
    setHydrate(true);
    setRepositoryPath('');
    setLocalError('');
    setBusy(false);
  }, []);

  const handleClose = () => {
    if (busy) return;
    reset();
    onClose();
  };

  const handleChooseBundleTauri = async () => {
    try {
      const { open } = await import('@tauri-apps/plugin-dialog');
      const selected = await open({
        multiple: false,
        title: 'Import Share Agent bundle',
        filters: [{ name: 'Share Agent', extensions: ['json'] }],
      });
      if (!selected || typeof selected !== 'string') return;
      setFilePath(selected);
      setFileLabel(selected.split(/[/\\]/).pop() ?? selected);
      setLocalError('');
    } catch (e) {
      setLocalError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleChooseRepoTauri = async () => {
    try {
      const { open } = await import('@tauri-apps/plugin-dialog');
      const selected = await open({
        directory: true,
        multiple: false,
        title: 'Remap repository path',
      });
      if (!selected || typeof selected !== 'string') return;
      setRepositoryPath(selected);
    } catch (e) {
      setLocalError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleChooseBundle = () => {
    setLocalError('');
    if (isTauriRuntime()) {
      void handleChooseBundleTauri();
    } else {
      setLocalError(
        'Browser import needs a hub-readable absolute path. Use the desktop app, or paste a path the hub can open.'
      );
    }
  };

  const handleImport = async () => {
    setLocalError('');
    const path = filePath.trim();
    if (!path) {
      setLocalError('Choose a Share Agent bundle (.json) first.');
      return;
    }
    setBusy(true);
    try {
      const result = await api.importAgentBundle({
        filePath: path,
        hydrate,
        repositoryPath: repositoryPath.trim() || undefined,
      });
      const base = result.message || (result.name ? `Imported ${result.name}` : 'Agent imported');
      const msg =
        base +
        (result.lora_train_suggestion
          ? ' — LoRA metadata present; pull/compose the adapter from Settings → Specialist tuning if needed.'
          : '');
      reset();
      onImported(msg);
      onClose();
    } catch (e) {
      setLocalError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
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
        className="bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-lg flex flex-col"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="agent-import-title"
      >
        <div className="px-5 py-4 border-b border-slack-border">
          <h2 id="agent-import-title" className="text-lg font-bold text-slack-text">
            Import agent
          </h2>
          <p className="text-xs text-slack-textMuted mt-1">
            Load a Share Agent bundle from a friend or another machine. Hydrate keeps knowledge from the
            file; optionally remap the repo path for re-index.
          </p>
        </div>

        <div className="px-5 py-4 space-y-4">
          {localError ? (
            <p className="text-xs text-red-400" role="alert">
              {localError}
            </p>
          ) : null}

          <div>
            <label className="block text-xs font-medium text-slack-textMuted mb-1">Bundle file</label>
            <div className="flex gap-2">
              <button
                type="button"
                className="px-3 py-1.5 text-sm rounded border border-slack-border hover:bg-slack-bgHover"
                onClick={handleChooseBundle}
                disabled={busy}
              >
                Choose file…
              </button>
              <span className="text-sm text-slack-text truncate self-center">
                {fileLabel || 'No file selected'}
              </span>
            </div>
            {!isTauriRuntime() ? (
              <input
                className="mt-2 w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
                placeholder="/absolute/path/to/agent-share-bundle.json"
                value={filePath}
                onChange={(e) => {
                  setFilePath(e.target.value);
                  setFileLabel(e.target.value.split(/[/\\]/).pop() || e.target.value);
                }}
                disabled={busy}
              />
            ) : null}
            <input
              ref={fileInputRef}
              type="file"
              accept="application/json,.json"
              className="hidden"
              onChange={() => {
                /* path-only import; browser File has no hub path */
              }}
            />
          </div>

          <label className="flex items-start gap-2 text-sm text-slack-text cursor-pointer">
            <input
              type="checkbox"
              className="mt-1"
              checked={hydrate}
              onChange={(e) => setHydrate(e.target.checked)}
              disabled={busy}
            />
            <span>
              <span className="font-medium">Hydrate from bundle</span>
              <span className="block text-xs text-slack-textMuted">
                Rebuild knowledge from embedded resources (recommended when the original repo path is gone).
              </span>
            </span>
          </label>

          <div>
            <label className="block text-xs font-medium text-slack-textMuted mb-1">
              Remap repository path (optional)
            </label>
            <div className="flex gap-2">
              {isTauriRuntime() ? (
                <button
                  type="button"
                  className="px-3 py-1.5 text-sm rounded border border-slack-border hover:bg-slack-bgHover shrink-0"
                  onClick={() => void handleChooseRepoTauri()}
                  disabled={busy}
                >
                  Choose folder…
                </button>
              ) : null}
              <input
                className="flex-1 px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
                placeholder="/path/to/local/checkout"
                value={repositoryPath}
                onChange={(e) => setRepositoryPath(e.target.value)}
                disabled={busy}
              />
            </div>
          </div>
        </div>

        <div className="px-5 py-3 border-t border-slack-border flex justify-end gap-2">
          <button
            type="button"
            className="text-sm text-slack-textMuted px-3 py-1.5"
            onClick={handleClose}
            disabled={busy}
          >
            Cancel
          </button>
          <button
            type="button"
            className="px-3 py-1.5 rounded bg-slack-accent text-white text-sm disabled:opacity-50"
            onClick={() => void handleImport()}
            disabled={busy || !filePath.trim()}
          >
            {busy ? 'Importing…' : 'Import'}
          </button>
        </div>
      </div>
    </div>
  );
}
