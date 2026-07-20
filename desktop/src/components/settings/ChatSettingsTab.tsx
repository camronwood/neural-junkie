import { shallow } from 'zustand/shallow';
import { useEffect, useState } from 'react';
import { useSettingsStore } from '../../stores/settingsStore';
import { useChatStore } from '../../stores/chatStore';
import { agentSidebarHideKey, parseDMDisplayName } from '../../utils/dmChannelDisplay';
import {
  loadConversationModeSetting,
  saveConversationModeSetting,
  type ConversationModeSetting,
} from '../../utils/conversationMode';
import type { SettingsTabProps } from './settingsShared';

function addUnique(items: string[] | undefined, value: string): string[] {
  const cur = items ?? [];
  return cur.includes(value) ? cur : [...cur, value];
}

function removeItem(items: string[] | undefined, value: string): string[] {
  return (items ?? []).filter((x) => x !== value);
}

export function ChatSettingsTab({ isActive }: SettingsTabProps) {
  const { settings, updateSettings } = useSettingsStore();
  const { channels, agents } = useChatStore(
    (s) => ({
      channels: s.channels,
      agents: s.agents,
    }),
    shallow
  );
  const [conversationMode, setConversationMode] = useState<ConversationModeSetting>(() =>
    loadConversationModeSetting()
  );

  useEffect(() => {
    if (!isActive) return;
    setConversationMode(loadConversationModeSetting());
  }, [isActive]);

  if (!isActive) return null;

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
    <div className="space-y-8">
      <div>
        <h3 className="text-lg font-semibold text-slack-text mb-2">Conversation mode</h3>
        <p className="text-sm text-slack-textMuted mb-3">
          Auto infers whether a message is discussion or hands-on code work. When Auto is unsure, the agent
          asks you in chat before using tools or editing files. Force Chat or Code here if you want a fixed
          override.
        </p>
        <div className="flex flex-wrap gap-2">
          {(
            [
              { id: 'auto', label: 'Auto', hint: 'Infer (clarify when unsure)' },
              { id: 'chat', label: 'Chat', hint: 'Discussion only' },
              { id: 'code', label: 'Code', hint: 'Workspace / tools' },
            ] as const
          ).map((opt) => {
            const selected = conversationMode === opt.id;
            return (
              <button
                key={opt.id}
                type="button"
                onClick={() => {
                  setConversationMode(opt.id);
                  saveConversationModeSetting(opt.id);
                }}
                className={`px-3 py-2 rounded border text-left text-sm transition-colors ${
                  selected
                    ? 'border-slack-accent bg-slack-accent/20 text-slack-text'
                    : 'border-slack-border bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
                title={opt.hint}
              >
                <span className="font-medium block">{opt.label}</span>
                <span className="text-[11px] opacity-80">{opt.hint}</span>
              </button>
            );
          })}
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-slack-text mb-2">User rules (markdown)</h3>
        <p className="text-sm text-slack-textMuted mb-2">
          Included on every message you send (main chat and threads). Agents treat this as your standing
          instructions.
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
          DM rows, collaborations, or agent shortcuts you hid. Use Unhide to restore them, or Delete to keep them
          off the sidebar permanently.
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
                  <span className="text-sm text-slack-text truncate">Agent shortcut: {label}</span>
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
  );
}
