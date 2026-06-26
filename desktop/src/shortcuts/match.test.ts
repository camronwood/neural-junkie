import { describe, expect, it, vi } from 'vitest';
import { eventMatchesChord, getKeyEventContext, parseChord, shouldSkipShortcut } from './match';

describe('parseChord', () => {
  it('parses mod+shift+e', () => {
    expect(parseChord('mod+shift+e')).toEqual({
      key: 'e',
      mod: true,
      shift: true,
      alt: false,
      ctrl: false,
    });
  });

  it('parses alt+up', () => {
    expect(parseChord('alt+up')).toEqual({
      key: 'up',
      mod: false,
      shift: false,
      alt: true,
      ctrl: false,
    });
  });
});

describe('eventMatchesChord', () => {
  it('matches mod+, on mac metaKey', () => {
    vi.stubGlobal('navigator', { platform: 'MacIntel' });
    const event = {
      key: ',',
      metaKey: true,
      ctrlKey: false,
      shiftKey: false,
      altKey: false,
    } as KeyboardEvent;
    expect(eventMatchesChord(event, 'mod+,')).toBe(true);
    vi.unstubAllGlobals();
  });

  it('matches mod+shift+p on windows ctrlKey', () => {
    vi.stubGlobal('navigator', { platform: 'Win32' });
    const event = {
      key: 'p',
      metaKey: false,
      ctrlKey: true,
      shiftKey: true,
      altKey: false,
    } as KeyboardEvent;
    expect(eventMatchesChord(event, 'mod+shift+p')).toBe(true);
    vi.unstubAllGlobals();
  });
});

describe('shouldSkipShortcut', () => {
  it('skips global shortcuts in inputs unless allowed', () => {
    const ctx = { inInput: true, inMonaco: false, inTerminal: false };
    expect(shouldSkipShortcut({}, ctx)).toBe(true);
    expect(shouldSkipShortcut({ allowInInput: true }, ctx)).toBe(false);
  });

  it('skips when monaco focused if skipInMonaco', () => {
    const ctx = { inInput: true, inMonaco: true, inTerminal: false };
    expect(shouldSkipShortcut({ skipInMonaco: true }, ctx)).toBe(true);
  });
});

describe('getKeyEventContext', () => {
  it('detects monaco editor context', () => {
    const monaco = document.createElement('div');
    monaco.className = 'monaco-editor';
    const child = document.createElement('div');
    monaco.appendChild(child);
    document.body.appendChild(monaco);
    expect(getKeyEventContext(child).inMonaco).toBe(true);
    document.body.removeChild(monaco);
  });

  it('detects xterm helper textarea as terminal focus', () => {
    const xterm = document.createElement('div');
    xterm.className = 'xterm';
    const textarea = document.createElement('textarea');
    textarea.className = 'xterm-helper-textarea';
    xterm.appendChild(textarea);
    document.body.appendChild(xterm);
    expect(getKeyEventContext(textarea).inTerminal).toBe(true);
    document.body.removeChild(xterm);
  });
});
