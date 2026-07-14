import { useEffect, useRef } from 'react';
import type * as Monaco from 'monaco-editor';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useSettingsStore } from '../stores/settingsStore';
import { usePacksStore } from '../stores/packsStore';

export function useInlineCompletion(
  editor: Monaco.editor.IStandaloneCodeEditor | null,
  monaco: typeof Monaco | null,
  enabled: boolean,
  language: string | undefined,
  filePath: string | undefined
) {
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!editor || !monaco || !enabled) return;
    const ideOn = usePacksStore.getState().ideEnabled() && usePacksStore.getState().hasCapability('inline-completion');
    if (!ideOn) return;

    const provider = monaco.languages.registerInlineCompletionsProvider(
      { pattern: '**/*' },
      {
        provideInlineCompletions: async (model, position, _context, token) => {
          const layout = useSettingsStore.getState().layoutSettings;
          if (!layout.inlineCompletionEnabled) {
            return { items: [] };
          }
          const lineNum = position.lineNumber;
          const line = model.getLineContent(lineNum);
          const before = line.slice(0, position.column - 1);
          const after = line.slice(position.column - 1);
          if (before.trim().length < 3) return { items: [] };
          const contextStart = Math.max(1, lineNum - 30);
          const contextEnd = Math.min(model.getLineCount(), lineNum + 30);
          const contextLines: string[] = [];
          for (let i = contextStart; i <= contextEnd; i++) {
            contextLines.push(model.getLineContent(i));
          }
          const fileContext = contextLines.join('\n');

          return new Promise((resolve) => {
            if (debounceRef.current) clearTimeout(debounceRef.current);
            debounceRef.current = setTimeout(async () => {
              if (token.isCancellationRequested) {
                resolve({ items: [] });
                return;
              }
              abortRef.current?.abort();
              abortRef.current = new AbortController();
              try {
                const api = new ChatAPI(getHubBaseURL());
                const { completion } = await api.devComplete({
                  prefix: before,
                  suffix: after,
                  language,
                  path: filePath,
                  context: fileContext,
                  model: layout.inlineCompletionModel,
                });
                if (!completion || token.isCancellationRequested) {
                  resolve({ items: [] });
                  return;
                }
                const range = new monaco.Range(
                  position.lineNumber,
                  position.column,
                  position.lineNumber,
                  position.column
                );
                resolve({
                  items: [
                    {
                      insertText: completion,
                      range,
                    },
                  ],
                });
              } catch {
                resolve({ items: [] });
              }
            }, 250);
          });
        },
        disposeInlineCompletions: () => {},
      }
    );

    return () => provider.dispose();
  }, [editor, monaco, enabled, language, filePath]);
}
