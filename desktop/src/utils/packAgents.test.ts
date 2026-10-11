import { describe, expect, it } from 'vitest';
import {
  agentTypeFromSidebarKey,
  isHardDeletableAgentType,
  isPackSpecialistAgentType,
  isSoftDeleteOnlyAgentType,
} from './packAgents';

describe('packAgents', () => {
  it('recognizes pack specialists', () => {
    expect(isPackSpecialistAgentType('backend')).toBe(true);
    expect(isPackSpecialistAgentType('Frontend')).toBe(true);
    expect(isPackSpecialistAgentType('biology')).toBe(true);
    expect(isPackSpecialistAgentType('expert')).toBe(false);
  });

  it('soft-deletes pack/system agents only', () => {
    expect(isSoftDeleteOnlyAgentType('backend')).toBe(true);
    expect(isSoftDeleteOnlyAgentType('assistant')).toBe(true);
    expect(isSoftDeleteOnlyAgentType('cli')).toBe(true);
    expect(isSoftDeleteOnlyAgentType('expert')).toBe(false);
    expect(isSoftDeleteOnlyAgentType('repo')).toBe(false);
  });

  it('hard-deletes custom experts and repo agents', () => {
    expect(isHardDeletableAgentType('expert')).toBe(true);
    expect(isHardDeletableAgentType('repo')).toBe(true);
    expect(isHardDeletableAgentType('backend')).toBe(false);
  });

  it('parses type from sidebar key', () => {
    expect(agentTypeFromSidebarKey('backend:BackendEngineer')).toBe('backend');
    expect(agentTypeFromSidebarKey('expert:Guitar-Guru')).toBe('expert');
  });
});
