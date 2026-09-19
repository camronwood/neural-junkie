/** Module-level accept/dismiss counters for inline (ghost-text) completions. */

export type InlineCompletionMetricsSnapshot = {
  accepts: number;
  dismisses: number;
  partialAccepts: number;
  shown: number;
};

let accepts = 0;
let dismisses = 0;
let partialAccepts = 0;
let shown = 0;

export function RecordInlineCompletionAccept(): void {
  accepts += 1;
}

export function RecordInlineCompletionDismiss(): void {
  dismisses += 1;
}

export function RecordInlineCompletionPartialAccept(): void {
  partialAccepts += 1;
}

export function RecordInlineCompletionShown(): void {
  shown += 1;
}

export function getSnapshot(): InlineCompletionMetricsSnapshot {
  return { accepts, dismisses, partialAccepts, shown };
}

/** Test / debug helper. */
export function resetInlineCompletionMetrics(): void {
  accepts = 0;
  dismisses = 0;
  partialAccepts = 0;
  shown = 0;
}
