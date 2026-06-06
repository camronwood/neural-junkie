import { useEffect } from 'react';
import { dispatchShortcut } from './dispatcher';

export function useShortcutDispatcher(enabled = true) {
  useEffect(() => {
    if (!enabled) return;

    const onKeyDown = (event: KeyboardEvent) => {
      void dispatchShortcut(event);
    };

    document.addEventListener('keydown', onKeyDown, true);
    return () => document.removeEventListener('keydown', onKeyDown, true);
  }, [enabled]);
}
