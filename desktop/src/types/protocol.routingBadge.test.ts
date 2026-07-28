import { describe, expect, it } from 'vitest';
import { formatRoutingBadgeLabel, type RoutingMeta } from './protocol';

describe('formatRoutingBadgeLabel', () => {
  it('prefers executed knowledge over planned targets', () => {
    const meta: RoutingMeta = {
      model: 'qwen2.5-coder:14b',
      knowledge_targets: ['prior_reference', 'codebase'],
      knowledge_executed: ['codebase'],
    };
    expect(formatRoutingBadgeLabel(meta)).toBe('qwen2.5-coder:14b · codebase');
  });

  it('falls back to planned targets when nothing executed', () => {
    const meta: RoutingMeta = {
      model: 'qwen3.5:9b',
      knowledge_targets: ['prior_reference', 'codebase'],
    };
    expect(formatRoutingBadgeLabel(meta)).toBe('qwen3.5:9b · prior_reference+codebase');
  });

  it('falls back to source when retrieval is empty', () => {
    const meta: RoutingMeta = {
      model: 'qwen3.5:9b',
      source: 'local_model',
      knowledge_route: 'general',
    };
    expect(formatRoutingBadgeLabel(meta)).toBe('qwen3.5:9b · local_model');
  });
});
