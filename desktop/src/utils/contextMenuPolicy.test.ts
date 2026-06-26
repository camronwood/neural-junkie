import { describe, expect, it } from 'vitest';
import { shouldAllowNativeContextMenu } from './contextMenuPolicy';

describe('shouldAllowNativeContextMenu', () => {
  it('allows monaco and xterm', () => {
    const monaco = document.createElement('div');
    monaco.className = 'monaco-editor';
    const child = document.createElement('div');
    monaco.appendChild(child);
    document.body.appendChild(monaco);
    expect(shouldAllowNativeContextMenu(child)).toBe(true);
    document.body.removeChild(monaco);

    const xterm = document.createElement('div');
    xterm.className = 'xterm';
    document.body.appendChild(xterm);
    expect(shouldAllowNativeContextMenu(xterm)).toBe(true);
    document.body.removeChild(xterm);
  });

  it('allows form controls', () => {
    const input = document.createElement('input');
    expect(shouldAllowNativeContextMenu(input)).toBe(true);
  });

  it('blocks generic app chrome', () => {
    const div = document.createElement('div');
    expect(shouldAllowNativeContextMenu(div)).toBe(false);
  });
});
