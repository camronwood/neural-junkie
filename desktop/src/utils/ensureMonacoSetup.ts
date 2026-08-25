/** Lazy-load Monaco workers + loader config (avoid ~6MB+ on cold start). */
let setupPromise: Promise<void> | null = null;

export function ensureMonacoSetup(): Promise<void> {
  if (!setupPromise) {
    setupPromise = import('../monacoSetup').then(() => undefined);
  }
  return setupPromise;
}
