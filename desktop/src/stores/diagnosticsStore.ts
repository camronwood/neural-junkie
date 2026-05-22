import { createWithEqualityFn as create } from 'zustand/traditional';

export interface EditorDiagnostic {
  path: string;
  line: number;
  column: number;
  endLine?: number;
  endColumn?: number;
  message: string;
  severity: 'error' | 'warning' | 'info';
}

interface DiagnosticsState {
  byPath: Record<string, EditorDiagnostic[]>;
  setForPath: (path: string, items: EditorDiagnostic[]) => void;
  clearPath: (path: string) => void;
  allForWorkspace: (prefix?: string) => EditorDiagnostic[];
}

export const useDiagnosticsStore = create<DiagnosticsState>((set, get) => ({
  byPath: {},
  setForPath: (path, items) =>
    set((s) => ({ byPath: { ...s.byPath, [path]: items } })),
  clearPath: (path) =>
    set((s) => {
      const next = { ...s.byPath };
      delete next[path];
      return { byPath: next };
    }),
  allForWorkspace: (prefix) => {
    const all: EditorDiagnostic[] = [];
    for (const [path, items] of Object.entries(get().byPath)) {
      if (!prefix || path.startsWith(prefix)) {
        all.push(...items);
      }
    }
    return all.sort((a, b) => a.path.localeCompare(b.path) || a.line - b.line);
  },
}));
