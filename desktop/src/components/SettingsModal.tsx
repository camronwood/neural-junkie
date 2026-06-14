import { useState, useEffect } from 'react';
import { useSettingsStore } from '../stores/settingsStore';
import { useChatStore } from '../stores/chatStore';
import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';
import {
  type SettingsTab,
  SETTINGS_NAV_GROUPS,
  resolveSettingsTab,
} from './settings/settingsNav';
import { AppearanceSettingsTab } from './settings/AppearanceSettingsTab';
import { LayoutSettingsTab } from './settings/LayoutSettingsTab';
import { KeyboardSettingsTab } from './settings/KeyboardSettingsTab';
import { ChatSettingsTab } from './settings/ChatSettingsTab';
import { ProvidersSettingsTab } from './settings/ProvidersSettingsTab';
import { ModelsPerformanceSettingsTab } from './settings/ModelsPerformanceSettingsTab';
import { CollabRoutingSettingsTab } from './settings/CollabRoutingSettingsTab';
import { MemoryLearningSettingsTab } from './settings/MemoryLearningSettingsTab';
import { ApiCredentialsSettingsTab } from './settings/ApiCredentialsSettingsTab';
import { AssistantToolsSettingsTab } from './settings/AssistantToolsSettingsTab';
import { SlackSettingsTab } from './settings/SlackSettingsTab';
import { DomainPacksSettingsTab } from './settings/DomainPacksSettingsTab';
import { SecuritySettingsTab } from './settings/SecuritySettingsTab';
import { AboutSettingsTab } from './settings/AboutSettingsTab';

export type { SettingsTab } from './settings/settingsNav';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialTab?: SettingsTab | string;
}

export function SettingsModal({ isOpen, onClose, initialTab }: SettingsModalProps) {
  const { loadSettings, loadLayoutSettings } = useSettingsStore();
  const chatServerAddr = useChatStore((s) => s.serverAddr);
  const hubHttp =
    chatServerAddr.startsWith('http') ? chatServerAddr : `http://${chatServerAddr}`;
  const [activeTab, setActiveTab] = useState<SettingsTab>('appearance');

  useEffect(() => {
    if (isOpen && initialTab) {
      const resolved = resolveSettingsTab(initialTab) ?? 'appearance';
      setActiveTab(resolved);
    }
  }, [isOpen, initialTab]);

  useEffect(() => {
    if (isOpen) {
      loadSettings();
      loadLayoutSettings();
    }
  }, [isOpen, loadSettings, loadLayoutSettings]);

  useShortcutOverlay('settings', isOpen, onClose);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 p-4" role="presentation">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="nj-settings-title"
        className="relative z-10 mx-auto flex h-full w-full max-w-[1600px] flex-col overflow-hidden rounded-xl border border-slack-border bg-slack-bg shadow-2xl"
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slack-border px-6 py-4">
          <h2 id="nj-settings-title" className="text-xl font-bold text-slack-text">
            Settings
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close settings"
            className="text-slack-textMuted transition-colors hover:text-slack-text"
          >
            <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex min-h-0 flex-1">
          <nav
            className="flex w-[220px] shrink-0 flex-col overflow-y-auto border-r border-slack-border bg-slack-bgHover/20 py-2"
            aria-label="Settings sections"
          >
            {SETTINGS_NAV_GROUPS.map((group) => (
              <div key={group.title} className="mb-2">
                <div className="px-4 py-2 text-xs font-semibold uppercase tracking-wide text-slack-textMuted">
                  {group.title}
                </div>
                {group.items.map((tab) => (
                  <button
                    key={tab.id}
                    type="button"
                    onClick={() => setActiveTab(tab.id)}
                    className={`w-full border-l-2 px-4 py-2.5 text-left text-sm font-medium transition-colors ${
                      activeTab === tab.id
                        ? 'border-slack-accent bg-slack-bgHover text-slack-text'
                        : 'border-transparent text-slack-textMuted hover:bg-slack-bgHover/50 hover:text-slack-text'
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>
            ))}
          </nav>

          <div className="min-w-0 flex-1 overflow-y-auto p-6">
            <AppearanceSettingsTab hubHttp={hubHttp} isActive={activeTab === 'appearance'} />
            <LayoutSettingsTab hubHttp={hubHttp} isActive={activeTab === 'layout'} />
            <KeyboardSettingsTab hubHttp={hubHttp} isActive={activeTab === 'keyboard'} />
            <ChatSettingsTab hubHttp={hubHttp} isActive={activeTab === 'chat'} />
            <ProvidersSettingsTab hubHttp={hubHttp} isActive={activeTab === 'providers'} />
            <ModelsPerformanceSettingsTab hubHttp={hubHttp} isActive={activeTab === 'models-performance'} />
            <CollabRoutingSettingsTab hubHttp={hubHttp} isActive={activeTab === 'collab-routing'} />
            <MemoryLearningSettingsTab hubHttp={hubHttp} isActive={activeTab === 'memory-learning'} />
            <ApiCredentialsSettingsTab hubHttp={hubHttp} isActive={activeTab === 'api-credentials'} />
            <AssistantToolsSettingsTab hubHttp={hubHttp} isActive={activeTab === 'assistant-tools'} />
            <SlackSettingsTab hubHttp={hubHttp} isActive={activeTab === 'slack'} />
            <DomainPacksSettingsTab hubHttp={hubHttp} isActive={activeTab === 'domain-packs'} />
            <SecuritySettingsTab hubHttp={hubHttp} isActive={activeTab === 'security'} />
            <AboutSettingsTab hubHttp={hubHttp} isActive={activeTab === 'about'} />
          </div>
        </div>
      </div>
    </div>
  );
}
