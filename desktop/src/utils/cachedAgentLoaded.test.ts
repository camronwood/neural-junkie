import { describe, expect, it } from 'vitest';
import type { AgentInfo, CachedAgentInfo } from '../types/protocol';
import { findLiveAgentForCached, isCachedAgentAlreadyLoaded } from './cachedAgentLoaded';

describe('isCachedAgentAlreadyLoaded', () => {
  const cached: CachedAgentInfo = {
    type: 'repo',
    name: 'neural-junkie-expert',
    path: '/Users/me/projects/neural-junkie',
    last_used: '',
    cache_size: 1,
    metadata: { agent_names: ['neural-junkie-expert'] },
  };

  it('returns true when a repo agent shares the same path', () => {
    const agents: AgentInfo[] = [
      {
        id: 'live-1',
        name: 'other-name',
        type: 'repo',
        expertise: [],
        status: 'active',
        model: 'qwen',
        repository_path: '/Users/me/projects/neural-junkie/',
        is_paused: false,
      } as AgentInfo,
    ];
    expect(isCachedAgentAlreadyLoaded(cached, agents, [])).toBe(true);
  });

  it('returns true when agent name is in loading set', () => {
    expect(isCachedAgentAlreadyLoaded(cached, [], ['neural-junkie-expert'])).toBe(true);
  });

  it('returns false when no matching live agent', () => {
    expect(isCachedAgentAlreadyLoaded(cached, [], [])).toBe(false);
  });

  it('returns true when a live expert shares the same name', () => {
    const expertCached: CachedAgentInfo = {
      type: 'expert',
      name: 'SwiftExpert',
      path: 'dm-user-swiftexpert',
      last_used: '',
      cache_size: 0,
      metadata: { expert_slug: 'ios' },
    };
    const agents: AgentInfo[] = [
      {
        id: 'live-expert',
        name: 'SwiftExpert',
        type: 'expert',
        expertise: [],
        status: 'active',
        model: 'gemma3:12b',
        is_paused: false,
      } as AgentInfo,
    ];
    expect(isCachedAgentAlreadyLoaded(expertCached, agents, [])).toBe(true);
  });
});

describe('findLiveAgentForCached', () => {
  it('returns the live agent for matching repo path', () => {
    const cached: CachedAgentInfo = {
      type: 'repo',
      name: 'neural-junkie Expert',
      path: '/tmp/repo',
      last_used: '',
      cache_size: 1,
      metadata: {},
    };
    const live: AgentInfo = {
      id: 'x',
      name: 'neural-junkie-expert',
      type: 'repo',
      expertise: [],
      status: 'active',
      model: '',
      repository_path: '/tmp/repo',
      is_paused: false,
    };
    expect(findLiveAgentForCached(cached, [live])?.id).toBe('x');
  });
});
