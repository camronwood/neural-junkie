import { useEffect } from 'react';
import type * as Monaco from 'monaco-editor';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useDiagnosticsStore, type EditorDiagnostic } from '../stores/diagnosticsStore';

function normalizeWorkspacePath(path: string): string {
  return path.replace(/\\/g, '/').replace(/^\.\//, '');
}

function pathsMatch(workspacePath: string, diagnosticPath: string): boolean {
  const a = normalizeWorkspacePath(workspacePath);
  const b = normalizeWorkspacePath(diagnosticPath);
  return a === b || a.endsWith(`/${b}`) || b.endsWith(`/${a}`);
}

function toMonacoSeverity(
  severity: EditorDiagnostic['severity'],
  monaco: typeof Monaco
): Monaco.MarkerSeverity {
  if (severity === 'error') return monaco.MarkerSeverity.Error;
  if (severity === 'warning') return monaco.MarkerSeverity.Warning;
  return monaco.MarkerSeverity.Info;
}

function applyMonacoMarkers(
  monaco: typeof Monaco,
  model: Monaco.editor.ITextModel,
  tabPath: string,
  items: EditorDiagnostic[]
) {
  const forTab = items.filter((d) => pathsMatch(tabPath, d.path));
  monaco.editor.setModelMarkers(
    model,
    'nj-lsp-lite',
    forTab.map((d) => ({
      startLineNumber: d.line,
      startColumn: d.column,
      endLineNumber: d.endLine ?? d.line,
      endColumn: d.endColumn ?? d.column + 1,
      message: d.message,
      severity: toMonacoSeverity(d.severity, monaco),
    }))
  );
}

/** Fetch workspace-wide Go/Rust/Python diagnostics and sync store + Monaco markers. */
export function useWorkspaceLSPDiagnostics(
  editor: Monaco.editor.IStandaloneCodeEditor | null,
  monaco: typeof Monaco | null,
  workspaceId: string | undefined,
  tabPath: string | undefined,
  language: string | undefined,
  enabled = true
) {
  useEffect(() => {
    if (!enabled || !workspaceId || !tabPath) return;
    const lang = (language || '').toLowerCase();
    const ext = tabPath.split('.').pop()?.toLowerCase();
    let lspLang: 'go' | 'rust' | 'python' | null = null;
    if (lang === 'go' || ext === 'go') lspLang = 'go';
    else if (lang === 'rust' || ext === 'rs') lspLang = 'rust';
    else if (lang === 'python' || ext === 'py') lspLang = 'python';
    if (!lspLang) return;

    let cancelled = false;
    const api = new ChatAPI(getHubBaseURL());
    const fetch =
      lspLang === 'go'
        ? api.getGoLSPDiagnostics(workspaceId)
        : api.getLSPDiagnostics(lspLang, workspaceId);

    void fetch.then((items) => {
      if (cancelled) return;
      const byPath: Record<string, EditorDiagnostic[]> = {};
      for (const d of items) {
        const rel = normalizeWorkspacePath(d.path);
        const entry: EditorDiagnostic = {
          path: rel,
          line: d.line,
          column: d.column,
          message: d.message,
          severity: d.severity === 'warning' ? 'warning' : 'error',
        };
        byPath[rel] = [...(byPath[rel] ?? []), entry];
      }
      const store = useDiagnosticsStore.getState();
      const next = { ...store.byPath };
      for (const key of Object.keys(next)) {
        const existing = next[key] ?? [];
        const withoutLsp = existing.filter((x) => !x.message.startsWith('[lsp]'));
        if (withoutLsp.length === 0) delete next[key];
        else next[key] = withoutLsp;
      }
      for (const [path, diags] of Object.entries(byPath)) {
        next[path] = [
          ...(next[path] ?? []),
          ...diags.map((d) => ({ ...d, message: `[lsp] ${d.message}` })),
        ];
      }
      useDiagnosticsStore.setState({ byPath: next });

      const model = editor?.getModel();
      if (monaco && model && tabPath) {
        const all = Object.values(next).flat();
        applyMonacoMarkers(monaco, model, tabPath, all);
      }
    });

    return () => {
      cancelled = true;
    };
  }, [editor, monaco, workspaceId, tabPath, language, enabled]);
}
