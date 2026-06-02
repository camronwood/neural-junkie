import type { AgentInfo, Channel } from '../types/protocol';
import { parseDMDisplayName } from './dmChannelDisplay';

export type SidebarDMRow =
  | { kind: 'channel'; channel: Channel }
  | { kind: 'shortcut'; agent: AgentInfo };

export function dmDisplayName(row: SidebarDMRow): string {
  return row.kind === 'channel' ? parseDMDisplayName(row.channel) : row.agent.name;
}

export function buildSidebarDMRows(
  dmChannels: Channel[],
  agentShortcuts: AgentInfo[]
): SidebarDMRow[] {
  const rows: SidebarDMRow[] = [
    ...dmChannels.map((channel) => ({ kind: 'channel' as const, channel })),
    ...agentShortcuts.map((agent) => ({ kind: 'shortcut' as const, agent })),
  ];
  return rows.sort((a, b) =>
    dmDisplayName(a).localeCompare(dmDisplayName(b), undefined, { sensitivity: 'base' })
  );
}
