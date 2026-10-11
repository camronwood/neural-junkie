import { describe, expect, it } from 'vitest';
import { addToCappedSet } from './cappedSet';

describe('addToCappedSet', () => {
  it('prunes oldest entries when over cap', () => {
    const s = new Set<string>();
    addToCappedSet(s, 'a', 2);
    addToCappedSet(s, 'b', 2);
    addToCappedSet(s, 'c', 2);
    expect(s.size).toBe(2);
    expect(s.has('a')).toBe(false);
    expect(s.has('b')).toBe(true);
    expect(s.has('c')).toBe(true);
  });
});
