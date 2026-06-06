import { useCallback, useEffect, useRef, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

interface QuickOpenModalProps {
  isOpen: boolean;
  workspaceId: string | undefined;
  onClose: () => void;
  onOpenPath: (path: string) => void | Promise<void>;
}

export function QuickOpenModal({ isOpen, workspaceId, onClose, onOpenPath }: QuickOpenModalProps) {
  const [query, setQuery] = useState('');
  const [paths, setPaths] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const search = useCallback(
    async (q: string) => {
      if (!workspaceId) {
        setPaths([]);
        return;
      }
      setLoading(true);
      try {
        const api = new ChatAPI(getHubBaseURL());
        const result = await api.searchWorkspaceFiles(workspaceId, q);
        setPaths(result);
        setHighlight(0);
      } catch {
        setPaths([]);
      } finally {
        setLoading(false);
      }
    },
    [workspaceId]
  );

  useEffect(() => {
    if (!isOpen) return;
    setQuery('');
    void search('');
    const t = setTimeout(() => inputRef.current?.focus(), 50);
    return () => clearTimeout(t);
  }, [isOpen, search]);

  useEffect(() => {
    if (!isOpen) return;
    const t = setTimeout(() => void search(query), 200);
    return () => clearTimeout(t);
  }, [query, isOpen, search]);

  useEffect(() => {
    if (!isOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setHighlight((h) => Math.min(h + 1, Math.max(0, paths.length - 1)));
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setHighlight((h) => Math.max(h - 1, 0));
      }
      if (e.key === 'Enter' && paths[highlight]) {
        e.preventDefault();
        void onOpenPath(paths[highlight]);
        onClose();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [isOpen, paths, highlight, onClose, onOpenPath]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh] bg-black/50"
      role="dialog"
      aria-label="Quick open file"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="w-full max-w-lg rounded-lg border border-slack-border bg-slack-bg shadow-xl overflow-hidden">
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Go to file…"
          className="w-full px-4 py-3 text-sm bg-slack-bg text-slack-text border-b border-slack-border outline-none"
        />
        <ul className="max-h-64 overflow-y-auto text-sm">
          {loading && paths.length === 0 && (
            <li className="px-4 py-2 text-slack-textMuted">Searching…</li>
          )}
          {!loading && paths.length === 0 && (
            <li className="px-4 py-2 text-slack-textMuted">No files</li>
          )}
          {paths.map((p, i) => (
            <li key={p}>
              <button
                type="button"
                className={`w-full text-left px-4 py-1.5 font-mono text-xs ${
                  i === highlight ? 'bg-slack-accent/30 text-slack-text' : 'text-slack-textMuted hover:bg-slack-bgHover'
                }`}
                onMouseEnter={() => setHighlight(i)}
                onClick={() => {
                  void onOpenPath(p);
                  onClose();
                }}
              >
                {p}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
