import { useEffect, useRef } from 'react';

interface ChatFindBarProps {
  query: string;
  onQueryChange: (query: string) => void;
  onClose: () => void;
}

export function ChatFindBar({ query, onQueryChange, onClose }: ChatFindBarProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  return (
    <div
      className="sticky top-0 z-10 flex items-center gap-2 border-b border-slack-border bg-slack-bg px-3 py-2"
      data-testid="chat-find-bar"
    >
      <input
        ref={inputRef}
        type="search"
        value={query}
        onChange={(e) => onQueryChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.stopPropagation();
            onClose();
          }
        }}
        placeholder="Search messages…"
        aria-label="Search messages in this channel"
        className="flex-1 rounded-md border border-slack-border bg-slack-bgHover px-2 py-1.5 text-sm text-slack-text placeholder:text-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
      />
      <button
        type="button"
        onClick={onClose}
        className="shrink-0 rounded px-2 py-1 text-xs text-slack-textMuted hover:bg-slack-bgHover hover:text-slack-text"
        aria-label="Close find bar"
      >
        Esc
      </button>
    </div>
  );
}
