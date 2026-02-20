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

  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    const w = saved ? parseInt(saved, 10) : DEFAULT_WIDTH;
    return (w >= MIN_WIDTH && w <= 400) ? w : DEFAULT_WIDTH;
  });
  const [isResizing, setIsResizing] = useState(false);
  const sidebarRef = useRef<HTMLDivElement>(null);

  // Separate channels by type
  const publicChannels = channels.filter(c => c.type === 'public' || !c.type);
  const customChannels = channels.filter(c => c.type === 'custom');
  const dmChannels = channels.filter(c => c.type === 'dm');

  // Build a set of agent IDs that already have an active DM
  const agentsWithDM = new Set(
    dmChannels.flatMap(c => c.agents?.map(a => a.id) ?? c.members ?? [])
  );

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
    const displayName = agent?.name ?? ch.name.replace(/^dm-[^-]+-/, '');
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
        <span className="ml-auto text-[10px] opacity-50">{agent.type}</span>
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
            {publicChannels.map(ch => (
              <ChannelItem key={ch.id} ch={ch} />
            ))}
            {customChannels.map(ch => (
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
            {dmChannels.map(ch => (
              <DMItem key={ch.id} ch={ch} />
            ))}

            {/* Agents without a DM yet */}
            {agents
              .filter(a => a.status === 'active' && !agentsWithDM.has(a.id))
              .map(agent => (
                <AgentDMEntry key={agent.id} agent={agent} />
              ))}
          </div>
        </div>
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
