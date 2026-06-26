import type { AgentInfo, Channel } from '../types/protocol';

/** Stable sidebar hide key (survives agent restarts that assign new UUIDs). */
export function agentSidebarHideKey(agent: Pick<AgentInfo, 'type' | 'name'>): string {
  return `${agent.type}:${agent.name}`;
}

export function parseDMDisplayName(dmChannel: Channel): string {
  const directAgent = dmChannel.agents?.[0]?.name;
  if (directAgent) return directAgent;

  const desc = dmChannel.description || '';
  const m = desc.match(/^Direct message with\s+(.+)$/i);
  if (m && m[1]) {
    return m[1].trim();
  }

  const raw = dmChannel.name.replace(/^dm-[^-]+-/, '');
  if (!raw) return dmChannel.name;
  if (raw.endsWith('expert')) {
    const stem = raw.slice(0, -6);
    return `${stem.charAt(0).toUpperCase()}${stem.slice(1)}Expert`;
  }
  return `${raw.charAt(0).toUpperCase()}${raw.slice(1)}`;
}

/** Agent the user is DMing — used for outbound metadata (not multi-agent routing). */
export function resolveDmPartnerAgent(
  channelName: string,
  channel: Pick<Channel, 'type' | 'agents' | 'description' | 'name'> | undefined,
  agents: AgentInfo[]
): AgentInfo | undefined {
  const name = (channelName ?? channel?.name ?? '').trim();
  if (!name.startsWith('dm-')) return undefined;
  const member = channel?.agents?.[0];
  if (member?.type) return member;
  const displayName = channel
    ? parseDMDisplayName(channel as Channel)
    : name.replace(/^dm-[^-]+-/, '').replace(/^./, (c) => c.toUpperCase());
  const needle = displayName.toLowerCase();
  return agents.find((a) => a.name.toLowerCase() === needle);
}

/** DM channel slugs that belong to this agent (by membership or display name). */
export function dmChannelNamesForAgent(channels: Channel[], agent: AgentInfo): string[] {
  const nameLower = agent.name.toLowerCase();
  return channels
    .filter((c) => c.type === 'dm')
    .filter((c) => {
      const ids = c.agents?.map((a) => a.id) ?? c.members ?? [];
      if (ids.includes(agent.id)) return true;
      return parseDMDisplayName(c).toLowerCase() === nameLower;
    })
    .map((c) => c.name);
}
