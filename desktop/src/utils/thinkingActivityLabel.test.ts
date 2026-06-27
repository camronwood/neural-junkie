import { describe, expect, it } from 'vitest';
import {
  THINKING_ACTIVITY_GENERATING_MUSIC,
  THINKING_ACTIVITY_REASONING,
  THINKING_ACTIVITY_USING_TOOL,
  THINKING_ACTIVITY_VERIFYING,
} from '../types/protocol';
import { formatThinkingActivityLabel, formatToolStepLabel } from './thinkingActivityLabel';

describe('formatThinkingActivityLabel', () => {
  it('formats tool activity with detail', () => {
    expect(formatThinkingActivityLabel(THINKING_ACTIVITY_USING_TOOL, 'read_file — package.json')).toBe(
      'is using read_file — package.json'
    );
  });

  it('formats verification', () => {
    expect(formatThinkingActivityLabel(THINKING_ACTIVITY_VERIFYING, 'npm run build')).toBe(
      'is verifying — npm run build'
    );
  });

  it('formats reasoning with detail', () => {
    expect(formatThinkingActivityLabel(THINKING_ACTIVITY_REASONING, 'Running npm run build…')).toBe(
      'is reasoning — Running npm run build…'
    );
  });

  it('formats music generation', () => {
    expect(formatThinkingActivityLabel(THINKING_ACTIVITY_GENERATING_MUSIC, 'lo-fi chill')).toBe(
      'is generating music — lo-fi chill'
    );
  });

  it('defaults to thinking', () => {
    expect(formatThinkingActivityLabel()).toBe('is thinking');
  });
});

describe('formatToolStepLabel', () => {
  it('prefers preview text', () => {
    expect(
      formatToolStepLabel({
        kind: 'start',
        name: 'read_file',
        preview: '[read_file] start: package.json',
      })
    ).toBe('[read_file] start: package.json');
  });
});
