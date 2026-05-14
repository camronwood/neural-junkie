import { useEffect } from 'react';
import { OllamaManager } from './OllamaManager';
import { OllamaModelLibrary } from './OllamaModelLibrary';

interface OllamaModelLibraryModalProps {
  isOpen: boolean;
  onClose: () => void;
  serverAddr: string;
  switchAllAgentProviders: (provider: string, model: string) => Promise<void>;
}

export function OllamaModelLibraryModal({
  isOpen,
  onClose,
  serverAddr,
  switchAllAgentProviders,
}: OllamaModelLibraryModalProps) {
  useEffect(() => {
    if (!isOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto py-6 px-4" role="presentation">
      <div
        className="fixed inset-0 bg-black/60"
        onClick={onClose}
        aria-hidden
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="nj-model-library-title"
        className="relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-slack-border bg-slack-bg shadow-2xl max-h-[min(90vh,900px)]"
      >
        <div className="flex shrink-0 items-center justify-between gap-3 border-b border-slack-border px-4 py-3">
          <h2 id="nj-model-library-title" className="text-lg font-semibold text-slack-text">
            Ollama model library
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded px-2 py-1 text-sm text-slack-textMuted hover:bg-slack-bgHover hover:text-slack-text focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
            aria-label="Close"
          >
            Esc
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4 space-y-6">
          <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
            <OllamaManager serverAddr={serverAddr} />
          </div>
          <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
            <OllamaModelLibrary
              serverAddr={serverAddr}
              switchAllAgentProviders={switchAllAgentProviders}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
