/** Boot-fix / build-failure signals that should route to implementers, not architects. */
export function hasBootFixRoutingSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  return /\b(not booting|won't boot|does not boot|make start-all|failed to scan|esbuild|vite dev|white screen|blank screen|no rule to make target)\b/i.test(
    text
  );
}
