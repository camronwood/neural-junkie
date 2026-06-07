import type { AgentInfo, Channel } from '../types/protocol';
import type { Settings } from '../stores/settingsStore';
import { buildSidebarDMRows } from './sidebarDmRows';
import {
  isAgentShownInSidebar,
  isDmChannelVisibleInSidebar,
  isSidebarChannelDeleted,
} from './sidebarVisibility';
import { parseDMDisplayName } from './dmChannelDisplay';
import { channelSidebarLabel, isSlackHubChannelName } from './slackChannelDisplay';

export interface NavigableChannel {
  name: string;
  type: Channel['type'] | 'agent-shortcut';
}

function sortChannels(channels: Channel[]): Channel[] {
  return [...channels].sort((a, b) => a.name.localeCompare(b.name));
}

export function buildNavigableChannelList(
  channels: Channel[],
  agents: AgentInfo[],
  options?: {
    sidebarAgentsVisible?: boolean;
    hiddenDmNames?: string[];
    hiddenCollabNames?: string[];
    settingsLoaded?: boolean;
    settings?: Settings;
  }
): NavigableChannel[] {
  const settingsLoaded = options?.settingsLoaded ?? false;
  const settings = options?.settings;
  const hiddenDmSet = new Set(settings?.hiddenDmChannelNames ?? options?.hiddenDmNames ?? []);
  const hiddenCollabSet = new Set(
    settings?.hiddenCollaborationChannelNames ?? options?.hiddenCollabNames ?? []
  );

  const publicChannels = sortChannels(
    channels.filter((c) => c.type === 'public' || !c.type)
  );
  const customChannels = sortChannels(
    channels.filter((c) => c.type === 'custom' && !isSlackHubChannelName(c.name))
  );
  const slackChannels = [...channels.filter((c) => isSlackHubChannelName(c.name))].sort((a, b) =>
    channelSidebarLabel(a).localeCompare(channelSidebarLabel(b), undefined, { sensitivity: 'base' })
  );
  const collaborationChannels = sortChannels(
    channels
      .filter((c) => c.type === 'collaboration')
      .filter((c) => !(settingsLoaded && settings && isSidebarChannelDeleted(settings, c)))
      .filter((c) => !(settingsLoaded && hiddenCollabSet.has(c.name)))
  );
  const dmChannels = channels
    .filter((c) => c.type === 'dm')
    .filter((c) => !(settingsLoaded && settings && isSidebarChannelDeleted(settings, c)))
    .filter((c) => !(settingsLoaded && hiddenDmSet.has(c.name)))
    .filter((c) => isDmChannelVisibleInSidebar(c, agents))
    .sort((a, b) =>
      parseDMDisplayName(a).localeCompare(parseDMDisplayName(b), undefined, { sensitivity: 'base' })
    );

  const agentsWithDM = new Set(
    dmChannels.flatMap((c) => {
      const ids = c.agents?.map((a) => a.id) ?? c.members ?? [];
      if (ids.length > 0) return ids;
      const inferredName = parseDMDisplayName(c);
      const matched = agents.find((a) => a.name.toLowerCase() === inferredName.toLowerCase());
      return matched ? [matched.id] : [];
    })
  );

  const agentShortcuts = agents
    .filter((a) => isAgentShownInSidebar(a) && !agentsWithDM.has(a.id))
    .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }));

  const dmRows = buildSidebarDMRows(
    dmChannels,
    options?.sidebarAgentsVisible !== false ? agentShortcuts : []
  );

  const result: NavigableChannel[] = [];

  for (const ch of publicChannels) {
    result.push({ name: ch.name, type: ch.type ?? 'public' });
  }
  for (const ch of customChannels) {
    result.push({ name: ch.name, type: 'custom' });
  }
  for (const ch of slackChannels) {
    result.push({ name: ch.name, type: 'custom' });
  }
  for (const ch of collaborationChannels) {
    result.push({ name: ch.name, type: 'collaboration' });
  }
  for (const row of dmRows) {
    if (row.kind === 'channel') {
      result.push({ name: row.channel.name, type: 'dm' });
    } else {
      result.push({ name: row.agent.name, type: 'agent-shortcut' });
    }
  }

  return result;
}

export function adjacentChannel(
  list: NavigableChannel[],
  currentName: string,
  direction: 'prev' | 'next'
): NavigableChannel | null {
  if (list.length === 0) return null;
  const idx = list.findIndex((c) => c.name === currentName);
  if (idx === -1) {
    return direction === 'next' ? list[0] : list[list.length - 1];
  }
  const nextIdx = direction === 'next' ? (idx + 1) % list.length : (idx - 1 + list.length) % list.length;
  return list[nextIdx] ?? null;
}
