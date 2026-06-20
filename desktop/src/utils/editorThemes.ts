import type { ColorTheme } from '../stores/settingsStore';
import type { ITheme } from '@xterm/xterm';

/** Monaco editor theme id for the active color theme. */
export function getMonacoThemeId(theme: ColorTheme): string {
  switch (theme) {
    case 'flat':
      return 'nj-flat';
    case 'roving':
      return 'nj-roving';
    case 'brand':
      return 'nj-brand';
    default:
      return 'vs-dark';
  }
}

/** xterm.js palette keyed by app color theme. */
export function getTerminalTheme(theme: ColorTheme): ITheme {
  switch (theme) {
    case 'flat':
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
    case 'roving':
      return {
        background: '#f9f7f2',
        foreground: '#3b2f3d',
        cursor: '#9b7fa8',
        selectionBackground: '#d8c9de',
        black: '#3b2f3d',
        red: '#b84a5a',
        green: '#4a8f6e',
        yellow: '#a8844a',
        blue: '#6b7fa8',
        magenta: '#9b7fa8',
        cyan: '#7a9b9b',
        white: '#f9f7f2',
        brightBlack: '#6e6270',
        brightRed: '#c45a6a',
        brightGreen: '#5aa07e',
        brightYellow: '#c49a5a',
        brightBlue: '#7b8fb8',
        brightMagenta: '#ab8fb8',
        brightCyan: '#8aabab',
        brightWhite: '#ffffff',
      };
    case 'brand':
      return {
        background: '#1a161a',
        foreground: '#ffffff',
        cursor: '#f44a69',
        selectionBackground: '#2d262d',
        black: '#120d11',
        red: '#f44a69',
        green: '#34d399',
        yellow: '#e0af68',
        blue: '#c0b8c0',
        magenta: '#ff5a79',
        cyan: '#a098a0',
        white: '#ffffff',
        brightBlack: '#4a424a',
        brightRed: '#ff5a79',
        brightGreen: '#4ade80',
        brightYellow: '#f0c878',
        brightBlue: '#d0c8d0',
        brightMagenta: '#ff7a93',
        brightCyan: '#b0a8b0',
        brightWhite: '#ffffff',
      };
    default:
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

  monaco.editor.defineTheme('nj-roving', {
    base: 'vs',
    inherit: true,
    rules: [
      { token: 'comment', foreground: '6e6270', fontStyle: 'italic' },
      { token: 'keyword', foreground: '856b91' },
      { token: 'string', foreground: '6b7fa8' },
      { token: 'number', foreground: 'a8844a' },
      { token: 'type', foreground: '9b7fa8' },
      { token: 'type.identifier', foreground: '9b7fa8' },
      { token: 'identifier', foreground: '3b2f3d' },
      { token: 'delimiter', foreground: '6e6270' },
      { token: 'tag', foreground: '856b91' },
      { token: 'attribute.name', foreground: '9b7fa8' },
      { token: 'attribute.value', foreground: '6b7fa8' },
    ],
    colors: {
      'editor.background': '#f9f7f2',
      'editor.foreground': '#3b2f3d',
      'editorLineNumber.foreground': '#a89aa8',
      'editorLineNumber.activeForeground': '#9b7fa8',
      'editor.selectionBackground': '#d8c9de',
      'editor.inactiveSelectionBackground': '#f0ebe3',
      'editorCursor.foreground': '#9b7fa8',
      'editor.lineHighlightBackground': '#f0ebe3',
      'editorIndentGuide.background': '#e5ddd4',
      'editorIndentGuide.activeBackground': '#9b7fa8',
      'editorWidget.background': '#ffffff',
      'editorWidget.border': '#e5ddd4',
      'input.background': '#ffffff',
      'input.border': '#e5ddd4',
      'dropdown.background': '#ffffff',
      'list.hoverBackground': '#f0ebe3',
      'list.activeSelectionBackground': '#d8c9de',
      'minimap.background': '#f9f7f2',
      'scrollbarSlider.background': '#e5ddd480',
      'scrollbarSlider.hoverBackground': '#c4b8ae80',
    },
  });

  monaco.editor.defineTheme('nj-brand', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'comment', foreground: 'a098a0', fontStyle: 'italic' },
      { token: 'keyword', foreground: 'ff5a79' },
      { token: 'string', foreground: 'c0b8c0' },
      { token: 'number', foreground: 'e0af68' },
      { token: 'type', foreground: 'f44a69' },
      { token: 'type.identifier', foreground: 'f44a69' },
      { token: 'identifier', foreground: 'ffffff' },
      { token: 'delimiter', foreground: 'a098a0' },
      { token: 'tag', foreground: 'ff5a79' },
      { token: 'attribute.name', foreground: 'f44a69' },
      { token: 'attribute.value', foreground: 'c0b8c0' },
    ],
    colors: {
      'editor.background': '#1a161a',
      'editor.foreground': '#ffffff',
      'editorLineNumber.foreground': '#6a626a',
      'editorLineNumber.activeForeground': '#f44a69',
      'editor.selectionBackground': '#2d262d',
      'editor.inactiveSelectionBackground': '#252028',
      'editorCursor.foreground': '#f44a69',
      'editor.lineHighlightBackground': '#252028',
      'editorIndentGuide.background': '#2d262d',
      'editorIndentGuide.activeBackground': '#f44a69',
      'editorWidget.background': '#120d11',
      'editorWidget.border': '#2d262d',
      'input.background': '#252028',
      'input.border': '#2d262d',
      'dropdown.background': '#120d11',
      'list.hoverBackground': '#252028',
      'list.activeSelectionBackground': '#2d262d',
      'minimap.background': '#1a161a',
      'scrollbarSlider.background': '#2d262d80',
      'scrollbarSlider.hoverBackground': '#4a424a80',
    },
  });
}
