/** Console logging only in Vite dev (not Vitest, not production). */
export function devLog(...args: unknown[]): void {
  if (import.meta.env.DEV && !import.meta.env.VITEST) {
    console.log(...args);
  }
}
