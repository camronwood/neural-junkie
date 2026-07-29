import { useCallback, useEffect, useState } from 'react';
import { OllamaManager } from './OllamaManager';
import { OllamaModelLibrary } from './OllamaModelLibrary';
import { HfModelLibrary } from './HfModelLibrary';
import { InstalledModelsLibrary } from './InstalledModelsLibrary';
import { LoraTrainingPanel, type LoraTrainPrefill } from './LoraTrainingPanel';
import { usePacksStore } from '../stores/packsStore';
import { PACK_CAP } from '../stores/packCapabilities';
import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';
import { useModelTransferStore } from '../stores/modelTransferStore';

type LibrarySource = 'installed' | 'ollama' | 'huggingface' | 'train';
type BrowseDepth = 'grid' | 'detail';

interface ModelLibraryModalProps {
  isOpen: boolean;
  onClose: () => void;
  serverAddr: string;
  switchAllAgentProviders: (provider: string, model: string) => Promise<void>;
  switchAgentProvider?: (agentId: string, provider: string, model: string) => Promise<void>;
  runtimeAgents?: { id: string; name: string; type: string }[];
  onAfterModelChange?: () => void;
  defaultChannel?: string;
  initialTab?: LibrarySource;
  loraTrainPrefill?: LoraTrainPrefill | null;
}

export function ModelLibraryModal({
  isOpen,
  onClose,
  serverAddr,
  switchAllAgentProviders,
  switchAgentProvider,
  runtimeAgents,
  onAfterModelChange,
  defaultChannel,
  initialTab,
  loraTrainPrefill,
}: ModelLibraryModalProps) {
  const hasLoRATraining = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_TRAINING));
  const hasLoRACompose = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_COMPOSE));
  const hasActiveTransfers = useModelTransferStore((s) =>
    Object.values(s.transfers).some((t) => t.status === 'downloading')
  );
  const keepOllamaMounted = useModelTransferStore((s) =>
    Object.values(s.transfers).some((t) => t.source === 'ollama' && t.status === 'downloading')
  );
  const keepHfMounted = useModelTransferStore((s) =>
    Object.values(s.transfers).some((t) => t.source === 'huggingface' && t.status === 'downloading')
  );
  const [source, setSource] = useState<LibrarySource>(initialTab ?? 'ollama');
  const [browseDepth, setBrowseDepth] = useState<BrowseDepth>('grid');
  const [resetDetailSignal, setResetDetailSignal] = useState(0);

  useEffect(() => {
    if (isOpen && initialTab) {
      setSource(initialTab);
    }
  }, [isOpen, initialTab]);

  const handleSourceChange = useCallback((next: LibrarySource) => {
    setSource(next);
    setBrowseDepth('grid');
    setResetDetailSignal((n) => n + 1);
  }, []);

  const handleDownloadStarted = useCallback(() => {
    setSource('installed');
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
    }
  }, [isOpen]);

  const handleShortcutClose = useCallback(() => {
    if (browseDepth === 'detail') {
      handleBackFromDetail();
    } else {
      onClose();
    }
  }, [browseDepth, handleBackFromDetail, onClose]);

  useShortcutOverlay('modelLibrary', isOpen, handleShortcutClose);

  if (!isOpen) return null;

  const showBack = browseDepth === 'detail';
  // Keep download sources mounted (hidden) so in-flight pulls/downloads aren't aborted on tab switch.
  const showOllama = source === 'ollama' || keepOllamaMounted;
  const showHf = source === 'huggingface' || keepHfMounted;

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
                aria-selected={source === 'installed'}
                onClick={() => handleSourceChange('installed')}
                className={`px-3 py-1.5 font-medium transition-colors ${
                  source === 'installed'
                    ? 'bg-amber-600 text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Installed{hasActiveTransfers ? ' · …' : ''}
              </button>
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
              <button
                type="button"
                role="tab"
                aria-selected={source === 'train'}
                disabled={!hasLoRATraining}
                title={hasLoRATraining ? undefined : 'Enable Specialist tuning pack'}
                onClick={() => hasLoRATraining && handleSourceChange('train')}
                className={`px-3 py-1.5 font-medium transition-colors ${
                  source === 'train'
                    ? 'bg-purple-700 text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                } ${!hasLoRATraining ? 'opacity-40 cursor-not-allowed' : ''}`}
              >
                Train LoRA
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
          <div className={source === 'installed' ? '' : 'hidden'}>
            <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
              <InstalledModelsLibrary
                serverAddr={serverAddr}
                switchAllAgentProviders={switchAllAgentProviders}
                onAfterModelChange={onAfterModelChange}
                onViewChange={setBrowseDepth}
                resetDetailSignal={resetDetailSignal}
              />
            </div>
          </div>
          {browseDepth === 'grid' && source === 'ollama' && (
            <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
              <OllamaManager serverAddr={serverAddr} showLibraryHint={false} />
            </div>
          )}
          {showOllama && (
            <div className={source === 'ollama' ? '' : 'hidden'}>
              <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
                <OllamaModelLibrary
                  serverAddr={serverAddr}
                  switchAllAgentProviders={switchAllAgentProviders}
                  onAfterModelChange={onAfterModelChange}
                  onViewChange={setBrowseDepth}
                  resetDetailSignal={resetDetailSignal}
                  onDownloadStarted={handleDownloadStarted}
                />
              </div>
            </div>
          )}
          {showHf && (
            <div className={source === 'huggingface' ? '' : 'hidden'}>
              <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
                <HfModelLibrary
                  serverAddr={serverAddr}
                  switchAllAgentProviders={switchAllAgentProviders}
                  switchAgentProvider={switchAgentProvider}
                  runtimeAgents={runtimeAgents}
                  onAfterModelChange={onAfterModelChange}
                  onViewChange={setBrowseDepth}
                  resetDetailSignal={resetDetailSignal}
                  canComposeLoRA={hasLoRACompose}
                  onDownloadStarted={handleDownloadStarted}
                />
              </div>
            </div>
          )}
          {source === 'train' && (
            <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
              <LoraTrainingPanel
                serverAddr={serverAddr}
                defaultChannel={defaultChannel}
                switchAgentProvider={switchAgentProvider}
                runtimeAgents={runtimeAgents}
                prefill={loraTrainPrefill ?? undefined}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
