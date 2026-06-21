import { useEffect, useState } from 'react';
import { DomainPacksPanel, type DomainPacksSection } from './DomainPacksPanel';
import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';

export type DomainPacksTab = DomainPacksSection;

interface DomainPacksModalProps {
  isOpen: boolean;
  onClose: () => void;
  serverAddr: string;
  initialTab?: DomainPacksTab;
}

export function DomainPacksModal({
  isOpen,
  onClose,
  serverAddr,
  initialTab,
}: DomainPacksModalProps) {
  const [tab, setTab] = useState<DomainPacksTab>(initialTab ?? 'store');

  useEffect(() => {
    if (isOpen && initialTab) setTab(initialTab);
  }, [isOpen, initialTab]);

  useShortcutOverlay('domainPacks', isOpen, onClose);

  if (!isOpen) return null;

  const hubHttp = serverAddr.startsWith('http') ? serverAddr : `http://${serverAddr}`;

  return (
    <div className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto py-6 px-4" role="presentation">
      <div className="fixed inset-0 bg-black/60" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="nj-domain-packs-title"
        className="relative z-10 flex w-full max-w-4xl lg:max-w-5xl flex-col overflow-hidden rounded-xl border border-slack-border bg-slack-bg shadow-2xl max-h-[min(90vh,900px)]"
      >
        <div className="flex shrink-0 flex-col gap-3 border-b border-slack-border px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
          <h2 id="nj-domain-packs-title" className="text-lg font-semibold text-slack-text truncate">
            Domain packs
          </h2>
          <div className="flex items-center gap-2">
            <div className="flex overflow-hidden rounded-md border border-slack-border text-xs" role="tablist">
              {(
                [
                  ['store', 'Store'],
                  ['tools', 'Tools & MCP'],
                  ['develop', 'Develop'],
                ] as const
              ).map(([id, label]) => (
                <button
                  key={id}
                  type="button"
                  role="tab"
                  aria-selected={tab === id}
                  onClick={() => setTab(id)}
                  className={`px-3 py-1.5 font-medium transition-colors ${
                    tab === id
                      ? 'bg-teal-600 text-white'
                      : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={onClose}
              className="rounded px-2 py-1 text-sm text-slack-textMuted hover:bg-slack-bgHover hover:text-slack-text"
              aria-label="Close"
            >
              Esc
            </button>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4">
          <DomainPacksPanel hubHttp={hubHttp} isActive={isOpen} section={tab} />
        </div>
      </div>
    </div>
  );
}
