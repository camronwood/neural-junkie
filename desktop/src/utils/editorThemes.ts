import type { ColorTheme } from '../stores/settingsStore';
import type { ITheme } from '@xterm/xterm';

/** Monaco editor theme id for the active color theme. */
export function getMonacoThemeId(theme: ColorTheme): string {
  return theme === 'flat' ? 'nj-flat' : 'vs-dark';
}

/** xterm.js palette keyed by app color theme. */
export function getTerminalTheme(theme: ColorTheme): ITheme {
  if (theme === 'flat') {
    return {
      background: '#0d0d1a',
      foreground: '#e4e4ef',
      cursor: '#a78bfa',
      selectionBackground: '#3d3d5c',
      black: '#0d0d1a',
      red: '#f7768e',
      green: '#34d399',
      yellow: '#e0af68',
      blue: '#7dd3fc',
      magenta: '#a78bfa',
      cyan: '#7dd3fc',
      white: '#e4e4ef',
      brightBlack: '#3d3d5c',
      brightRed: '#f7768e',
      brightGreen: '#34d399',
      brightYellow: '#e0af68',
      brightBlue: '#7dd3fc',
      brightMagenta: '#c084fc',
      brightCyan: '#a5e4fc',
      brightWhite: '#f0f0f8',
    };
  }
  // Slack / Tokyo Night–adjacent default
  return {
    background: '#1a1b26',
    foreground: '#c0caf5',
    cursor: '#c0caf5',
    selectionBackground: '#33467c',
    black: '#15161e',
    red: '#f7768e',
    green: '#9ece6a',
    yellow: '#e0af68',
    blue: '#7aa2f7',
    magenta: '#bb9af7',
    cyan: '#7dcfff',
    white: '#a9b1d6',
    brightBlack: '#414868',
    brightRed: '#f7768e',
    brightGreen: '#9ece6a',
    brightYellow: '#e0af68',
    brightBlue: '#7aa2f7',
    brightMagenta: '#bb9af7',
    brightCyan: '#7dcfff',
    brightWhite: '#c0caf5',
  };
}

/** Register custom Monaco themes (call once per Monaco instance). */
export function registerMonacoThemes(monaco: typeof import('monaco-editor')): void {
  monaco.editor.defineTheme('nj-flat', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'comment', foreground: '8b8fa8', fontStyle: 'italic' },
      { token: 'keyword', foreground: 'c084fc' },
      { token: 'string', foreground: '7dd3fc' },
      { token: 'number', foreground: 'fbbf24' },
      { token: 'type', foreground: 'a5e4fc' },
      { token: 'type.identifier', foreground: 'a5e4fc' },
      { token: 'identifier', foreground: 'e4e4ef' },
      { token: 'delimiter', foreground: '8b8fa8' },
      { token: 'tag', foreground: 'c084fc' },
      { token: 'attribute.name', foreground: 'a78bfa' },
      { token: 'attribute.value', foreground: '7dd3fc' },
    ],
    colors: {
      'editor.background': '#0d0d1a',
      'editor.foreground': '#e4e4ef',
      'editorLineNumber.foreground': '#5a5a7a',
      'editorLineNumber.activeForeground': '#a78bfa',
      'editor.selectionBackground': '#3d3d5c',
      'editor.inactiveSelectionBackground': '#252b45',
      'editorCursor.foreground': '#a78bfa',
      'editor.lineHighlightBackground': '#16162d',
      'editorIndentGuide.background': '#3d3d5c',
      'editorIndentGuide.activeBackground': '#a78bfa',
      'editorWidget.background': '#16162d',
      'editorWidget.border': '#3d3d5c',
      'input.background': '#252b45',
      'input.border': '#3d3d5c',
      'dropdown.background': '#16162d',
      'list.hoverBackground': '#252b45',
      'list.activeSelectionBackground': '#3d3d5c',
      'minimap.background': '#0d0d1a',
      'scrollbarSlider.background': '#3d3d5c80',
      'scrollbarSlider.hoverBackground': '#5a5a7a80',
    },
  });
}
