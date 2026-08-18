import { describe, expect, it } from 'vitest';
import {
  isContinuationAvailable,
  getTerminalReason,
  OUTPUT_LENGTH_CONTINUATION_PROMPT,
} from '../types/protocol';

describe('response completion continuation metadata', () => {
  it('exposes continue only for length + continuation_available', () => {
    expect(
      isContinuationAvailable({
        terminal_reason: 'length',
        continuation_available: true,
      }),
    ).toBe(true);
    expect(
      isContinuationAvailable({
        terminal_reason: 'timeout',
        continuation_available: true,
      }),
    ).toBe(false);
    expect(
      isContinuationAvailable({
        terminal_reason: 'stop',
      }),
    ).toBe(false);
    expect(getTerminalReason({ terminal_reason: 'length' })).toBe('length');
  });

  it('uses a fixed continuation prompt', () => {
    expect(OUTPUT_LENGTH_CONTINUATION_PROMPT.toLowerCase()).toContain('continue');
    expect(OUTPUT_LENGTH_CONTINUATION_PROMPT.toLowerCase()).toContain('do not repeat');
  });
});
