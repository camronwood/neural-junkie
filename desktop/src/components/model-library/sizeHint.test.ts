import { describe, expect, it } from 'vitest';
import { actionLabelWithSize, estimateSizeHintFromName } from './sizeHint';

describe('sizeHint helpers', () => {
  it('estimates from parameter tags', () => {
    expect(estimateSizeHintFromName('llama3.1:8b')).toMatch(/GB/);
    expect(estimateSizeHintFromName('Qwen2.5-Coder-7B-Instruct')).toMatch(/GB/);
  });

  it('adds size to action labels', () => {
    expect(actionLabelWithSize('Install', '~4.7 GB')).toBe('Install · ~4.7 GB');
    expect(actionLabelWithSize('Download', 'Looking up size…')).toBe('Download');
  });
});
