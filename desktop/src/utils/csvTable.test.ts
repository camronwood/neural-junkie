import { describe, expect, it } from 'vitest';
import { escapeCsvCell, isEditableCsvPath, parseCsvTable, serializeCsvTable } from './csvTable';

describe('csvTable', () => {
  it('detects editable csv paths', () => {
    expect(isEditableCsvPath('exports/qc.csv')).toBe(true);
    expect(isEditableCsvPath('reports/IL-6_summary_report.csv')).toBe(false);
    expect(isEditableCsvPath('data.txt')).toBe(false);
  });

  it('round-trips simple table', () => {
    const raw = 'a,b\n1,2\n';
    const rows = parseCsvTable(raw);
    expect(rows).toEqual([
      ['a', 'b'],
      ['1', '2'],
    ]);
    expect(serializeCsvTable(rows)).toBe('a,b\n1,2');
  });

  it('handles quoted commas', () => {
    const raw = 'name,value\n"foo, bar",1';
    const rows = parseCsvTable(raw);
    expect(rows[1][0]).toBe('foo, bar');
    expect(serializeCsvTable(rows)).toBe('name,value\n"foo, bar",1');
  });

  it('escapes quotes in cells', () => {
    expect(escapeCsvCell('say "hi"')).toBe('"say ""hi"""');
  });
});
