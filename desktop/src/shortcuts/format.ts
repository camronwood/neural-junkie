import { isMacPlatform } from './match';

const KEY_LABELS: Record<string, string> = {
  escape: 'Esc',
  tab: 'Tab',
  backspace: 'Backspace',
  enter: 'Enter',
  ',': ',',
  '\\': '\\',
  ']': ']',
};

export function formatChord(chord: string): string {
  const isMac = isMacPlatform();
  const parts = chord.split('+').map((p) => p.trim());
  const key = parts[parts.length - 1];
  const mods: string[] = [];

  for (const part of parts.slice(0, -1)) {
    const lower = part.toLowerCase();
    if (lower === 'mod') {
      mods.push(isMac ? '⌘' : 'Ctrl');
    } else if (lower === 'shift') {
      mods.push(isMac ? '⇧' : 'Shift');
    } else if (lower === 'alt') {
      mods.push(isMac ? '⌥' : 'Alt');
    } else if (lower === 'ctrl') {
      mods.push('Ctrl');
    }
  }

  const keyLabel = KEY_LABELS[key.toLowerCase()] ?? key.toUpperCase();
  return [...mods, keyLabel].join(isMac ? '' : '+');
}

export function formatChordHint(chord: string): string {
  const formatted = formatChord(chord);
  return isMacPlatform() ? formatted : formatted.replace(/\+/g, '+');
}
