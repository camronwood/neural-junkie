import { describe, expect, it } from 'vitest';
import {
  capabilityOverrideAfterToggle,
  handoffNavigationTarget,
} from './capabilityPolicy';

describe('capabilityOverrideAfterToggle', () => {
  it('grants a denied sensitive capability sparsely', () => {
    expect(
      capabilityOverrideAfterToggle(
        {
          id: 'aws',
          qualified_id: 'aws/aws',
          exposure: 'sensitive',
        },
        { allow: [], deny: ['aws/aws'], effective: [] },
      ),
    ).toEqual({ allow: ['aws/aws'], deny: [] });
  });

  it('revokes an inherited safe capability', () => {
    expect(
      capabilityOverrideAfterToggle(
        {
          id: 'read',
          qualified_id: 'core/read',
          exposure: 'safe',
        },
        { allow: [], deny: [], effective: ['core/read'] },
      ),
    ).toEqual({ allow: [], deny: ['core/read'] });
  });
});

describe('handoffNavigationTarget', () => {
  it('uses the delegation channel when handoff starts', () => {
    expect(
      handoffNavigationTarget({
        handoff_event: 'handoff_started',
        handoff_channel: ' delegation-123 ',
        source_channel: 'general',
      }),
    ).toBe('delegation-123');
  });

  it('returns to the source channel when handoff finishes', () => {
    expect(
      handoffNavigationTarget({
        handoff_event: 'handoff_failed',
        handoff_channel: 'delegation-123',
        source_channel: 'general',
      }),
    ).toBe('general');
  });

  it('rejects missing or empty targets', () => {
    expect(handoffNavigationTarget({ handoff_event: 'handoff_completed', source_channel: ' ' })).toBeNull();
    expect(handoffNavigationTarget({ handoff_event: 'other', handoff_channel: 'x' })).toBeNull();
  });
});
