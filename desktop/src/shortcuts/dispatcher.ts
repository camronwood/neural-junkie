import { getShortcutHandler } from '../stores/shortcutHandlersStore';
import { eventMatchesChord, getKeyEventContext, shouldSkipShortcut } from './match';
import { SHORTCUT_REGISTRY } from './registry';
import type { ShortcutDefinition } from './types';

const sortedRegistry = [...SHORTCUT_REGISTRY].sort((a, b) => b.priority - a.priority);

export async function dispatchShortcut(event: KeyboardEvent): Promise<boolean> {
  const ctx = getKeyEventContext(event.target);

  for (const def of sortedRegistry) {
    if (!eventMatchesChord(event, def.chord)) continue;
    if (def.when && !def.when()) continue;
    if (shouldSkipShortcut(def, ctx)) continue;

    const handler = getShortcutHandler(def.handlerId);
    if (def.preventDefault !== false) {
      event.preventDefault();
      event.stopPropagation();
    }
    await handler();
    return true;
  }

  return false;
}

export function findShortcutById(id: string): ShortcutDefinition | undefined {
  return SHORTCUT_REGISTRY.find((s) => s.id === id);
}
