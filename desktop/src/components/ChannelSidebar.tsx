import { useState, useEffect, useRef, type Ref } from 'react';
import { formatChord } from '../shortcuts/format';
import type { Channel, AgentInfo, ChannelType } from '../types/protocol';
import { getAgentColor } from '../types/protocol';
import { shallow } from 'zustand/shallow';
import { useChatStore } from '../stores/chatStore';
import { useSettingsStore } from '../stores/settingsStore';
import { parseDMDisplayName } from '../utils/dmChannelDisplay';
import { buildSidebarDMRows } from '../utils/sidebarDmRows';
import { channelSidebarLabel, isSlackHubChannelName, isSlackMirrorChannelName } from '../utils/slackChannelDisplay';
import {
  agentSidebarHideKey,
  isAgentShortcutDeleted,
  isAgentShownInSidebar,
  isAgentShortcutHidden,
  isDmChannelVisibleInSidebar,
  isSidebarChannelDeleted,
} from '../utils/sidebarVisibility';
import { shrinkablePanelStyle } from '../utils/panelLayout';
import { useSlackSidebarControls } from '../hooks/useSlackSidebarControls';
import { SlackSidebarChip } from './SlackSidebarChip';
import { SlackIcon } from './icons/SlackIcon';

interface ChannelSidebarProps {
  channels: Channel[];
  agents: AgentInfo[];
  searchInputRef?: Ref<HTMLInputElement>;
  onSwitchChannel: (channelName: string) => void;
  onCreateChannel: () => void;
  onCreateDM: (agentId: string) => void;
  onOpenNewDM: () => void;
  onDeleteChannel?: (channelName: string) => void;
  onOpenChannelInfo?: (ch: Channel) => void;
}

const MIN_WIDTH = 180;
const COMPACT_MIN_WIDTH = 140;
const DEFAULT_WIDTH = 220;
const STORAGE_KEY = 'channel-sidebar-width';

const CLI_AGENT_ICONS: Record<string, string> = {
  aider: 'A',
  amazonq: 'Q',
  amp: '⚡',
  claude: '🧠',
  codex: '◎',
  copilot: '⌾',
  crush: '◆',
  cursor: '⌖',
  droid: 'D',
  gemini: '✦',
  kiro: 'K',
  opencode: 'OC',
};

const CLI_AGENT_MATCHERS: Array<[keyof typeof CLI_AGENT_ICONS, string[]]> = [
  ['amazonq', ['amazonq', 'amazon-q', 'amazon q']],
  ['opencode', ['opencode', 'open-code', 'open code']],
  ['aider', ['aider']],
  ['amp', ['amp']],
  ['claude', ['claude']],
  ['codex', ['codex']],
  ['copilot', ['copilot', 'github-copilot']],
  ['crush', ['crush']],
  ['cursor', ['cursor']],
  ['droid', ['droid', 'factory']],
  ['gemini', ['gemini']],
  ['kiro', ['kiro']],
];

function getCliAgentIcon(agent?: AgentInfo): string | null {
  if (!agent) return null;
  const isCliAgent = agent.type === 'cli' || agent.ai_provider?.endsWith('-cli');
  if (!isCliAgent) return null;

  const identity = [agent.ai_provider, agent.ai_model, agent.model, agent.name]
    .filter(Boolean)
    .join(' ')
    .toLowerCase();
  const match = CLI_AGENT_MATCHERS.find(([, aliases]) =>
    aliases.some((alias) => identity.includes(alias))
  );
  return match ? CLI_AGENT_ICONS[match[0]] : 'CLI';
}

function AgentSidebarMarker({
  agent,
  fallbackColor,
}: {
  agent?: AgentInfo;
  fallbackColor: string;
}) {
  const cliIcon = getCliAgentIcon(agent);
  if (cliIcon) {
    return (
      <span
        className="inline-flex h-4 min-w-4 items-center justify-center rounded bg-white/10 px-0.5 text-[9px] leading-none text-white/85 ring-1 ring-white/10 flex-shrink-0"
        title={`${agent?.name ?? 'CLI agent'} icon`}
      >
        {cliIcon}
      </span>
    );
  }

  return (
    <span
      className="w-2 h-2 rounded-full flex-shrink-0"
      style={{ backgroundColor: fallbackColor }}
    />
  );
}

