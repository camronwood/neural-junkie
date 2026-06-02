import { describe, expect, it } from 'vitest';
import type { AgentInfo, Channel } from '../types/protocol';
import { buildSidebarDMRows } from './sidebarDmRows';

function agent(id: string, name: string): AgentInfo {
  return {
    id,
    name,
    type: 'backend',
    status: 'active',
    capabilities: [],
    created_at: '',
    last_active: '',
  };
}

function dmChannel(name: string, displayAgent: string): Channel {
  return {
    id: name,
    name,
    type: 'dm',
    description: '',
    project: '',
    created_at: '',
    agents: [agent('x', displayAgent)],
  };
}

describe('buildSidebarDMRows', () => {
  it('merges and sorts DMs and shortcuts alphabetically', () => {
    const rows = buildSidebarDMRows(
      [dmChannel('dm-user-bob', 'Bob')],
      [agent('a1', 'Alice'), agent('a2', 'Charlie')]
    );
    expect(rows.map((r) => (r.kind === 'channel' ? parseName(r) : r.agent.name))).toEqual([
      'Alice',
      'Bob',
      'Charlie',
    ]);
  });

  it('keeps stable order when shortcut becomes channel', () => {
    const bob = agent('b1', 'Bob');
    const before = buildSidebarDMRows([], [agent('a1', 'Alice'), bob]);
    const after = buildSidebarDMRows([dmChannel('dm-user-bob', 'Bob')], [agent('a1', 'Alice')]);
    expect(before.map(rowKey)).toEqual(['shortcut:a1', 'shortcut:b1']);
    expect(after.map(rowKey)).toEqual(['shortcut:a1', 'channel:dm-user-bob']);
    expect(before.map((r) => (r.kind === 'shortcut' ? r.agent.name : parseName(r)))).toEqual(
      after.map((r) => (r.kind === 'shortcut' ? r.agent.name : parseName(r)))
    );
  });
});

function parseName(row: ReturnType<typeof buildSidebarDMRows>[number]): string {
  return row.kind === 'channel' ? row.channel.agents?.[0]?.name ?? row.channel.name : row.agent.name;
}

function rowKey(row: ReturnType<typeof buildSidebarDMRows>[number]): string {
  return row.kind === 'channel' ? `channel:${row.channel.name}` : `shortcut:${row.agent.id}`;
}
