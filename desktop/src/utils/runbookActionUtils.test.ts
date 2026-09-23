import { describe, expect, it } from 'vitest';
import { defaultActionConfig, defaultActionSpec } from './runbookActionUtils';

describe('runbookActionUtils', () => {
  it('defaultActionSpec for email includes to/subject/body', () => {
    const spec = defaultActionSpec('email');
    expect(spec.type).toBe('email');
    expect(spec.config).toEqual({ to: '', subject: '', body: '' });
  });

  it('defaultActionSpec for slack_message includes channel and text', () => {
    const spec = defaultActionSpec('slack_message');
    expect(spec.type).toBe('slack_message');
    expect(spec.config).toEqual({ channel_id: '', text: '' });
  });

  it('defaultActionConfig returns empty object for unknown types', () => {
    expect(defaultActionConfig('unknown')).toEqual({});
  });
});
