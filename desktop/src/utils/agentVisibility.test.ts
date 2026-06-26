import { describe, expect, it } from 'vitest';
import { isConsultOnlyRepoAgent, isUserFacingAgent } from './agentVisibility';
import type { AgentInfo } from '../types/protocol';

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
    expect(isUserFacingAgent(hidden)).toBe(false);
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
  });
});
