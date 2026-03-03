import { useState, useEffect, useRef } from 'react';
import type { Channel, AgentInfo } from '../types/protocol';
import { getAgentColor } from '../types/protocol';
import { useChatStore } from '../stores/chatStore';

interface ChannelSidebarProps {
  channels: Channel[];
  agents: AgentInfo[];
  onSwitchChannel: (channelName: string) => void;
  onCreateChannel: () => void;
  onCreateDM: (agentId: string) => void;
}

const MIN_WIDTH = 180;
const DEFAULT_WIDTH = 220;
const STORAGE_KEY = 'channel-sidebar-width';

export function ChannelSidebar({
  channels,
  agents,
  onSwitchChannel,
  onCreateChannel,
  onCreateDM,
}: ChannelSidebarProps) {
  const { channel: activeChannel, unreadChannels, channelThinkingAgents } = useChatStore();

  const parseDMDisplayName = (dmChannel: Channel): string => {
    const directAgent = dmChannel.agents?.[0]?.name;
    if (directAgent) return directAgent;

    // Preferred fallback: the server-provided description preserves casing.
    const desc = dmChannel.description || '';
    const m = desc.match(/^Direct message with\s+(.+)$/i);
    if (m && m[1]) {
      return m[1].trim();
    }

    // Last fallback: derive from channel slug, then normalize common expert suffixes.
    const raw = dmChannel.name.replace(/^dm-[^-]+-/, '');
    if (!raw) return dmChannel.name;
    if (raw.endsWith('expert')) {
      const stem = raw.slice(0, -6);
      return `${stem.charAt(0).toUpperCase()}${stem.slice(1)}Expert`;
    }
    return `${raw.charAt(0).toUpperCase()}${raw.slice(1)}`;
  };

  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    const w = saved ? parseInt(saved, 10) : DEFAULT_WIDTH;
    return (w >= MIN_WIDTH && w <= 400) ? w : DEFAULT_WIDTH;
  });
  const [isResizing, setIsResizing] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const sidebarRef = useRef<HTMLDivElement>(null);
  const normalizedQuery = searchQuery.trim().toLowerCase();

  // Separate channels by type, sorted alphabetically for stable ordering
  const publicChannels = channels
    .filter(c => c.type === 'public' || !c.type)
    .sort((a, b) => a.name.localeCompare(b.name));
  const customChannels = channels
    .filter(c => c.type === 'custom')
    .sort((a, b) => a.name.localeCompare(b.name));
  const dmChannels = channels
    .filter(c => c.type === 'dm')
    .sort((a, b) => a.name.localeCompare(b.name));

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
  const filteredDMChannels = dmChannels.filter((c) => {
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
    .filter(a => a.status === 'active' && !agentsWithDM.has(a.id))
    .filter(a => {
      if (!normalizedQuery) return true;
      return (
        a.name.toLowerCase().includes(normalizedQuery) ||
        a.type.toLowerCase().includes(normalizedQuery)
      );
    });

  const hasSearchResults =
    filteredPublicChannels.length > 0 ||
    filteredCustomChannels.length > 0 ||
    filteredDMChannels.length > 0 ||
    filteredAgentsWithoutDM.length > 0;

  // Resize drag handling
  useEffect(() => {
    if (!isResizing) return;

    const onMouseMove = (e: MouseEvent) => {
      const newWidth = Math.max(MIN_WIDTH, Math.min(400, e.clientX));
      setWidth(newWidth);
    };

    const onMouseUp = () => {
      setIsResizing(false);
      localStorage.setItem(STORAGE_KEY, String(width));
    };

    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('mouseup', onMouseUp);
    return () => {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    };
  }, [isResizing, width]);

  const TypingDots = ({ active }: { active?: boolean }) => (
    <span className="inline-flex ml-1 gap-[2px] items-center">
      <span className={`w-1 h-1 rounded-full animate-bounce ${active ? 'bg-white/80' : 'bg-slack-accent'}`} style={{ animationDelay: '0ms' }} />
      <span className={`w-1 h-1 rounded-full animate-bounce ${active ? 'bg-white/80' : 'bg-slack-accent'}`} style={{ animationDelay: '150ms' }} />
      <span className={`w-1 h-1 rounded-full animate-bounce ${active ? 'bg-white/80' : 'bg-slack-accent'}`} style={{ animationDelay: '300ms' }} />
    </span>
  );

  const ChannelItem = ({ ch }: { ch: Channel }) => {
    const isActive = ch.name === activeChannel;
    const isUnread = unreadChannels.has(ch.name);
    const isTyping = (channelThinkingAgents.get(ch.name)?.size ?? 0) > 0;
    return (
      <button
        onClick={() => onSwitchChannel(ch.name)}
        className={`w-full text-left px-2 py-1 rounded text-sm flex items-center transition-colors ${
          isActive
            ? 'bg-slack-accent text-white font-semibold'
            : isUnread
            ? 'text-white font-semibold hover:bg-white/10'
            : 'text-slack-textMuted hover:bg-white/10 hover:text-white'
        }`}
        title={ch.description || ch.name}
      >
        <span className="mr-1 opacity-60">#</span>
        <span className="truncate">{ch.name}</span>
        {isTyping && <TypingDots active={isActive} />}
        {isUnread && !isActive && !isTyping && (
          <span className="ml-auto inline-block w-2 h-2 rounded-full bg-slack-accent flex-shrink-0" />
        )}
      </button>
    );
  };

  const DMItem = ({ ch }: { ch: Channel }) => {
    const isActive = ch.name === activeChannel;
    const isUnread = unreadChannels.has(ch.name);
    const isTyping = (channelThinkingAgents.get(ch.name)?.size ?? 0) > 0;
    const agent = ch.agents?.[0];
    const displayName = parseDMDisplayName(ch);
    const color = agent ? getAgentColor(agent.type) : '#a9b9ba';

    return (
      <button
        onClick={() => onSwitchChannel(ch.name)}
        className={`w-full text-left px-2 py-1 rounded text-sm flex items-center gap-2 transition-colors ${
          isActive
            ? 'bg-slack-accent text-white font-semibold'
            : isUnread
            ? 'text-white font-semibold hover:bg-white/10'
            : 'text-slack-textMuted hover:bg-white/10 hover:text-white'
        }`}
        title={`DM with ${displayName}`}
      >
        <span
          className="w-2 h-2 rounded-full flex-shrink-0"
          style={{ backgroundColor: color }}
        />
        <span className="truncate">{displayName}</span>
        {isTyping && <TypingDots active={isActive} />}
        {isUnread && !isActive && !isTyping && (
          <span className="ml-auto inline-block w-2 h-2 rounded-full bg-slack-accent flex-shrink-0" />
        )}
      </button>
    );
  };

  const AgentDMEntry = ({ agent }: { agent: AgentInfo }) => {
    const hasDM = agentsWithDM.has(agent.id);
    const dmChannel = dmChannels.find(c =>
      c.agents?.some(a => a.id === agent.id) || c.members?.includes(agent.id)
    );

    return (
      <button
        onClick={() => {
          if (dmChannel) {
            onSwitchChannel(dmChannel.name);
          } else {
            onCreateDM(agent.id);
          }
        }}
        className={`w-full text-left px-2 py-1 rounded text-sm truncate flex items-center gap-2 transition-colors ${
          hasDM && dmChannel?.name === activeChannel
            ? 'bg-slack-accent text-white font-semibold'
            : 'text-slack-textMuted hover:bg-white/10 hover:text-white'
        }`}
        title={`Message ${agent.name}`}
      >
        <span
          className="w-2 h-2 rounded-full flex-shrink-0"
          style={{ backgroundColor: getAgentColor(agent.type) }}
        />
        {agent.name}
        <span className="ml-auto text-xs opacity-50">{agent.type}</span>
      </button>
    );
  };

  return (
    <div
      ref={sidebarRef}
      className="flex-shrink-0 bg-[#1a1d21] border-r border-slack-border flex flex-col overflow-hidden select-none"
      style={{ width: `${width}px` }}
    >
      {/* Header */}
      <div className="px-3 py-2 border-b border-white/10">
        <h2 className="text-sm font-bold text-white truncate">Neural Junkie</h2>
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search chats/channels..."
          className="mt-2 w-full px-2 py-1 rounded bg-[#0f1115] border border-white/10 text-xs text-white placeholder:text-white/40 focus:outline-none focus:ring-1 focus:ring-slack-accent"
        />
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
          </div>
        </div>

        {/* Direct Messages Section */}
        <div>
          <div className="flex items-center justify-between px-1 mb-1">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-white/50">
              Direct Messages
            </span>
          </div>
          <div className="space-y-0.5">
            {/* Active DM channels first */}
            {filteredDMChannels.map(ch => (
              <DMItem key={ch.id} ch={ch} />
            ))}

            {/* Agents without a DM yet */}
            {filteredAgentsWithoutDM.map(agent => (
              <AgentDMEntry key={agent.id} agent={agent} />
            ))}
          </div>
        </div>

        {normalizedQuery && !hasSearchResults && (
          <div className="px-2 text-xs text-slack-textMuted">No chats or channels found.</div>
        )}
      </div>

      {/* Resize handle */}
      <div
        className="absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-slack-accent/50 transition-colors"
        onMouseDown={() => setIsResizing(true)}
        style={{ zIndex: 10 }}
      />
    </div>
  );
}
