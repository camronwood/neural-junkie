import { useCallback, useEffect, useRef, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

export interface WorkspaceSymbol {
  name: string;
  path: string;
  line: number;
  kind: string;
  language: string;
}

interface SymbolModalProps {
  isOpen: boolean;
  workspaceId: string | undefined;
  onClose: () => void;
  onOpenSymbol: (path: string, line: number) => void | Promise<void>;
}

export function SymbolModal({ isOpen, workspaceId, onClose, onOpenSymbol }: SymbolModalProps) {
  const [query, setQuery] = useState('');
  const [symbols, setSymbols] = useState<WorkspaceSymbol[]>([]);
  const [loading, setLoading] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const search = useCallback(
    async (q: string) => {
      if (!workspaceId) {
        setSymbols([]);
        return;
      }
      setLoading(true);
      try {
        const api = new ChatAPI(getHubBaseURL());
        const result = await api.searchWorkspaceSymbols(workspaceId, q);
        setSymbols(result);
        setHighlight(0);
      } catch {
        setSymbols([]);
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
        setHighlight((h) => Math.min(h + 1, Math.max(0, symbols.length - 1)));
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setHighlight((h) => Math.max(h - 1, 0));
      }
      if (e.key === 'Enter' && symbols[highlight]) {
        e.preventDefault();
        void onOpenSymbol(symbols[highlight].path, symbols[highlight].line);
        onClose();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [isOpen, symbols, highlight, onClose, onOpenSymbol]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh] bg-black/50"
      role="dialog"
      aria-label="Go to symbol"
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
          placeholder="Go to symbol…"
          className="w-full px-4 py-3 text-sm bg-slack-bg text-slack-text border-b border-slack-border outline-none placeholder:text-slack-textMuted"
        />
        <ul className="max-h-64 overflow-y-auto text-sm">
          {loading && symbols.length === 0 && (
            <li className="px-4 py-2 text-slack-textMuted">Searching…</li>
          )}
          {symbols.map((s, i) => (
            <li key={`${s.path}:${s.line}:${s.name}`}>
              <button
                type="button"
                className={`w-full text-left px-4 py-2 ${
                  i === highlight ? 'bg-slack-accent/30 text-slack-text' : 'text-slack-textMuted hover:bg-slack-bgHover'
                }`}
                onMouseEnter={() => setHighlight(i)}
                onClick={() => {
                  void onOpenSymbol(s.path, s.line);
                  onClose();
                }}
              >
                <span className="text-slack-text font-medium">{s.name}</span>
                <span className="text-xs text-slack-textMuted ml-2">
                  {s.kind} · {s.path}:{s.line}
                </span>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
