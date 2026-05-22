import { useCallback, useEffect, useRef, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useEditorStore } from '../stores/editorStore';
import { useFileChangeStore } from '../stores/fileChangeStore';

interface FastEditModalProps {
  isOpen: boolean;
  workspaceId: string | undefined;
  onClose: () => void;
}

export function FastEditModal({ isOpen, workspaceId, onClose }: FastEditModalProps) {
  const [instruction, setInstruction] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const activeTab = useEditorStore((s) => {
    const id = s.activeTabId;
    return id ? (s.tabs.find((t) => t.id === id) ?? null) : null;
  });
  const selection = useEditorStore((s) => s.activeSelection);

  useEffect(() => {
    if (!isOpen) return;
    setInstruction('');
    setError(null);
    const t = setTimeout(() => inputRef.current?.focus(), 50);
    return () => clearTimeout(t);
  }, [isOpen]);

  const run = useCallback(async () => {
    if (!workspaceId || !instruction.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const api = new ChatAPI(getHubBaseURL());
      const agentType =
        activeTab?.path?.endsWith('.go') || activeTab?.language === 'go'
          ? 'backend'
          : activeTab?.path?.match(/\.(tsx?|jsx?)$/)
            ? 'frontend'
            : 'backend';
      await api.devFastEdit({
        workspaceId,
        path: activeTab?.path,
        instruction: instruction.trim(),
        selection: selection?.text,
        agentType,
      });
      await useFileChangeStore.getState().fetchPendingChanges();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [workspaceId, instruction, activeTab, selection, onClose]);

  useEffect(() => {
    if (!isOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[20vh] bg-black/50"
      role="dialog"
      aria-label="Fast edit"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="w-full max-w-lg rounded-lg border border-slack-border bg-slack-bg shadow-xl p-4">
        <p className="text-xs text-slack-textMuted mb-2">
          Fast edit {activeTab?.path ? `· ${activeTab.path}` : ''}
          {selection ? ` · lines ${selection.startLine}–${selection.endLine}` : ''}
        </p>
        <input
          ref={inputRef}
          type="text"
          value={instruction}
          onChange={(e) => setInstruction(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              void run();
            }
          }}
          placeholder="Describe the change…"
          disabled={loading}
          className="w-full px-3 py-2 text-sm bg-slack-bg text-slack-text border border-slack-border rounded outline-none"
        />
        {error && <p className="text-xs text-red-400 mt-2">{error}</p>}
        <div className="flex justify-end gap-2 mt-3">
          <button
            type="button"
            className="px-3 py-1 text-sm text-slack-textMuted"
            onClick={onClose}
            disabled={loading}
          >
            Cancel
          </button>
          <button
            type="button"
            className="px-3 py-1 text-sm bg-slack-accent text-white rounded disabled:opacity-50"
            onClick={() => void run()}
            disabled={loading || !instruction.trim()}
          >
            {loading ? 'Running…' : 'Apply'}
          </button>
        </div>
      </div>
    </div>
  );
}
