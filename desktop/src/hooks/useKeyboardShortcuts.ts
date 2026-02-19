import { useEffect } from 'react';

interface KeyboardShortcutsProps {
  onOpenSettings: () => void;
  onToggleTerminal?: () => void;
}

export function useKeyboardShortcuts({ onOpenSettings, onToggleTerminal }: KeyboardShortcutsProps) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Check for Cmd/Ctrl + , (comma) - Settings
      if ((e.metaKey || e.ctrlKey) && e.key === ',') {
        e.preventDefault();
        onOpenSettings();
      }
      
      // Check for Cmd/Ctrl + J - Terminal
      if ((e.metaKey || e.ctrlKey) && e.key === 'j') {
        e.preventDefault();
        onToggleTerminal?.();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onOpenSettings, onToggleTerminal]);
}
