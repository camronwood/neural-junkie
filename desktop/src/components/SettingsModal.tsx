import { useState, useEffect } from 'react';
import { shallow } from 'zustand/shallow';
import { useSettingsStore } from '../stores/settingsStore';
import { useChatStore } from '../stores/chatStore';
import { agentSidebarHideKey, parseDMDisplayName } from '../utils/dmChannelDisplay';
import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';
import { getShortcutsForDisplay, formatChord } from '../shortcuts';
import { AppearanceSettingsTab } from './settings/AppearanceSettingsTab';
import { IntegrationsSettingsTab } from './settings/IntegrationsSettingsTab';
import { AIProvidersSettingsTab } from './settings/AIProvidersSettingsTab';
import { DomainPacksSettingsTab } from './settings/DomainPacksSettingsTab';
import { SecuritySettingsTab } from './settings/SecuritySettingsTab';
import { AboutSettingsTab } from './settings/AboutSettingsTab';

export type SettingsTab =
  | 'appearance'
  | 'layout'
  | 'keyboard'
  | 'chat'
  | 'integrations'
  | 'ai-providers'
  | 'domain-packs'
  | 'security'
  | 'about';

const SETTINGS_NAV: Array<{ id: SettingsTab; label: string }> = [
  { id: 'appearance', label: 'Appearance' },
  { id: 'layout', label: 'Layout' },
  { id: 'keyboard', label: 'Keyboard' },
  { id: 'chat', label: 'Chat & agents' },
  { id: 'integrations', label: 'Integrations' },
  { id: 'ai-providers', label: 'AI Providers' },
  { id: 'domain-packs', label: 'Domain packs' },
  { id: 'security', label: 'Security' },
  { id: 'about', label: 'About' },
];

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialTab?: SettingsTab;
}

function addUnique(items: string[] | undefined, value: string): string[] {
  const cur = items ?? [];
  return cur.includes(value) ? cur : [...cur, value];
}

function removeItem(items: string[] | undefined, value: string): string[] {
  return (items ?? []).filter((x) => x !== value);
}

