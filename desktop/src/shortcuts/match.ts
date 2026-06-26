import type { ParsedChord } from './types';

export function isMacPlatform(): boolean {
  if (typeof navigator === 'undefined') return false;
  return navigator.platform.toUpperCase().includes('MAC');
}

export function parseChord(chord: string): ParsedChord {
  const parts = chord.toLowerCase().split('+').map((p) => p.trim());
  const key = parts[parts.length - 1];
  return {
    key,
    mod: parts.includes('mod'),
    shift: parts.includes('shift'),
    alt: parts.includes('alt'),
    ctrl: parts.includes('ctrl'),
  };
}

export interface KeyEventContext {
  inInput: boolean;
  inMonaco: boolean;
  inTerminal: boolean;
}

export function isTerminalFocusTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el) return false;
  return Boolean(
    el.closest?.('.xterm') ||
      el.closest?.('.xterm-helper-textarea') ||
      el.classList?.contains('xterm-helper-textarea')
  );
}

export function getKeyEventContext(target: EventTarget | null): KeyEventContext {
  const el = target as HTMLElement | null;
  const inMonaco = Boolean(el?.closest?.('.monaco-editor'));
  const inTerminal = isTerminalFocusTarget(target);
  const inInput =
    inMonaco ||
    inTerminal ||
    el?.tagName === 'INPUT' ||
    el?.tagName === 'TEXTAREA' ||
    el?.isContentEditable === true;
  return { inInput, inMonaco, inTerminal };
}

export function eventMatchesChord(event: KeyboardEvent, chord: string): boolean {
  const parsed = parseChord(chord);
  const isMac = isMacPlatform();
  const modPressed = isMac ? event.metaKey : event.ctrlKey;

  if (parsed.mod && !modPressed) return false;
  if (!parsed.mod && modPressed && parsed.key !== 'tab') return false;
  if (parsed.shift !== event.shiftKey) return false;
  if (parsed.alt !== event.altKey) return false;
  if (parsed.ctrl && !event.ctrlKey) return false;

  const eventKey = event.key.toLowerCase();
  const wantKey = parsed.key.toLowerCase();

  if (wantKey === 'escape') return eventKey === 'escape';
  if (wantKey === 'tab') return eventKey === 'tab';
  if (wantKey === 'backspace') return eventKey === 'backspace';
  if (wantKey === 'enter') return eventKey === 'enter';
  if (wantKey === ',') return event.key === ',';
  if (wantKey === '\\') return event.key === '\\' || event.key === '|';
  if (wantKey === ']') return event.key === ']';

  return eventKey === wantKey;
}

export function shouldSkipShortcut(
  def: { allowInInput?: boolean; skipInMonaco?: boolean; skipInTerminal?: boolean },
  ctx: KeyEventContext
): boolean {
  if (ctx.inInput && !def.allowInInput) {
    if (ctx.inMonaco && def.skipInMonaco === false) return false;
    if (ctx.inTerminal && def.skipInTerminal === false) return false;
    return true;
  }
  if (ctx.inMonaco && def.skipInMonaco) return true;
  if (ctx.inTerminal && def.skipInTerminal) return true;
  return false;
}
