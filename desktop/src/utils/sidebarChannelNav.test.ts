import { describe, expect, it } from 'vitest';
import { adjacentChannel, buildNavigableChannelList } from './sidebarChannelNav';
import type { Channel } from '../types/protocol';

const channels: Channel[] = [
  {
    id: '1',
    name: 'general',
    description: '',
    type: 'public',
    created: '',
    agents: [],
  },
  {
    id: '2',
    name: 'random',
    description: '',
    type: 'public',
    created: '',
    agents: [],
  },
];

describe('sidebarChannelNav', () => {
  it('builds ordered navigable list', () => {
    const list = buildNavigableChannelList(channels, []);
    expect(list.map((c) => c.name)).toEqual(['general', 'random']);
  });

  it('cycles channels with adjacentChannel', () => {
    const list = buildNavigableChannelList(channels, []);
    expect(adjacentChannel(list, 'general', 'next')?.name).toBe('random');
    expect(adjacentChannel(list, 'random', 'next')?.name).toBe('general');
    expect(adjacentChannel(list, 'general', 'prev')?.name).toBe('random');
  });
});
