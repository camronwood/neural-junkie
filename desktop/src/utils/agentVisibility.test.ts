import { describe, expect, it } from 'vitest';
import {
  isConsultOnlyRepoAgent,
  isUserFacingAgent,
  isUserFacingCachedAgent,
  isWorkspaceIndexAgent,
} from './agentVisibility';
import type { AgentInfo, CachedAgentInfo } from '../types/protocol';

describe('agentVisibility', () => {
  it('hides consult-only repo agents', () => {
    const hidden: AgentInfo = {
      id: '1',
      name: '__index:myapp',
      type: 'repo',
      consult_only: true,
      expertise: [],
      status: 'active',
      model: '',
      is_paused: false,
    };
    expect(isConsultOnlyRepoAgent(hidden)).toBe(true);
    expect(isWorkspaceIndexAgent(hidden)).toBe(true);
    expect(isUserFacingAgent(hidden)).toBe(false);
  });

  it('hides cached index-* workspace indexes', () => {
    const cached: CachedAgentInfo = {
      type: 'repo',
      name: 'index-dickory-docs',
      path: '/Users/camronwood/development/projects/dickory-docs',
      last_used: '2026-09-22T10:00:00Z',
      cache_size: 1000,
      metadata: {},
    };
    expect(isWorkspaceIndexAgent(cached)).toBe(true);
    expect(isUserFacingCachedAgent(cached)).toBe(false);
  });

  it('shows visible repo agents', () => {
    const visible: AgentInfo = {
      id: '2',
      name: 'MyAppExpert',
      type: 'repo',
      expertise: [],
      status: 'active',
      model: '',
      is_paused: false,
    };
    expect(isUserFacingAgent(visible)).toBe(true);
    expect(
      isUserFacingCachedAgent({
        type: 'repo',
        name: 'MyAppExpert',
        path: '/tmp/app',
        last_used: '',
        cache_size: 1,
        metadata: {},
      })
    ).toBe(true);
  });
});
