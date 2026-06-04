import { describe, expect, it } from 'vitest';

function matchesQuery(item: { id: string; label: string }, query: string): boolean {
  if (!query) return true;
  const hay = `${item.label} ${item.id}`.toLowerCase();
  return hay.includes(query);
}

describe('SearchableTimPicker filter', () => {
  const items = [
    { id: 'abc123', label: 'HIF12A plate 1' },
    { id: 'def456', label: 'Calibration' },
  ];

  it('matches label', () => {
    expect(matchesQuery(items[0], 'hif12a')).toBe(true);
    expect(matchesQuery(items[1], 'hif12a')).toBe(false);
  });

  it('matches id substring', () => {
    expect(matchesQuery(items[0], 'abc1')).toBe(true);
  });

  it('empty query matches all', () => {
    expect(matchesQuery(items[0], '')).toBe(true);
  });
});
