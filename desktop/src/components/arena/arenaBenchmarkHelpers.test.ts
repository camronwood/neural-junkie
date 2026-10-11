import { describe, expect, it } from 'vitest';
import {
  benchmarkProgressLabel,
  formatLogicAccuracy,
  formatPassRate,
  isBenchmarkJobActive,
  rankingFromJob,
  toggleModelSelection,
} from './arenaBenchmarkHelpers';
import type { ArenaBenchmarkJob } from './arenaSidecarApi';

describe('arenaBenchmarkHelpers', () => {
  it('detects active jobs', () => {
    expect(isBenchmarkJobActive('queued')).toBe(true);
    expect(isBenchmarkJobActive('running')).toBe(true);
    expect(isBenchmarkJobActive('completed')).toBe(false);
    expect(isBenchmarkJobActive('cancelled')).toBe(false);
  });

  it('formats rates and logic scores', () => {
    expect(formatPassRate(0.75)).toBe('75%');
    expect(formatPassRate(undefined)).toBe('—');
    expect(formatLogicAccuracy(2, 4)).toBe('2/4');
    expect(formatLogicAccuracy(0, 0)).toBe('—');
  });

  it('builds progress labels', () => {
    const job: ArenaBenchmarkJob = {
      id: '1',
      status: 'running',
      suite: 'quick',
      models: ['a', 'b'],
      model_index: 0,
      current_model: 'a',
      current_challenge: 'logic-set',
    };
    expect(benchmarkProgressLabel(job)).toContain('Model 1/2');
    expect(benchmarkProgressLabel({ ...job, status: 'completed' })).toBe('Completed');
  });

  it('ranks from job results when ranking missing', () => {
    const ranks = rankingFromJob({
      id: '1',
      status: 'completed',
      suite: 'quick',
      models: ['b', 'a'],
      results: [
        { model: 'b', status: 'completed', pass_rate: 0.5, logic_correct: 1 },
        { model: 'a', status: 'completed', pass_rate: 1, logic_correct: 2 },
      ],
    });
    expect(ranks[0]?.model).toBe('a');
    expect(ranks[0]?.rank).toBe(1);
  });

  it('toggles model selection', () => {
    expect(toggleModelSelection(['a'], 'b')).toEqual(['a', 'b']);
    expect(toggleModelSelection(['a', 'b'], 'a')).toEqual(['b']);
  });
});
