import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { Workspace } from '../stores/fileExplorerStore';
import { filterWorkspacesForSwitcher } from '../utils/workspaceOrder';

export interface WorkspaceSwitcherModalProps {
  isOpen: boolean;
  onClose: () => void;
  workspaces: Workspace[];
  activeWorkspaceId: string | null;
  onSelect: (id: string) => void;
}

export function WorkspaceSwitcherModal({
  isOpen,
  onClose,
  workspaces,
  activeWorkspaceId,
  onSelect,
}: WorkspaceSwitcherModalProps) {
  const [query, setQuery] = useState('');
  const [highlightIndex, setHighlightIndex] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const highlightIndexRef = useRef(0);
  const rowRefs = useRef<(HTMLButtonElement | null)[]>([]);

  const filtered = useMemo(
    () => filterWorkspacesForSwitcher(workspaces, query),
    [workspaces, query]
  );

  useEffect(() => {
    highlightIndexRef.current = highlightIndex;
  }, [highlightIndex]);

  useEffect(() => {
    if (!isOpen) return;
    setQuery('');
    setHighlightIndex(0);
    highlightIndexRef.current = 0;
    const t = window.setTimeout(() => searchRef.current?.focus(), 0);
    return () => window.clearTimeout(t);
  }, [isOpen]);

  useEffect(() => {
    setHighlightIndex(0);
    highlightIndexRef.current = 0;
    rowRefs.current = [];
  }, [query]);

  useEffect(() => {
    setHighlightIndex((prev) =>
      filtered.length === 0 ? 0 : Math.min(prev, filtered.length - 1)
    );
  }, [filtered.length]);

  useEffect(() => {
    if (!isOpen || filtered.length === 0) return;
    rowRefs.current[highlightIndex]?.scrollIntoView({ block: 'nearest' });
  }, [highlightIndex, filtered.length, isOpen]);

  const pick = useCallback(
    (id: string) => {
      onSelect(id);
      onClose();
    },
    [onSelect, onClose]
  );

  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose();
        return;
      }
      if (filtered.length === 0) return;

      if (event.key === 'ArrowDown') {
        event.preventDefault();
        setHighlightIndex((prev) => (prev < filtered.length - 1 ? prev + 1 : 0));
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault();
        setHighlightIndex((prev) => (prev > 0 ? prev - 1 : filtered.length - 1));
      }
      if (event.key === 'Enter') {
        event.preventDefault();
        const ws = filtered[highlightIndexRef.current];
        if (ws) pick(ws.id);
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, onClose, filtered, pick]);

  if (!isOpen) return null;

  const listboxId = 'workspace-switcher-listbox';

  return (
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[12vh] bg-black/50"
      role="dialog"
      aria-modal="true"
      aria-label="Switch workspace"
      data-workspace-switcher-modal
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="bg-slack-bg border border-slack-border rounded shadow-xl w-full max-w-md mx-4 overflow-hidden">
        <div className="px-4 py-3 border-b border-slack-border">
          <h3 className="text-sm font-bold text-slack-text mb-2">Switch workspace</h3>
          <input
            ref={searchRef}
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter by name, path, or branch..."
            className="w-full px-3 py-2 text-sm bg-slack-bg border border-slack-border rounded text-slack-text placeholder:text-slack-textMuted focus:outline-none focus:border-slack-accent"
            aria-label="Filter workspaces"
            aria-controls={listboxId}
            aria-autocomplete="list"
            autoComplete="off"
          />
          <p className="mt-2 text-xs text-slack-textMuted">
            Up/Down to highlight, Enter to switch, Esc to close
          </p>
        </div>

        <ul
          id={listboxId}
          className="max-h-64 overflow-y-auto py-1"
          role="listbox"
          aria-label="Workspaces"
        >
          {filtered.length === 0 ? (
            <li className="px-4 py-6 text-center text-sm text-slack-textMuted" role="presentation">
              {workspaces.length === 0
                ? 'No workspaces yet. Use + in the Files header.'
                : 'No workspaces match your filter.'}
            </li>
          ) : (
            filtered.map((ws, index) => {
              const highlighted = index === highlightIndex;
              const isActive = ws.id === activeWorkspaceId;
              return (
                <li key={ws.id} role="presentation">
                  <button
                    ref={(el) => {
                      rowRefs.current[index] = el;
                    }}
                    type="button"
                    role="option"
                    aria-selected={highlighted}
                    onClick={() => pick(ws.id)}
                    onMouseEnter={() => setHighlightIndex(index)}
                    className={`w-full text-left px-4 py-2 transition-colors ${
                      highlighted ? 'bg-slack-accent/20' : 'hover:bg-slack-bgHover'
                    }`}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span
                        className={`text-sm font-medium truncate ${
                          isActive ? 'text-slack-accent' : 'text-slack-text'
                        }`}
                      >
                        {ws.name}
                        {isActive ? ' - active' : ''}
                      </span>
                      {ws.is_git_repo && ws.git_branch && (
                        <span className="text-xs text-slack-textMuted flex-shrink-0 font-mono">
                          {ws.git_branch}
                        </span>
                      )}
                    </div>
                    <div className="text-xs text-slack-textMuted truncate font-mono" title={ws.path}>
                      {ws.path}
                    </div>
                  </button>
                </li>
              );
            })
          )}
        </ul>
      </div>
    </div>
  );
}
