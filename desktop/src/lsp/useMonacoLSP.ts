import { useEffect, useRef } from 'react';
import type * as Monaco from 'monaco-editor';
import { getHubBaseURL } from '../config/hubUrl';
import { useDiagnosticsStore, type EditorDiagnostic } from '../stores/diagnosticsStore';
import { getLspConnection, lspRequest, type LspLang } from './lspConnection';

function lspLangFromPath(path: string, language?: string): LspLang | null {
  const lang = (language || '').toLowerCase();
  const ext = path.split('.').pop()?.toLowerCase();
  if (lang === 'go' || ext === 'go') return 'go';
  if (lang === 'rust' || ext === 'rs') return 'rust';
  if (lang === 'python' || ext === 'py') return 'python';
  return null;
}

function toFileURI(workspaceRoot: string | undefined, relPath: string): string {
  const root = (workspaceRoot || '').replace(/\\/g, '/');
  const rel = relPath.replace(/\\/g, '/').replace(/^\//, '');
  if (root.startsWith('/')) {
    return `file://${root}/${rel}`;
  }
  return `file:///${root}/${rel}`;
}

function relPathFromUri(uri: string, workspaceRoot?: string): string {
  const root = (workspaceRoot || '').replace(/\\/g, '/');
  const u = uri.replace(/^file:\/\//, '');
  if (root && u.startsWith(root)) {
    return u.slice(root.length).replace(/^\//, '');
  }
  return u;
}

function lspSeverityToEditor(sev?: number): EditorDiagnostic['severity'] {
  if (sev === 1) return 'error';
  if (sev === 2) return 'warning';
  return 'info';
}

/** LSP via WebSocket sync + REST-backed Monaco providers (IDE v4). */
export function useMonacoLSP(
  editor: Monaco.editor.IStandaloneCodeEditor | null,
  monaco: typeof Monaco | null,
  workspaceId: string | undefined,
  workspaceRoot: string | undefined,
  tabPath: string | undefined,
  language: string | undefined,
  content: string | undefined
) {
  const disposables = useRef<Monaco.IDisposable[]>([]);
  const setForPath = useDiagnosticsStore((s) => s.setForPath);

  useEffect(() => {
    for (const d of disposables.current) d.dispose();
    disposables.current = [];
    if (!editor || !monaco || !workspaceId || !tabPath) return;
    const lspLang = lspLangFromPath(tabPath, language);
    if (!lspLang) return;

    const uri = toFileURI(workspaceRoot, tabPath);
    const conn = getLspConnection(workspaceId, lspLang);
    let cancelled = false;

    const applyDiagnostics = (params: {
      uri: string;
      diagnostics: Array<{
        range: { start: { line: number; character: number }; end: { line: number; character: number } };
        message: string;
        severity?: number;
      }>;
    }) => {
      const rel = relPathFromUri(params.uri, workspaceRoot);
      const items: EditorDiagnostic[] = params.diagnostics.map((d) => ({
        path: rel,
        line: d.range.start.line + 1,
        column: d.range.start.character + 1,
        endLine: d.range.end.line + 1,
        endColumn: d.range.end.character + 1,
        message: d.message,
        severity: lspSeverityToEditor(d.severity),
      }));
      setForPath(rel, items);
      const model = editor.getModel();
      if (model && model.uri.toString() === uri) {
        monaco.editor.setModelMarkers(
          model,
          'nj-lsp',
          items.map((d) => ({
            startLineNumber: d.line,
            startColumn: d.column,
            endLineNumber: d.endLine ?? d.line,
            endColumn: d.endColumn ?? d.column + 1,
            message: d.message,
            severity:
              d.severity === 'error'
                ? monaco.MarkerSeverity.Error
                : d.severity === 'warning'
                  ? monaco.MarkerSeverity.Warning
                  : monaco.MarkerSeverity.Info,
          }))
        );
      }
    };

    void conn.connect().then(() => {
      if (cancelled) return;
      conn.didOpen(uri, lspLang, editor.getValue());
      disposables.current.push({
        dispose: conn.onNotification('textDocument/publishDiagnostics', (params) => {
          applyDiagnostics(params as Parameters<typeof applyDiagnostics>[0]);
        }),
      });
    });

    const model = editor.getModel();
    if (model) {
      disposables.current.push(
        model.onDidChangeContent(() => {
          conn.didChange(uri, model.getValue());
        })
      );
    }

    disposables.current.push(
      monaco.languages.registerHoverProvider(lspLang, {
        provideHover: async (m, position) => {
          try {
            const res = await fetch(`${getHubBaseURL()}/api/lsp/hover`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                workspace_id: workspaceId,
                lang: lspLang,
                uri,
                line: position.lineNumber - 1,
                character: position.column - 1,
                text: m.getValue(),
              }),
            });
            if (!res.ok) return null;
            const data = (await res.json()) as { contents?: { value?: string } };
            const value = data?.contents?.value;
            if (!value) return null;
            return {
              range: new monaco.Range(position.lineNumber, position.column, position.lineNumber, position.column),
              contents: [{ value }],
            };
          } catch {
            return null;
          }
        },
      })
    );

    disposables.current.push(
      monaco.languages.registerCompletionItemProvider(lspLang, {
        triggerCharacters: ['.', ':', '/', '"', "'"],
        provideCompletionItems: async (m, position) => {
          try {
            const result = await lspRequest<{ items?: Monaco.languages.CompletionItem[] } | Monaco.languages.CompletionItem[]>(
              workspaceId,
              lspLang,
              'textDocument/completion',
              {
                textDocument: { uri },
                position: { line: position.lineNumber - 1, character: position.column - 1 },
              },
              { uri, text: m.getValue() }
            );
            const items = Array.isArray(result) ? result : result?.items ?? [];
            return { suggestions: items as Monaco.languages.CompletionItem[] };
          } catch {
            return { suggestions: [] };
          }
        },
      })
    );

    disposables.current.push(
      monaco.languages.registerDefinitionProvider(lspLang, {
        provideDefinition: async (m, position) => {
          try {
            const result = await lspRequest<Monaco.languages.Location | Monaco.languages.Location[]>(
              workspaceId,
              lspLang,
              'textDocument/definition',
              {
                textDocument: { uri },
                position: { line: position.lineNumber - 1, character: position.column - 1 },
              },
              { uri, text: m.getValue() }
            );
            if (!result) return null;
            return Array.isArray(result) ? result : [result];
          } catch {
            return null;
          }
        },
      })
    );

    disposables.current.push(
      monaco.languages.registerReferenceProvider(lspLang, {
        provideReferences: async (m, position) => {
          try {
            const result = await lspRequest<Monaco.languages.Location[]>(
              workspaceId,
              lspLang,
              'textDocument/references',
              {
                textDocument: { uri },
                position: { line: position.lineNumber - 1, character: position.column - 1 },
                context: { includeDeclaration: true },
              },
              { uri, text: m.getValue() }
            );
            return result ?? [];
          } catch {
            return [];
          }
        },
      })
    );

    disposables.current.push(
      monaco.languages.registerRenameProvider(lspLang, {
        provideRenameEdits: async (m, position, newName) => {
          try {
            const result = await lspRequest<{ changes?: Record<string, Array<{ range: unknown; newText: string }>> }>(
              workspaceId,
              lspLang,
              'textDocument/rename',
              {
                textDocument: { uri },
                position: { line: position.lineNumber - 1, character: position.column - 1 },
                newName,
              },
              { uri, text: m.getValue() }
            );
            if (!result?.changes) return { edits: [] };
            const edits: Monaco.languages.IWorkspaceTextEdit[] = [];
            for (const [editUri, changeList] of Object.entries(result.changes)) {
              const resource = monaco.Uri.parse(editUri);
              for (const ch of changeList) {
                const r = ch.range as {
                  start: { line: number; character: number };
                  end: { line: number; character: number };
                };
                edits.push({
                  resource,
                  versionId: undefined,
                  textEdit: {
                    range: new monaco.Range(
                      r.start.line + 1,
                      r.start.character + 1,
                      r.end.line + 1,
                      r.end.character + 1
                    ),
                    text: ch.newText,
                  },
                });
              }
            }
            return { edits };
          } catch {
            return { edits: [] };
          }
        },
      })
    );

    return () => {
      cancelled = true;
      conn.didClose(uri);
      for (const d of disposables.current) d.dispose();
      disposables.current = [];
    };
  }, [editor, monaco, workspaceId, workspaceRoot, tabPath, language, content, setForPath]);
}
