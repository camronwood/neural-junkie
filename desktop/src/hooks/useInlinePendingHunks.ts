import { useEffect, useRef } from 'react';
import { shallow } from 'zustand/shallow';
import { useDiagnosticsStore } from '../stores/diagnosticsStore';
import { useEditorStore } from '../stores/editorStore';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { applyHunkToContent, parseUnifiedDiff } from '../utils/parseUnifiedDiff';

const TS_LANGS = new Set([
  'typescript',
  'javascript',
  'typescriptreact',
  'javascriptreact',
  'tsx',
  'jsx',
]);

export function useMonacoDiagnostics(
  editor: import('monaco-editor').editor.IStandaloneCodeEditor | null,
  monaco: typeof import('monaco-editor') | null,
  tabPath: string | undefined,
  language: string | undefined
) {
  useEffect(() => {
    if (!editor || !monaco || !tabPath) return;
    const model = editor.getModel();
    if (!model) return;
    const lang = (language || '').toLowerCase();
    const isJsTs =
      TS_LANGS.has(lang) ||
      /\.(tsx?|jsx?)$/i.test(tabPath);
    if (!isJsTs) {
      return;
    }
    const ts = monaco.languages.typescript;
    const opts = {
      target: ts.ScriptTarget.ES2020,
      allowNonTsExtensions: true,
      moduleResolution: ts.ModuleResolutionKind.NodeJs,
      module: ts.ModuleKind.ESNext,
      noEmit: true,
      strict: false,
    };
    ts.typescriptDefaults.setCompilerOptions(opts);
    ts.javascriptDefaults.setCompilerOptions(opts);

    const sync = () => {
      const markers = monaco.editor.getModelMarkers({ resource: model.uri });
      useDiagnosticsStore.getState().setForPath(
        tabPath,
        markers.map((m) => ({
          path: tabPath,
          line: m.startLineNumber,
          column: m.startColumn,
          endLine: m.endLineNumber,
          endColumn: m.endColumn,
          message: m.message,
          severity:
            m.severity === monaco.MarkerSeverity.Error
              ? ('error' as const)
              : m.severity === monaco.MarkerSeverity.Warning
                ? ('warning' as const)
                : ('info' as const),
        }))
      );
    };
    sync();
    const sub = monaco.editor.onDidChangeMarkers(sync);
    return () => sub.dispose();
  }, [editor, monaco, tabPath, language]);
}

export function useInlinePendingHunks(
  editor: import('monaco-editor').editor.IStandaloneCodeEditor | null,
  monaco: typeof import('monaco-editor') | null,
  activeTabId: string | null
) {
  const { pendingChanges, previewData, selectedChangeId } = useFileChangeStore(
    (s) => ({
      pendingChanges: s.pendingChanges,
      previewData: s.previewData,
      selectedChangeId: s.selectedChangeId,
    }),
    shallow
  );
  const decorationIdsRef = useRef<string[]>([]);
  const acceptedHunksRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (!editor || !monaco || !activeTabId) return;
    const tab = useEditorStore.getState().getTabById(activeTabId);
    if (!tab) return;

    const change =
      pendingChanges.find((c) => {
        const p = c.file_path || c.new_path || '';
        return p === tab.path || p.endsWith(`/${tab.path}`);
      }) ?? null;

    const model = editor.getModel();
    if (!model) return;

    if (!change) {
      decorationIdsRef.current = editor.deltaDecorations(decorationIdsRef.current, []);
      acceptedHunksRef.current.clear();
      return;
    }

    if (!previewData?.diff || selectedChangeId !== change.id) {
      void useFileChangeStore.getState().getFileDiff(change.id);
      return;
    }

    const hunks = parseUnifiedDiff(previewData.diff).filter((h) => !acceptedHunksRef.current.has(h.id));
    const decorations: import('monaco-editor').editor.IModelDeltaDecoration[] = [];

    let lineOffset = 0;
    for (const hunk of hunks) {
      const start = hunk.newStart + lineOffset;
      if (hunk.removedLines.length > 0) {
        decorations.push({
          range: new monaco.Range(start, 1, start + hunk.removedLines.length - 1, 1),
          options: {
            isWholeLine: true,
            className: 'nj-inline-removed',
            glyphMarginClassName: 'nj-glyph-removed',
            glyphMarginHoverMessage: { value: 'Reject hunk' },
          },
        });
      }
      if (hunk.addedLines.length > 0) {
        decorations.push({
          range: new monaco.Range(start, 1, start, 1),
          options: {
            isWholeLine: true,
            className: 'nj-inline-added',
            glyphMarginClassName: 'nj-glyph-added',
            glyphMarginHoverMessage: { value: 'Accept hunk' },
          },
        });
      }
      lineOffset += hunk.addedLines.length - hunk.removedLines.length;
    }

    decorationIdsRef.current = editor.deltaDecorations(decorationIdsRef.current, decorations);

    const sub = editor.onMouseDown((e) => {
      if (e.target.type !== monaco.editor.MouseTargetType.GUTTER_GLYPH_MARGIN) return;
      const line = e.target.position?.lineNumber;
      if (!line) return;
      const hunk = hunks.find(
        (h) => line >= h.newStart && line <= h.newStart + Math.max(h.removedLines.length, h.addedLines.length)
      );
      if (!hunk) return;
      const next = applyHunkToContent(model.getValue(), hunk);
      useEditorStore.getState().updateTabContent(activeTabId, next);
      model.setValue(next);
      acceptedHunksRef.current.add(hunk.id);
      useFileChangeStore.getState().selectChange(change.id);
    });

    return () => {
      sub.dispose();
      decorationIdsRef.current = editor.deltaDecorations(decorationIdsRef.current, []);
    };
  }, [editor, monaco, activeTabId, pendingChanges, previewData, selectedChangeId]);
}
