import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { devLog } from './devLog';

describe('devLog', () => {
  beforeEach(() => {
    vi.spyOn(console, 'log').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('does not log under Vitest even when DEV is true', () => {
    devLog('should stay quiet in tests');
    expect(console.log).not.toHaveBeenCalled();
  });
});
