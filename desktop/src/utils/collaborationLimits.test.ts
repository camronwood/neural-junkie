import { describe, expect, it } from 'vitest';
import { MAX_COLLAB_AGENTS } from './collaborationLimits';

describe('MAX_COLLAB_AGENTS', () => {
  it('caps runbook agent selection at 3', () => {
    const pool = ['a1', 'a2', 'a3', 'a4', 'a5'];
    const picked = pool.slice(0, Math.min(MAX_COLLAB_AGENTS, pool.length));
    expect(picked).toHaveLength(3);
    expect(MAX_COLLAB_AGENTS).toBe(3);
  });
});
