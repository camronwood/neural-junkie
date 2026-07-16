import { describe, expect, it } from 'vitest';
import {
  formatCostUsd,
  formatTokenCount,
  formatUsageTelemetryHeadline,
} from './inferenceUsageFormat';

describe('inferenceUsageFormat', () => {
  it('formats token counts', () => {
    expect(formatTokenCount(842)).toBe('842');
    expect(formatTokenCount(12400)).toBe('12.4k');
    expect(formatTokenCount(1_200_000)).toBe('1.2M');
  });

  it('formats usage headline', () => {
    expect(
      formatUsageTelemetryHeadline({ prompt_tokens: 12000, completion_tokens: 890, tok_per_s: 42.3 }),
    ).toBe('12.0k in · 890 out · 42 tok/s');
  });

  it('formats cost', () => {
    expect(formatCostUsd(0.1834)).toBe('$0.18');
    expect(formatCostUsd(0.002)).toBe('$0.0020');
  });
});