export function SettingsModal({ isOpen, onClose, initialTab }: SettingsModalProps) {
  const {
    settings,
    layoutSettings,
    loadSettings,
    updateSettings,
    loadLayoutSettings,
    updateLayoutSettings,
  } = useSettingsStore();
  const { serverAddr: chatServerAddr, channels, agents } = useChatStore(
    (s) => ({
      serverAddr: s.serverAddr,
      channels: s.channels,
      agents: s.agents,
    }),
    shallow
  );
  const hubHttp =
    chatServerAddr.startsWith('http') ? chatServerAddr : `http://${chatServerAddr}`;
  const [activeTab, setActiveTab] = useState<SettingsTab>('appearance');

  useEffect(() => {
    if (isOpen && initialTab) {
      setActiveTab(initialTab);
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

  const findAgentForDmChannel = (channelName: string) => {
    const ch = channels.find((c) => c.name === channelName);
    if (!ch) return undefined;
    const agentId = ch.agents?.[0]?.id ?? ch.members?.[0];
    return (
      (agentId ? agents.find((a) => a.id === agentId) : undefined) ??
      agents.find((a) => a.name.toLowerCase() === parseDMDisplayName(ch).toLowerCase())
    );
  };

  const unhideDmChannel = (name: string) => {
    const agent = findAgentForDmChannel(name);
    const key = agent ? agentSidebarHideKey(agent) : '';
    void updateSettings({
      hiddenDmChannelNames: removeItem(settings.hiddenDmChannelNames, name),
      deletedDmChannelNames: removeItem(settings.deletedDmChannelNames, name),
      ...(agent
        ? {
            hiddenAgentIdsForSidebar: removeItem(settings.hiddenAgentIdsForSidebar, agent.id),
            hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
            deletedAgentIdsForSidebar: removeItem(settings.deletedAgentIdsForSidebar, agent.id),
            deletedAgentSidebarKeys: removeItem(settings.deletedAgentSidebarKeys, key),
          }
        : {}),
    });
  };

  const deleteHiddenDmChannel = (name: string) => {
    const agent = findAgentForDmChannel(name);
    const key = agent ? agentSidebarHideKey(agent) : '';
    void updateSettings({
      hiddenDmChannelNames: removeItem(settings.hiddenDmChannelNames, name),
      deletedDmChannelNames: addUnique(settings.deletedDmChannelNames, name),
      ...(agent
        ? {
            hiddenAgentIdsForSidebar: removeItem(settings.hiddenAgentIdsForSidebar, agent.id),
            hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
            deletedAgentIdsForSidebar: addUnique(settings.deletedAgentIdsForSidebar, agent.id),
            deletedAgentSidebarKeys: addUnique(settings.deletedAgentSidebarKeys, key),
          }
        : {}),
    });
  };

  const unhideCollaborationChannel = (name: string) => {
    void updateSettings({
      hiddenCollaborationChannelNames: removeItem(settings.hiddenCollaborationChannelNames, name),
      deletedCollaborationChannelNames: removeItem(settings.deletedCollaborationChannelNames, name),
    });
  };

  const deleteHiddenCollaborationChannel = (name: string) => {
    void updateSettings({
      hiddenCollaborationChannelNames: removeItem(settings.hiddenCollaborationChannelNames, name),
      deletedCollaborationChannelNames: addUnique(settings.deletedCollaborationChannelNames, name),
    });
  };

  const unhideAgentShortcutKey = (key: string) => {
    void updateSettings({
      hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
      deletedAgentSidebarKeys: removeItem(settings.deletedAgentSidebarKeys, key),
    });
  };

  const deleteHiddenAgentShortcutKey = (key: string) => {
    void updateSettings({
      hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
      deletedAgentSidebarKeys: addUnique(settings.deletedAgentSidebarKeys, key),
    });
  };

  const unhideAgentShortcutId = (id: string) => {
    const agent = agents.find((a) => a.id === id);
    const key = agent ? agentSidebarHideKey(agent) : '';
    void updateSettings({
      hiddenAgentIdsForSidebar: removeItem(settings.hiddenAgentIdsForSidebar, id),
      deletedAgentIdsForSidebar: removeItem(settings.deletedAgentIdsForSidebar, id),
      ...(agent
        ? {
            hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
            deletedAgentSidebarKeys: removeItem(settings.deletedAgentSidebarKeys, key),
          }
        : {}),
    });
  };

  const deleteHiddenAgentShortcutId = (id: string) => {
    const agent = agents.find((a) => a.id === id);
    const key = agent ? agentSidebarHideKey(agent) : '';
    void updateSettings({
      hiddenAgentIdsForSidebar: removeItem(settings.hiddenAgentIdsForSidebar, id),
      deletedAgentIdsForSidebar: addUnique(settings.deletedAgentIdsForSidebar, id),
      ...(agent
        ? {
            hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
            deletedAgentSidebarKeys: addUnique(settings.deletedAgentSidebarKeys, key),
          }
        : {}),
    });
  };

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
            {SETTINGS_NAV.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                className={`border-l-2 px-4 py-2.5 text-left text-sm font-medium transition-colors ${
                  activeTab === tab.id
                    ? 'border-slack-accent bg-slack-bgHover text-slack-text'
                    : 'border-transparent text-slack-textMuted hover:bg-slack-bgHover/50 hover:text-slack-text'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </nav>

          <div className="min-w-0 flex-1 overflow-y-auto p-6">
            <AppearanceSettingsTab hubHttp={hubHttp} isActive={activeTab === 'appearance'} />
            {activeTab === 'layout' && (
  <div className="space-y-6">
    <div className="mb-4">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Layout preset</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        IDE puts the editor and agent first; Team keeps the classic chat-first workspace.
      </p>
      <div className="flex gap-2">
        {(['team', 'ide'] as const).map((preset) => (
          <button
            key={preset}
            type="button"
            onClick={() => {
              void import('../utils/layoutPresets').then(({ panelsForPreset }) =>
                updateLayoutSettings(panelsForPreset(preset))
              );
            }}
            className={`px-3 py-1.5 text-sm rounded border ${
              (layoutSettings.layoutPreset ?? 'team') === preset
                ? 'border-slack-accent bg-slack-accent/20 text-slack-text'
                : 'border-slack-border text-slack-textMuted hover:text-slack-text'
            }`}
          >
            {preset === 'ide' ? 'IDE (project-first)' : 'Team (chat-first)'}
          </button>
        ))}
      </div>
    </div>

    <div className="mb-4">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Toolbar actions</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        On wide screens, show toolbar chips in the top bar or a right sidebar. Narrow windows
        always use the sidebar.
      </p>
      <div className="flex gap-2">
        {(['top', 'sidebar'] as const).map((placement) => (
          <button
            key={placement}
            type="button"
            onClick={() => updateLayoutSettings({ toolbarChipsPlacement: placement })}
            className={`px-3 py-1.5 text-sm rounded border ${
              (layoutSettings.toolbarChipsPlacement ?? 'top') === placement
                ? 'border-slack-accent bg-slack-accent/20 text-slack-text'
                : 'border-slack-border text-slack-textMuted hover:text-slack-text'
            }`}
          >
            {placement === 'top' ? 'Top bar' : 'Right sidebar'}
          </button>
        ))}
      </div>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Memory monitor</div>
        <div className="text-sm text-slack-textMuted">
          Show live system RAM and Ollama loaded-model usage in the toolbar
        </div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.memoryMonitorEnabled !== false}
          onChange={(e) => updateLayoutSettings({ memoryMonitorEnabled: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Routing badges on messages</div>
        <div className="text-sm text-slack-textMuted">
          Show which model ran on each agent reply (chat model, tool model, routing reason)
        </div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.showRoutingOnMessages !== false}
          onChange={(e) => updateLayoutSettings({ showRoutingOnMessages: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Inline completion (ghost text)</div>
        <div className="text-sm text-slack-textMuted">Ollama FIM via hub when Software development pack is on</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.inlineCompletionEnabled ?? false}
          onChange={(e) => updateLayoutSettings({ inlineCompletionEnabled: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Editor agent trust</div>
        <div className="text-sm text-slack-textMuted">
          How file changes from IDE-mode chat are applied (Ask/Agent toggle on the main composer)
        </div>
      </div>
      <select
        value={layoutSettings.editorAgentTrust ?? 'interactive'}
        onChange={(e) =>
          updateLayoutSettings({
            editorAgentTrust: e.target.value as 'interactive' | 'auto_apply_edits' | 'yolo',
          })
        }
        className="text-sm bg-slack-bg border border-slack-border rounded px-2 py-1"
      >
        <option value="interactive">Interactive (approve each)</option>
        <option value="auto_apply_edits">Auto-apply edits</option>
        <option value="yolo">Yolo (tools)</option>
      </select>
    </div>

    <div className="mb-4">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Panel Visibility</h3>
      <p className="text-sm text-slack-textMuted">
        Configure which panels are visible by default when the app starts. You can still toggle panels manually at any time.
      </p>
    </div>

    {/* Files Panel */}
    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Files Panel</div>
        <div className="text-sm text-slack-textMuted">File explorer for browsing workspaces and files</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.filesPanelVisible}
          onChange={(e) => updateLayoutSettings({ filesPanelVisible: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    {/* Editor Panel */}
    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Editor Panel</div>
        <div className="text-sm text-slack-textMuted">Code editor for viewing and editing files</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.editorPanelVisible}
          onChange={(e) => updateLayoutSettings({ editorPanelVisible: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    {/* Terminal Panel */}
    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Terminal Panel</div>
        <div className="text-sm text-slack-textMuted">Terminal for executing commands and viewing output</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.terminalPanelVisible}
          onChange={(e) => updateLayoutSettings({ terminalPanelVisible: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    {/* My Agents Panel */}
    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">My Agents Panel</div>
        <div className="text-sm text-slack-textMuted">Manage and view your custom repository agents</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.myAgentsPanelVisible}
          onChange={(e) => updateLayoutSettings({ myAgentsPanelVisible: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    {/* Pending Changes Panel */}
    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Pending Changes Panel</div>
        <div className="text-sm text-slack-textMuted">View and manage pending file changes</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.pendingChangesPanelVisible}
          onChange={(e) => updateLayoutSettings({ pendingChangesPanelVisible: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    {/* Sidebar agent shortcuts */}
    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Sidebar agent shortcuts</div>
        <div className="text-sm text-slack-textMuted">
          Show agents without a DM under Direct Messages. Turn off for a cleaner list; open DMs stay.
        </div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.sidebarAgentsVisible}
          onChange={(e) => updateLayoutSettings({ sidebarAgentsVisible: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>
  </div>
            )}
            {activeTab === 'keyboard' && (
  <div className="space-y-4">
    <div>
      <h3 className="text-lg font-semibold text-slack-text mb-2">Keyboard shortcuts</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Fixed defaults (not customizable yet). Toolbar buttons show the same chords on hover.
      </p>
    </div>
    <div className="border border-slack-border rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-slack-bgHover/50 text-left text-slack-textMuted">
          <tr>
            <th className="px-4 py-2 font-medium">Action</th>
            <th className="px-4 py-2 font-medium w-40">Shortcut</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slack-border">
          {getShortcutsForDisplay().map((row) => (
            <tr key={row.id} className="text-slack-text">
              <td className="px-4 py-2">{row.label}</td>
              <td className="px-4 py-2">
                <kbd className="rounded bg-slack-bgHover px-1.5 py-0.5 font-mono text-xs">
                  {formatChord(row.chord)}
                </kbd>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </div>
            )}
            {activeTab === 'chat' && (
  <div className="space-y-8">
    <div>
      <h3 className="text-lg font-semibold text-slack-text mb-2">User rules (markdown)</h3>
      <p className="text-sm text-slack-textMuted mb-2">
        Included on every message you send (main chat and threads). Agents treat this as your standing instructions.
      </p>
      <textarea
        value={settings.userRulesMarkdown ?? ''}
        onChange={(e) => void updateSettings({ userRulesMarkdown: e.target.value })}
        rows={10}
        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text font-mono focus:outline-none focus:ring-2 focus:ring-slack-accent"
        placeholder={'- Prefer concise answers\n- Stack: Rust, TypeScript'}
      />
      <p className="text-xs text-slack-textMuted mt-1">{(settings.userRulesMarkdown ?? '').length} characters</p>
    </div>

    <div>
      <h3 className="text-lg font-semibold text-slack-text mb-2">Hidden from sidebar</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        DM rows, collaborations, or agent shortcuts you hid. Use Unhide to restore them, or Delete to keep them off the sidebar permanently.
      </p>
      {(settings.hiddenDmChannelNames?.length ?? 0) === 0 &&
      (settings.hiddenCollaborationChannelNames?.length ?? 0) === 0 &&
      (settings.hiddenAgentSidebarKeys?.length ?? 0) === 0 &&
      (settings.hiddenAgentIdsForSidebar?.length ?? 0) === 0 ? (
        <p className="text-sm text-slack-textMuted">None. Use × on a row in the sidebar.</p>
      ) : (
        <div className="space-y-2">
          {(settings.hiddenDmChannelNames ?? []).map((name) => (
            <div
              key={`dm-${name}`}
              className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
            >
              <span className="text-sm text-slack-text truncate" title={name}>
                DM: {channels.find((c) => c.name === name)?.description || name}
              </span>
              <div className="shrink-0 flex items-center gap-3">
                <button
                  type="button"
                  className="text-xs text-slack-accent hover:underline"
                  onClick={() => unhideDmChannel(name)}
                >
                  Unhide
                </button>
                <button
                  type="button"
                  className="text-xs text-red-400 hover:text-red-300 hover:underline"
                  onClick={() => deleteHiddenDmChannel(name)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
          {(settings.hiddenCollaborationChannelNames ?? []).map((name) => {
            const ch = channels.find((c) => c.name === name);
            const label =
              ch?.description?.trim() && !ch.description.startsWith('collab-')
                ? ch.description
                : name.replace(/^collab-/, '').slice(0, 8) || name;
            return (
              <div
                key={`collab-${name}`}
                className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
              >
                <span className="text-sm text-slack-text truncate" title={name}>
                  Collab: {label}
                </span>
                <div className="shrink-0 flex items-center gap-3">
                  <button
                    type="button"
                    className="text-xs text-slack-accent hover:underline"
                    onClick={() => unhideCollaborationChannel(name)}
                  >
                    Unhide
                  </button>
                  <button
                    type="button"
                    className="text-xs text-red-400 hover:text-red-300 hover:underline"
                    onClick={() => deleteHiddenCollaborationChannel(name)}
                  >
                    Delete
                  </button>
                </div>
              </div>
            );
          })}
          {(settings.hiddenAgentSidebarKeys ?? []).map((key) => {
            const label = key.includes(':') ? key.slice(key.indexOf(':') + 1) : key;
            return (
              <div
                key={`agk-${key}`}
                className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
              >
                <span className="text-sm text-slack-text truncate">
                  Agent shortcut: {label}
                </span>
                <div className="shrink-0 flex items-center gap-3">
                  <button
                    type="button"
                    className="text-xs text-slack-accent hover:underline"
                    onClick={() => unhideAgentShortcutKey(key)}
                  >
                    Unhide
                  </button>
                  <button
                    type="button"
                    className="text-xs text-red-400 hover:text-red-300 hover:underline"
                    onClick={() => deleteHiddenAgentShortcutKey(key)}
                  >
                    Delete
                  </button>
                </div>
              </div>
            );
          })}
          {(settings.hiddenAgentIdsForSidebar ?? []).map((id) => (
            <div
              key={`ag-${id}`}
              className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
            >
              <span className="text-sm text-slack-text truncate">
                Agent shortcut: {agents.find((a) => a.id === id)?.name || id}
              </span>
              <div className="shrink-0 flex items-center gap-3">
                <button
                  type="button"
                  className="text-xs text-slack-accent hover:underline"
                  onClick={() => unhideAgentShortcutId(id)}
                >
                  Unhide
                </button>
                <button
                  type="button"
                  className="text-xs text-red-400 hover:text-red-300 hover:underline"
                  onClick={() => deleteHiddenAgentShortcutId(id)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  </div>
            )}
            <IntegrationsSettingsTab hubHttp={hubHttp} isActive={activeTab === 'integrations'} />
            <AIProvidersSettingsTab hubHttp={hubHttp} isActive={activeTab === 'ai-providers'} />
            <DomainPacksSettingsTab hubHttp={hubHttp} isActive={activeTab === 'domain-packs'} />
            <SecuritySettingsTab hubHttp={hubHttp} isActive={activeTab === 'security'} />
            <AboutSettingsTab hubHttp={hubHttp} isActive={activeTab === 'about'} />
          </div>
        </div>
      </div>
    </div>
  );
}
