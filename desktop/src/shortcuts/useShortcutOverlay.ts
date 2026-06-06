import { useEffect } from 'react';
import {
  useShortcutContextStore,
  type ShortcutOverlayType,
} from '../stores/shortcutContextStore';

export function useShortcutOverlay(
  type: ShortcutOverlayType,
  isOpen: boolean,
  onClose: () => void
) {
  const pushOverlay = useShortcutContextStore((s) => s.pushOverlay);
  const popOverlay = useShortcutContextStore((s) => s.popOverlay);

  useEffect(() => {
    if (!isOpen) {
      popOverlay(type);
      return;
    }
    pushOverlay({ type, onClose });
    return () => popOverlay(type);
  }, [isOpen, onClose, popOverlay, pushOverlay, type]);
}
