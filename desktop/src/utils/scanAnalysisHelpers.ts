/** Small display helpers shared by scan viewers. */

export function formatConcDisplay(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) return '—';
  if (value >= 100) return value.toFixed(1);
  if (value >= 1) return value.toFixed(2);
  return value.toFixed(4);
}