function maxSidebarWidth(): number {
  return Math.min(window.innerWidth * 0.4, 520);
}

function clampSidebarWidth(w: number): number {
  return Math.max(MIN_WIDTH, Math.min(maxSidebarWidth(), w));
}

/** DM rooms use the `dm-` slug; collaboration rooms use `collab-`. Coerce type when the hub omits or mis-stores `type` so rows appear under the right sidebar group. */
function normalizeChannelRow(ch: Channel): Channel {
  const name = (ch.name ?? '').trim();
  if (name.startsWith('dm-') && (ch.type === 'public' || !ch.type)) {
    return { ...ch, type: 'dm' as ChannelType };
  }
  if (
    name.startsWith('collab-') &&
    (ch.type === 'public' || !ch.type || ch.type === 'custom')
  ) {
    return { ...ch, type: 'collaboration' as ChannelType };
  }
  return ch;
}

export function ChannelSidebar({
  channels,
  agents,
  searchInputRef,
  onSwitchChannel,
  onCreateChannel,
  onCreateDM,
  onOpenNewDM,
  onDeleteChannel,
  onOpenChannelInfo,
}: ChannelSidebarProps) {
  const channelsNorm = channels.map(normalizeChannelRow);
  const { channel: activeChannel, unreadChannels, unreadCounts, channelThinkingAgents } = useChatStore(
    (s) => ({
      channel: s.channel,
      unreadChannels: s.unreadChannels,
      unreadCounts: s.unreadCounts,
      channelThinkingAgents: s.channelThinkingAgents,
    }),
    shallow
  );
  const sidebarAgentsVisible = useSettingsStore(s => s.layoutSettings.sidebarAgentsVisible);
  const settings = useSettingsStore(s => s.settings);
  const settingsLoaded = useSettingsStore(s => s.isLoaded);
  const updateSettings = useSettingsStore(s => s.updateSettings);
  const slackControls = useSlackSidebarControls();

  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    const w = saved ? parseInt(saved, 10) : DEFAULT_WIDTH;
    if (Number.isNaN(w)) return DEFAULT_WIDTH;
    return clampSidebarWidth(w);
  });
  const [isResizing, setIsResizing] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const sidebarRef = useRef<HTMLDivElement>(null);
  const resizeStartX = useRef(0);
  const resizeStartWidth = useRef(width);
  const currentWidthRef = useRef(width);

  useEffect(() => {
    currentWidthRef.current = width;
  }, [width]);
  const normalizedQuery = searchQuery.trim().toLowerCase();

  const hiddenDmSet = new Set(settings.hiddenDmChannelNames ?? []);
  const hiddenCollabSet = new Set(settings.hiddenCollaborationChannelNames ?? []);
  const isShortcutHidden = (agent: AgentInfo) =>
    settingsLoaded && isAgentShortcutHidden(settings, agent);
  const isShortcutDeleted = (agent: AgentInfo) =>
    settingsLoaded && isAgentShortcutDeleted(settings, agent);

  // Separate channels by type, sorted alphabetically for stable ordering
  const publicChannels = channelsNorm
    .filter(c => c.type === 'public' || !c.type)
    .sort((a, b) => a.name.localeCompare(b.name));
  const customChannels = channelsNorm
    .filter(c => c.type === 'custom' && !isSlackHubChannelName(c.name))
    .sort((a, b) => a.name.localeCompare(b.name));
  const slackChannels = channelsNorm
    .filter(c => isSlackHubChannelName(c.name))
    .sort((a, b) => channelSidebarLabel(a).localeCompare(channelSidebarLabel(b), undefined, { sensitivity: 'base' }));
  const collaborationChannels = channelsNorm
    .filter(c => c.type === 'collaboration')
    .filter(c => !(settingsLoaded && isSidebarChannelDeleted(settings, c)))
    .sort((a, b) => a.name.localeCompare(b.name));
  const dmChannels = channelsNorm
    .filter(c => c.type === 'dm')
    .filter(c => !(settingsLoaded && isSidebarChannelDeleted(settings, c)))
    .filter(c => isDmChannelVisibleInSidebar(c, agents))
    .sort((a, b) =>
      parseDMDisplayName(a).localeCompare(parseDMDisplayName(b), undefined, { sensitivity: 'base' })
    );

  const collaborationChannelLabel = (ch: Channel): string => {
    const desc = (ch.description || '').trim();
    if (desc && !desc.startsWith('collab-')) {
      return desc.length > 48 ? `${desc.slice(0, 45)}…` : desc;
    }
    const short = ch.name.replace(/^collab-/, '').slice(0, 8);
    return short ? `collab ${short}` : ch.name;
  };

  const filteredPublicChannels = publicChannels.filter((c) => {
    if (!normalizedQuery) return true;
    return (
      c.name.toLowerCase().includes(normalizedQuery) ||
      (c.description || '').toLowerCase().includes(normalizedQuery)
    );
  });
  const filteredCustomChannels = customChannels.filter((c) => {
    if (!normalizedQuery) return true;
    return (
      c.name.toLowerCase().includes(normalizedQuery) ||
      (c.description || '').toLowerCase().includes(normalizedQuery)
    );
  });
  const filteredSlackChannels = slackChannels.filter((c) => {
    if (!normalizedQuery) return true;
    const label = channelSidebarLabel(c).toLowerCase();
    return (
      label.includes(normalizedQuery) ||
      c.name.toLowerCase().includes(normalizedQuery) ||
      (c.description || '').toLowerCase().includes(normalizedQuery)
    );
  });
  const filteredCollaborationChannels = collaborationChannels.filter((c) => {
    const hidden = settingsLoaded && hiddenCollabSet.has(c.name);
    if (hidden && !normalizedQuery) return false;
    if (!normalizedQuery) return true;
    return (
      c.name.toLowerCase().includes(normalizedQuery) ||
      (c.description || '').toLowerCase().includes(normalizedQuery) ||
      collaborationChannelLabel(c).toLowerCase().includes(normalizedQuery)
    );
  });
  const filteredDMChannels = dmChannels.filter((c) => {
    const hidden = settingsLoaded && hiddenDmSet.has(c.name);
    if (hidden && !normalizedQuery) return false;
    if (!normalizedQuery) return true;
    return parseDMDisplayName(c).toLowerCase().includes(normalizedQuery);
  });

  // Build a set of agent IDs that already have an active DM.
  // Fallback to matching by parsed DM display name when restored sessions
  // don't include agents/members on DM channels.
  const agentsWithDM = new Set(
    dmChannels.flatMap(c => {
      const ids = c.agents?.map(a => a.id) ?? c.members ?? [];
      if (ids.length > 0) return ids;
      const inferredName = parseDMDisplayName(c);
      const matched = agents.find(a => a.name.toLowerCase() === inferredName.toLowerCase());
      return matched ? [matched.id] : [];
    })
  );

  const filteredAgentsWithoutDM = agents
    .filter(a => isAgentShownInSidebar(a) && !agentsWithDM.has(a.id) && !isShortcutDeleted(a))
    .filter(a => {
      const hidden = settingsLoaded && isShortcutHidden(a);
      if (hidden && !normalizedQuery) return false;
      if (!normalizedQuery) return true;
      return (
        a.name.toLowerCase().includes(normalizedQuery) ||
        a.type.toLowerCase().includes(normalizedQuery)
      );
    })
    .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }));

  const sidebarDMRows = buildSidebarDMRows(
    filteredDMChannels,
    sidebarAgentsVisible ? filteredAgentsWithoutDM : []
  );

  const hasSearchResults =
    filteredPublicChannels.length > 0 ||
    filteredCustomChannels.length > 0 ||
    filteredSlackChannels.length > 0 ||
    filteredCollaborationChannels.length > 0 ||
    sidebarDMRows.length > 0;

  useEffect(() => {
    const onWindowResize = () => {
      const clamped = clampSidebarWidth(currentWidthRef.current);
      if (clamped !== currentWidthRef.current) {
        setWidth(clamped);
        localStorage.setItem(STORAGE_KEY, String(clamped));
      }
    };
    window.addEventListener('resize', onWindowResize);
    return () => window.removeEventListener('resize', onWindowResize);
  }, []);

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      const delta = e.clientX - resizeStartX.current;
      const next = clampSidebarWidth(resizeStartWidth.current + delta);
      setWidth(next);
      currentWidthRef.current = next;
    };

    const onMouseUp = () => {
      if (!isResizing) return;
      setIsResizing(false);
      localStorage.setItem(STORAGE_KEY, String(currentWidthRef.current));
    };

    if (isResizing) {
      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    }

    return () => {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing]);

  const handleResizeStart = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    resizeStartX.current = e.clientX;
    resizeStartWidth.current = currentWidthRef.current;
    setIsResizing(true);
  };

  const TypingDots = ({ active }: { active?: boolean }) => (
    <span className="inline-flex ml-1 gap-[2px] items-center">
      <span className={`w-1 h-1 rounded-full animate-bounce ${active ? 'bg-white/80' : 'bg-slack-accent'}`} style={{ animationDelay: '0ms' }} />
      <span className={`w-1 h-1 rounded-full animate-bounce ${active ? 'bg-white/80' : 'bg-slack-accent'}`} style={{ animationDelay: '150ms' }} />
      <span className={`w-1 h-1 rounded-full animate-bounce ${active ? 'bg-white/80' : 'bg-slack-accent'}`} style={{ animationDelay: '300ms' }} />
    </span>
  );

  const hideDmFromSidebar = (channelName: string) => {
    const cur = settings.hiddenDmChannelNames ?? [];
    if (cur.includes(channelName)) return;
    void updateSettings({ hiddenDmChannelNames: [...cur, channelName] });
  };

  const hideCollabFromSidebar = (channelName: string) => {
    const cur = settings.hiddenCollaborationChannelNames ?? [];
    if (cur.includes(channelName)) return;
    void updateSettings({ hiddenCollaborationChannelNames: [...cur, channelName] });
    if (channelName === activeChannel) {
      onSwitchChannel('general');
    }
  };

  const hideAgentShortcutFromSidebar = (agent: AgentInfo) => {
    const key = agentSidebarHideKey(agent);
    const keys = settings.hiddenAgentSidebarKeys ?? [];
    if (keys.includes(key)) return;
    void updateSettings({ hiddenAgentSidebarKeys: [...keys, key] });
  };

  const SlackChannelItem = ({ ch }: { ch: Channel }) => {
    const isActive = ch.name === activeChannel;
    const isUnread = unreadChannels.has(ch.name);
    const unreadCount = unreadCounts[ch.name] ?? 0;
    const isTyping = (channelThinkingAgents.get(ch.name)?.size ?? 0) > 0;
    const displayName = channelSidebarLabel(ch);
    const rowClass = `flex-1 min-w-0 text-left px-2 py-1 rounded text-sm flex items-center gap-1.5 transition-colors ${
      isActive
        ? 'bg-slack-accent text-white font-semibold'
        : isUnread
          ? 'text-white font-semibold hover:bg-white/10'
          : 'text-slack-textMuted hover:bg-white/10 hover:text-white'
    }`;

    return (
      <div className="group flex items-center gap-0.5 w-full min-w-0">
        <button
          type="button"
          onClick={() => onSwitchChannel(ch.name)}
          className={rowClass}
          title={displayName}
        >
          <SlackIcon className="w-3 h-3 shrink-0 opacity-80" />
          <span className="truncate">{displayName}</span>
          {isTyping && <TypingDots active={isActive} />}
          {isUnread && !isActive && !isTyping && (
            <span
              className="ml-auto inline-flex h-5 min-w-[1.25rem] shrink-0 items-center justify-center rounded-full bg-slack-accent px-1.5 text-[10px] font-semibold leading-none text-white"
              aria-label={`${unreadCount} unread`}
            >
              {unreadCount > 99 ? '99+' : unreadCount > 0 ? unreadCount : ''}
            </span>
          )}
        </button>
        {onOpenChannelInfo && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onOpenChannelInfo(ch);
            }}
            className="shrink-0 opacity-0 group-hover:opacity-100 focus:opacity-100 px-1 py-0.5 text-[11px] text-white/35 hover:text-white/90 rounded hover:bg-white/10"
            title="Channel details"
            aria-label={`Details for ${displayName}`}
          >
            ⓘ
          </button>
        )}
      </div>
    );
  };

  const ChannelItem = ({ ch }: { ch: Channel }) => {
    const isActive = ch.name === activeChannel;
    const isUnread = unreadChannels.has(ch.name);
    const isTyping = (channelThinkingAgents.get(ch.name)?.size ?? 0) > 0;
    const displayName = channelSidebarLabel(ch, collaborationChannelLabel);
    const isSlackMirror = isSlackMirrorChannelName(ch.name);
    const rowClass = `flex-1 min-w-0 text-left px-2 py-1 rounded text-sm flex items-center transition-colors ${
      isActive
        ? 'bg-slack-accent text-white font-semibold'
        : isUnread
          ? 'text-white font-semibold hover:bg-white/10'
          : 'text-slack-textMuted hover:bg-white/10 hover:text-white'
    }`;
    const showDelete = ch.type === 'custom' && onDeleteChannel;
    const isCollab = ch.type === 'collaboration';
    const isHiddenCollabRow = isCollab && hiddenCollabSet.has(ch.name) && normalizedQuery.length > 0;

    return (
      <div className="group flex items-center gap-0.5 w-full min-w-0">
        <button
          type="button"
          onClick={() => onSwitchChannel(ch.name)}
          className={rowClass}
          title={
            isCollab
              ? `${displayName} (${ch.name})`
              : isSlackMirror
                ? displayName
                : (ch.description || ch.name)
          }
        >
          <span className="mr-1 opacity-60">{isCollab ? '🤝' : isSlackMirror ? '⎘' : '#'}</span>
          <span className="truncate">{displayName}</span>
          {isHiddenCollabRow && (
            <span className="text-[10px] uppercase text-white/50 shrink-0">hidden</span>
          )}
          {isTyping && <TypingDots active={isActive} />}
          {isUnread && !isActive && !isTyping && (
            <span className="ml-auto inline-block w-2 h-2 rounded-full bg-slack-accent flex-shrink-0" />
          )}
        </button>
        {onOpenChannelInfo && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onOpenChannelInfo(ch);
            }}
            className="shrink-0 opacity-0 group-hover:opacity-100 focus:opacity-100 px-1 py-0.5 text-[11px] text-white/35 hover:text-white/90 rounded hover:bg-white/10"
            title="Channel details"
            aria-label={`Details for ${ch.name}`}
          >
            ⓘ
          </button>
        )}
        {isCollab && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              hideCollabFromSidebar(ch.name);
            }}
            className="shrink-0 opacity-0 group-hover:opacity-100 focus:opacity-100 px-1 py-0.5 text-[11px] text-white/35 hover:text-white/90 rounded hover:bg-white/10"
            title="Hide from sidebar"
            aria-label={`Hide collaboration ${displayName} from sidebar`}
          >
            ×
          </button>
        )}
        {showDelete && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onDeleteChannel!(ch.name);
            }}
            className="shrink-0 opacity-0 group-hover:opacity-100 focus:opacity-100 px-1 py-0.5 text-[11px] text-white/35 hover:text-red-400 rounded hover:bg-white/10"
            title="Delete channel"
            aria-label={`Delete channel ${ch.name}`}
          >
            ×
          </button>
        )}
      </div>
    );
  };

  const DMItem = ({ ch }: { ch: Channel }) => {
    const isActive = ch.name === activeChannel;
    const isUnread = unreadChannels.has(ch.name);
    const isTyping = (channelThinkingAgents.get(ch.name)?.size ?? 0) > 0;
    const agent = ch.agents?.[0];
    const displayName = parseDMDisplayName(ch);
    const color = agent ? getAgentColor(agent.type) : '#a9b9ba';
    const isHiddenRow = hiddenDmSet.has(ch.name) && normalizedQuery.length > 0;

    return (
      <div className="group flex items-center gap-0.5 w-full min-w-0">
        <button
          type="button"
          onClick={() => onSwitchChannel(ch.name)}
          className={`flex-1 min-w-0 text-left px-2 py-1 rounded text-sm flex items-center gap-2 transition-colors ${
            isActive
              ? 'bg-slack-accent text-white font-semibold'
              : isUnread
              ? 'text-white font-semibold hover:bg-white/10'
              : 'text-slack-textMuted hover:bg-white/10 hover:text-white'
          }`}
          title={`DM with ${displayName}`}
        >
          <AgentSidebarMarker agent={agent} fallbackColor={color} />
          <span className="truncate">{displayName}</span>
          {isHiddenRow && (
            <span className="text-[10px] uppercase text-white/50 shrink-0">hidden</span>
          )}
          {isTyping && <TypingDots active={isActive} />}
          {isUnread && !isActive && !isTyping && (
            <span className="ml-auto inline-block w-2 h-2 rounded-full bg-slack-accent flex-shrink-0" />
          )}
        </button>
        {onOpenChannelInfo && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onOpenChannelInfo(ch);
            }}
            className="shrink-0 opacity-0 group-hover:opacity-100 focus:opacity-100 px-1 py-0.5 text-[11px] text-white/35 hover:text-white/90 rounded hover:bg-white/10"
            title="Channel details"
            aria-label={`Details for DM ${displayName}`}
          >
            ⓘ
          </button>
        )}
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            hideDmFromSidebar(ch.name);
          }}
          className="shrink-0 opacity-0 group-hover:opacity-100 focus:opacity-100 px-1 py-0.5 text-[11px] text-white/35 hover:text-white/90 rounded hover:bg-white/10"
          title="Hide from sidebar"
          aria-label={`Hide DM ${displayName} from sidebar`}
        >
          ×
        </button>
      </div>
    );
  };

  const AgentDMEntry = ({ agent }: { agent: AgentInfo }) => {
    const hasDM = agentsWithDM.has(agent.id);
    const dmChannel = dmChannels.find(c =>
      c.agents?.some(a => a.id === agent.id) || c.members?.includes(agent.id)
    );
    const isHiddenShortcut = isShortcutHidden(agent) && normalizedQuery.length > 0;

    return (
      <div className="group flex items-center gap-0.5 w-full min-w-0">
        <button
          type="button"
          onClick={() => {
            if (dmChannel) {
              onSwitchChannel(dmChannel.name);
            } else {
              onCreateDM(agent.id);
            }
          }}
          className={`flex-1 min-w-0 text-left px-2 py-1 rounded text-sm truncate flex items-center gap-2 transition-colors ${
            hasDM && dmChannel?.name === activeChannel
              ? 'bg-slack-accent text-white font-semibold'
              : 'text-slack-textMuted hover:bg-white/10 hover:text-white'
          }`}
          title={`Message ${agent.name}`}
        >
          <AgentSidebarMarker agent={agent} fallbackColor={getAgentColor(agent.type)} />
          <span className="truncate">{agent.name}</span>
          {agent.tool_count != null && agent.tool_count > 0 && (
            <span
              className="text-[10px] px-1 rounded bg-white/10 text-white/70 shrink-0"
              title={`${agent.tool_count} hub tool(s)`}
            >
              {agent.tool_count}
            </span>
          )}
          {isHiddenShortcut && (
            <span className="text-[10px] uppercase text-white/50 shrink-0">hidden</span>
          )}
        </button>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            hideAgentShortcutFromSidebar(agent);
          }}
          className="shrink-0 opacity-0 group-hover:opacity-100 focus:opacity-100 px-1 py-0.5 text-[11px] text-white/35 hover:text-white/90 rounded hover:bg-white/10"
          title="Hide shortcut from sidebar"
          aria-label={`Hide ${agent.name} from sidebar`}
        >
          ×
        </button>
      </div>
    );
  };

  return (
    <div
      ref={sidebarRef}
      className="relative bg-slack-bg border-r border-slack-border flex flex-col overflow-hidden select-none"
      style={shrinkablePanelStyle(width, COMPACT_MIN_WIDTH)}
    >
      {/* Header */}
      <div className="px-3 py-2 border-b border-white/10">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-sm font-bold text-white truncate">Neural Junkie</h2>
        </div>
        <div className="mt-2 flex items-center gap-1">
          <input
            ref={searchInputRef}
            type="text"
            data-testid="channel-sidebar-search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={`Search chats/channels (${formatChord('mod+0')})`}
            className="flex-1 min-w-0 px-2 py-1 rounded bg-nj-surface border border-white/10 text-xs text-slack-text placeholder:text-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
          />
          {normalizedQuery.length > 0 && (
            <button
              type="button"
              onClick={() => setSearchQuery('')}
              className="shrink-0 px-2 py-1 text-[10px] font-semibold uppercase tracking-wide rounded border border-white/20 text-white/70 hover:bg-white/10"
              title="Clear search (hidden chats reappear)"
            >
              Clear
            </button>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-2 py-2 space-y-4 text-sm">
        {/* Channels Section */}
        <div>
          <div className="flex items-center justify-between px-1 mb-1">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-white/50">
              Channels
            </span>
            <button
              onClick={onCreateChannel}
              className="text-white/40 hover:text-white text-lg leading-none"
              title="Create channel"
            >
              +
            </button>
          </div>
          <div className="space-y-0.5">
            {filteredPublicChannels.map(ch => (
              <ChannelItem key={ch.id} ch={ch} />
            ))}
            {filteredCustomChannels.map(ch => (
              <ChannelItem key={ch.id} ch={ch} />
            ))}
            {filteredCollaborationChannels.length > 0 && (
              <>
                <div className="px-1 pt-2 pb-0.5 text-[10px] font-semibold uppercase tracking-wider text-white/40">
                  Collaborations
                </div>
                {filteredCollaborationChannels.map(ch => (
                  <ChannelItem key={ch.id} ch={ch} />
                ))}
              </>
            )}
          </div>
        </div>

        {(slackControls.slackConnected || filteredSlackChannels.length > 0) && (
          <div className="mt-3">
            <div className="flex items-center justify-between gap-2 px-1 mb-1">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-white/50">
                Slack
              </span>
              <SlackSidebarChip controls={slackControls} />
            </div>
            <div className="space-y-0.5">
              {filteredSlackChannels.map((ch) => (
                <SlackChannelItem key={ch.id} ch={ch} />
              ))}
              {filteredSlackChannels.length === 0 && !normalizedQuery ? (
                <p className="px-2 py-1 text-[11px] text-white/40">
                  Bind a Slack channel in Settings to see it here.
                </p>
              ) : null}
            </div>
          </div>
        )}

        {/* Direct Messages Section */}
        <div>
          <div className="flex items-center justify-between px-1 mb-1">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-white/50">
              Direct Messages
            </span>
            <button
              type="button"
              onClick={onOpenNewDM}
              className="text-white/40 hover:text-white text-lg leading-none"
              title="New direct message with a new agent"
            >
              +
            </button>
          </div>
          <div className="space-y-0.5">
            {sidebarDMRows.map((row) =>
              row.kind === 'channel' ? (
                <DMItem key={row.channel.id} ch={row.channel} />
              ) : (
                <AgentDMEntry key={row.agent.id} agent={row.agent} />
              )
            )}
          </div>
        </div>

        {normalizedQuery && !hasSearchResults && (
          <div className="px-2 text-xs text-slack-textMuted">
            No chats or channels match this search. Clear the search box (Clear button above) to show all
            channels and direct messages again.
          </div>
        )}
      </div>

      {/* Resize handle — right edge */}
      <div
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize channel sidebar"
        onMouseDown={handleResizeStart}
        className={`absolute top-0 right-0 bottom-0 z-20 cursor-col-resize group ${
          isResizing ? 'bg-slack-accent/40' : ''
        }`}
        style={{ width: 6, marginRight: -3, touchAction: 'none' }}
      >
        <div className="absolute inset-0 bg-transparent group-hover:bg-slack-accent/30 transition-colors" />
        <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-1 h-10 bg-white/20 group-hover:bg-slack-accent rounded-full opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none" />
      </div>
    </div>
  );
}
