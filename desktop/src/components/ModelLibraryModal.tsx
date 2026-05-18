import { useCallback, useEffect, useState } from 'react';
import { OllamaManager } from './OllamaManager';
import { OllamaModelLibrary } from './OllamaModelLibrary';
import { HfModelLibrary } from './HfModelLibrary';

type LibrarySource = 'ollama' | 'huggingface';
type BrowseDepth = 'grid' | 'detail';

interface ModelLibraryModalProps {
  isOpen: boolean;
  onClose: () => void;
  serverAddr: string;
  switchAllAgentProviders: (provider: string, model: string) => Promise<void>;
  onAfterModelChange?: () => void;
}

export function ModelLibraryModal({
  isOpen,
  onClose,
  serverAddr,
  switchAllAgentProviders,
  onAfterModelChange,
}: ModelLibraryModalProps) {
  const [source, setSource] = useState<LibrarySource>('ollama');
  const [browseDepth, setBrowseDepth] = useState<BrowseDepth>('grid');
  const [resetDetailSignal, setResetDetailSignal] = useState(0);

  const handleSourceChange = useCallback((next: LibrarySource) => {
    setSource(next);
    setBrowseDepth('grid');
    setResetDetailSignal((n) => n + 1);
  }, []);

  const handleBackFromDetail = useCallback(() => {
    setBrowseDepth('grid');
    setResetDetailSignal((n) => n + 1);
  }, []);

  useEffect(() => {
    if (!isOpen) {
      setBrowseDepth('grid');
      return;
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      e.preventDefault();
      if (browseDepth === 'detail') {
        handleBackFromDetail();
      } else {
        onClose();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [isOpen, onClose, browseDepth, handleBackFromDetail]);

  if (!isOpen) return null;

  const showBack = browseDepth === 'detail';

  return (
    <div className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto py-6 px-4" role="presentation">
      <div className="fixed inset-0 bg-black/60" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="nj-model-library-title"
        className="relative z-10 flex w-full max-w-4xl lg:max-w-5xl flex-col overflow-hidden rounded-xl border border-slack-border bg-slack-bg shadow-2xl max-h-[min(90vh,900px)]"
      >
        <div className="flex shrink-0 flex-col gap-3 border-b border-slack-border px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2 min-w-0">
            {showBack && (
              <button
                type="button"
                onClick={handleBackFromDetail}
                className="shrink-0 rounded px-2 py-1 text-sm text-amber-400 hover:bg-slack-bgHover hover:text-amber-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
              >
                ← Back
              </button>
            )}
            <h2 id="nj-model-library-title" className="text-lg font-semibold text-slack-text truncate">
              Model library
            </h2>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex rounded-md border border-slack-border overflow-hidden text-xs" role="tablist">
              <button
                type="button"
                role="tab"
                aria-selected={source === 'ollama'}
                onClick={() => handleSourceChange('ollama')}
                className={`px-3 py-1.5 font-medium transition-colors ${
                  source === 'ollama'
                    ? 'bg-amber-600 text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Ollama
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={source === 'huggingface'}
                onClick={() => handleSourceChange('huggingface')}
                className={`px-3 py-1.5 font-medium transition-colors ${
                  source === 'huggingface'
                    ? 'bg-amber-600 text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Hugging Face
              </button>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="rounded px-2 py-1 text-sm text-slack-textMuted hover:bg-slack-bgHover hover:text-slack-text focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
              aria-label="Close"
            >
              Esc
            </button>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4 space-y-4">
          {source === 'ollama' && (
            <>
              {browseDepth === 'grid' && (
                <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
                  <OllamaManager serverAddr={serverAddr} showLibraryHint={false} />
                </div>
              )}
              <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
                <OllamaModelLibrary
                  serverAddr={serverAddr}
                  switchAllAgentProviders={switchAllAgentProviders}
                  onAfterModelChange={onAfterModelChange}
                  onViewChange={setBrowseDepth}
                  resetDetailSignal={resetDetailSignal}
                />
              </div>
            </>
          )}
          {source === 'huggingface' && (
            <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
              <HfModelLibrary
                serverAddr={serverAddr}
                switchAllAgentProviders={switchAllAgentProviders}
                onAfterModelChange={onAfterModelChange}
                onViewChange={setBrowseDepth}
                resetDetailSignal={resetDetailSignal}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
