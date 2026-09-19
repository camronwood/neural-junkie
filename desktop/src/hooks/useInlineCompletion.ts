import { useEffect, useRef } from 'react';
import type * as Monaco from 'monaco-editor';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useSettingsStore } from '../stores/settingsStore';
import { usePacksStore } from '../stores/packsStore';
import { useEditorStore } from '../stores/editorStore';
import {
  buildInlineCompletionLocalContext,
  buildNeighborSnippets,
} from '../utils/inlineCompletionContext';
import {
  RecordInlineCompletionAccept,
  RecordInlineCompletionDismiss,
  RecordInlineCompletionPartialAccept,
  RecordInlineCompletionShown,
} from '../utils/inlineCompletionMetrics';

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
    const ideOn =
      usePacksStore.getState().ideEnabled() &&
      usePacksStore.getState().hasCapability('inline-completion');
    if (!ideOn) return;

    const provider = monaco.languages.registerInlineCompletionsProvider(
      { pattern: '**/*' },
      {
        debounceDelayMs: 250,
        provideInlineCompletions: async (model, position, _context, token) => {
          const layout = useSettingsStore.getState().layoutSettings;
          if (!layout.inlineCompletionEnabled) {
            return { items: [] };
          }

          const offset = model.getOffsetAt(position);
          const fullText = model.getValue();
          const line = model.getLineContent(position.lineNumber);
          const before = line.slice(0, position.column - 1);
          if (before.trim().length < 3) return { items: [] };

          const { prefix, suffix, context: fileContext } = buildInlineCompletionLocalContext(
            fullText,
            offset
          );

          const editorState = useEditorStore.getState();
          const neighbor_snippets = buildNeighborSnippets({
            tabs: editorState.tabs,
            activeTabId: editorState.activeTabId,
            activePath: filePath,
            recentEdits: editorState.recentEdits,
          });

          // Multiple items enable Monaco cycling (Alt+] / Alt+[) on explicit and automatic triggers.
          const n = 2;

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
                const modelName =
                  (layout.inlineCompletionModel || '').trim() || 'qwen2.5-coder:1.5b';
                const result = await api.devCompleteStream(
                  {
                    prefix,
                    suffix,
                    language,
                    path: filePath,
                    context: fileContext,
                    model: modelName,
                    neighbor_snippets,
                    n,
                    signal: abortRef.current.signal,
                  }
                );
                if (token.isCancellationRequested) {
                  resolve({ items: [] });
                  return;
                }
                const texts =
                  result.completions?.length > 0
                    ? result.completions
                    : result.completion
                      ? [result.completion]
                      : [];
                if (texts.length === 0) {
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
                  items: texts.map((insertText) => ({
                    insertText,
                    range,
                  })),
                  enableForwardStability: true,
                });
              } catch {
                resolve({ items: [] });
              }
            }, 250);
          });
        },
        handleItemDidShow: () => {
          RecordInlineCompletionShown();
        },
        handlePartialAccept: () => {
          RecordInlineCompletionPartialAccept();
        },
        handleEndOfLifetime: (_completions, _item, reason) => {
          if (reason.kind === monaco.languages.InlineCompletionEndOfLifeReasonKind.Accepted) {
            RecordInlineCompletionAccept();
          } else if (reason.kind === monaco.languages.InlineCompletionEndOfLifeReasonKind.Rejected) {
            RecordInlineCompletionDismiss();
          } else if (
            reason.kind === monaco.languages.InlineCompletionEndOfLifeReasonKind.Ignored &&
            reason.userTypingDisagreed
          ) {
            RecordInlineCompletionDismiss();
          }
        },
        disposeInlineCompletions: () => {},
      }
    );

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      abortRef.current?.abort();
      provider.dispose();
    };
  }, [editor, monaco, enabled, language, filePath]);
}
