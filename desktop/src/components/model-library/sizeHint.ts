/** Rough Q4-class download size from a parameter tag like `8b` / `:14b`. */
export function estimateSizeHintFromName(name: string): string | undefined {
  const lower = name.toLowerCase();
  const m =
    lower.match(/(?:^|[:\-_/\s])(\d+(?:\.\d+)?)b(?:\b|[_-]|$)/) ||
    lower.match(/(\d+(?:\.\d+)?)b-instruct/) ||
    lower.match(/(\d+(?:\.\d+)?)b/);
  if (!m) return undefined;
  const params = parseFloat(m[1]);
  if (!Number.isFinite(params) || params <= 0 || params > 1000) return undefined;
  const gb = params * 0.55;
  if (gb < 1) return `~${Math.max(0.1, Math.round(gb * 10) / 10)} GB`;
  if (gb < 10) return `~${gb.toFixed(1)} GB`;
  return `~${Math.round(gb)} GB`;
}

export function formatSizeBytes(n?: number): string | undefined {
  if (!n || !Number.isFinite(n) || n <= 0) return undefined;
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  if (i === 0) return `${Math.round(v)} B`;
  if (v >= 10) return `~${Math.round(v)} ${units[i]}`;
  return `~${v.toFixed(1)} ${units[i]}`;
}

export function actionLabelWithSize(label: string, sizeHint?: string): string {
  const size = sizeHint?.trim();
  if (!size || size.startsWith('Looking up')) return label;
  if (label.includes(size)) return label;
  return `${label} · ${size}`;
}
