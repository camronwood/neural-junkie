import { useEffect, useMemo, useRef, useState } from 'react';

export interface TimPickerItem {
  id: string;
  label: string;
}

interface SearchableTimPickerProps {
  items: TimPickerItem[];
  value: string;
  onChange: (id: string) => void;
  searchPlaceholder?: string;
  emptyLabel?: string;
}

function matchesQuery(item: TimPickerItem, query: string): boolean {
  if (!query) return true;
  const hay = `${item.label} ${item.id}`.toLowerCase();
  return hay.includes(query);
}

export function SearchableTimPicker({
  items,
  value,
  onChange,
  searchPlaceholder = 'Search by name or id…',
  emptyLabel = 'Nothing listed',
}: SearchableTimPickerProps) {
  const [query, setQuery] = useState('');
  const [highlightIndex, setHighlightIndex] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLUListElement>(null);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return items.filter((item) => matchesQuery(item, q));
  }, [items, query]);

  useEffect(() => {
    setHighlightIndex(0);
  }, [query, items.length]);

  useEffect(() => {
    if (filtered.length === 0) return;
    if (!filtered.some((i) => i.id === value)) {
      onChange(filtered[0].id);
    }
  }, [filtered, value, onChange]);

  useEffect(() => {
    const el = listRef.current?.children[highlightIndex] as HTMLElement | undefined;
    el?.scrollIntoView({ block: 'nearest' });
  }, [highlightIndex, filtered.length]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (filtered.length === 0) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setHighlightIndex((i) => (i < filtered.length - 1 ? i + 1 : 0));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlightIndex((i) => (i > 0 ? i - 1 : filtered.length - 1));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const pick = filtered[highlightIndex];
      if (pick) onChange(pick.id);
    }
  };

  const selected = items.find((i) => i.id === value);

  return (
    <div className="space-y-1">
      <input
        ref={searchRef}
        type="search"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={searchPlaceholder}
        className="w-full px-2 py-1.5 rounded border border-slack-border bg-slack-bg text-sm text-slack-text placeholder:text-slack-textMuted focus:outline-none focus:border-indigo-500"
        aria-label={searchPlaceholder}
        autoComplete="off"
      />
      <ul
        ref={listRef}
        role="listbox"
        aria-label="TIM items"
        className="max-h-52 overflow-y-auto rounded border border-slack-border bg-slack-bg divide-y divide-slack-border/60"
      >
        {filtered.length === 0 ? (
          <li className="px-3 py-4 text-center text-xs text-slack-textMuted" role="presentation">
            {items.length === 0 ? emptyLabel : 'No matches'}
          </li>
        ) : (
          filtered.map((item, index) => {
            const active = item.id === value;
            const highlighted = index === highlightIndex;
            return (
              <li key={item.id} role="presentation">
                <button
                  type="button"
                  role="option"
                  aria-selected={active}
                  onMouseEnter={() => setHighlightIndex(index)}
                  onClick={() => onChange(item.id)}
                  className={`w-full text-left px-3 py-2 text-sm transition-colors ${
                    active
                      ? 'bg-indigo-600/25 text-indigo-100'
                      : highlighted
                        ? 'bg-slack-bgHover text-slack-text'
                        : 'text-slack-text hover:bg-slack-bgHover'
                  }`}
                >
                  <div className="truncate font-medium">{item.label}</div>
                  <div className="truncate font-mono text-[10px] text-slack-textMuted">{item.id}</div>
                </button>
              </li>
            );
          })
        )}
      </ul>
      {selected && (
        <p className="text-[10px] text-slack-textMuted font-mono truncate" title={selected.id}>
          Selected: {selected.label} · {selected.id}
        </p>
      )}
    </div>
  );
}
