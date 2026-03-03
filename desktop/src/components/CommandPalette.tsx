import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { CommandForm } from './CommandForm';
import type { CommandDefinition, AgentInfo } from '../types/protocol';

interface CommandPaletteProps {
  commands: CommandDefinition[];
  agents: AgentInfo[];
  isOpen: boolean;
  initialFilter?: string;
  onClose: () => void;
  onExecute: (commandString: string) => void;
}

export function CommandPalette({
  commands,
  agents,
  isOpen,
  initialFilter = '',
  onClose,
  onExecute,
}: CommandPaletteProps) {
  const [query, setQuery] = useState(initialFilter);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [activeCommand, setActiveCommand] = useState<CommandDefinition | null>(null);

  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Reset state when opened/closed
  useEffect(() => {
    if (isOpen) {
      setQuery(initialFilter);
      setSelectedIndex(0);
      setActiveCommand(null);
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [isOpen, initialFilter]);

  // Lock body scroll while open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
      return () => { document.body.style.overflow = ''; };
    }
  }, [isOpen]);

  // Filter commands
  const filtered = useMemo(() => {
    if (!query) return commands;
    const lower = query.toLowerCase().replace(/^\//, '');
    return commands.filter(
      c =>
        c.name.toLowerCase().includes(lower) ||
        c.description.toLowerCase().includes(lower) ||
        c.category.toLowerCase().includes(lower)
    );
  }, [commands, query]);

  // Group by category
  const grouped = useMemo(() => {
    const map = new Map<string, CommandDefinition[]>();
    for (const cmd of filtered) {
      const list = map.get(cmd.category) ?? [];
      list.push(cmd);
      map.set(cmd.category, list);
    }
    return map;
  }, [filtered]);

  // Flat list for keyboard navigation
  const flatList = useMemo(() => {
    const result: CommandDefinition[] = [];
    for (const cmds of grouped.values()) {
      result.push(...cmds);
    }
    return result;
  }, [grouped]);

  // Clamp selected index
  useEffect(() => {
    setSelectedIndex(prev => Math.min(prev, Math.max(0, flatList.length - 1)));
  }, [flatList.length]);

  // Scroll selected item into view
  useEffect(() => {
    const item = listRef.current?.querySelector(`[data-index="${selectedIndex}"]`);
    item?.scrollIntoView({ block: 'nearest' });
  }, [selectedIndex]);

  const selectCommand = useCallback((cmd: CommandDefinition) => {
    if (cmd.arguments.length === 0) {
      onExecute(cmd.name);
      onClose();
    } else {
      setActiveCommand(cmd);
    }
  }, [onExecute, onClose]);

  const handleFormSubmit = useCallback((commandString: string) => {
    onExecute(commandString);
    onClose();
  }, [onExecute, onClose]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (activeCommand) return; // let the form handle keys when visible

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex(prev => Math.min(prev + 1, flatList.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex(prev => Math.max(prev - 1, 0));
        break;
      case 'Enter':
        e.preventDefault();
        if (flatList[selectedIndex]) {
          selectCommand(flatList[selectedIndex]);
        }
        break;
      case 'Escape':
        e.preventDefault();
        onClose();
        break;
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      {/* Palette */}
      <div
        className="relative bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-lg overflow-hidden flex flex-col"
        style={{ maxHeight: '60vh' }}
        onKeyDown={handleKeyDown}
      >
        {activeCommand ? (
          <CommandForm
            command={activeCommand}
            agents={agents}
            onSubmit={handleFormSubmit}
            onBack={() => setActiveCommand(null)}
          />
        ) : (
          <>
            {/* Search */}
            <div className="px-4 py-3 border-b border-slack-border">
              <input
                ref={inputRef}
                type="text"
                value={query}
                onChange={e => { setQuery(e.target.value); setSelectedIndex(0); }}
                placeholder="Search commands..."
                className="w-full bg-transparent text-sm text-slack-text placeholder-slack-textMuted focus:outline-none"
                autoComplete="off"
                spellCheck={false}
              />
            </div>

            {/* Results */}
            <div ref={listRef} className="flex-1 overflow-y-auto">
              {flatList.length === 0 ? (
                <div className="px-4 py-8 text-center text-sm text-slack-textMuted">
                  No matching commands
                </div>
              ) : (
                Array.from(grouped.entries()).map(([category, cmds]) => (
                  <div key={category}>
                    <div className="px-4 pt-3 pb-1 text-[11px] font-semibold uppercase tracking-wider text-slack-textMuted">
                      {category}
                    </div>
                    {cmds.map(cmd => {
                      const idx = flatList.indexOf(cmd);
                      const isSelected = idx === selectedIndex;
                      return (
                        <button
                          key={cmd.name}
                          data-index={idx}
                          onClick={() => selectCommand(cmd)}
                          onMouseEnter={() => setSelectedIndex(idx)}
                          className={`w-full text-left px-4 py-2 flex items-center gap-3 transition-colors ${
                            isSelected ? 'bg-slack-accent/10' : 'hover:bg-slack-bgHover'
                          }`}
                        >
                          <span className="font-mono text-sm text-slack-accent shrink-0">{cmd.name}</span>
                          <span className="text-xs text-slack-textMuted truncate">{cmd.description}</span>
                          {cmd.arguments.length > 0 && (
                            <span className="ml-auto text-xs text-slack-textMuted shrink-0 opacity-60">
                              {cmd.arguments.filter(a => a.required).length} arg{cmd.arguments.filter(a => a.required).length !== 1 ? 's' : ''}
                            </span>
                          )}
                        </button>
                      );
                    })}
                  </div>
                ))
              )}
            </div>

            {/* Footer hint */}
            <div className="px-4 py-2 border-t border-slack-border text-[11px] text-slack-textMuted flex gap-4">
              <span><kbd className="px-1 py-0.5 bg-slack-bgHover rounded text-xs">↑↓</kbd> navigate</span>
              <span><kbd className="px-1 py-0.5 bg-slack-bgHover rounded text-xs">↵</kbd> select</span>
              <span><kbd className="px-1 py-0.5 bg-slack-bgHover rounded text-xs">esc</kbd> close</span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
