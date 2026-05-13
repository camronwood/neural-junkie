/**
 * Dev-only performance marks for profiling chat / markdown work.
 * Enable with `localStorage.setItem('nj-perf-marks', '1')` then open DevTools Performance.
 */
const ENABLED =
  import.meta.env.DEV &&
  typeof performance !== 'undefined' &&
  typeof window !== 'undefined' &&
  window.localStorage?.getItem('nj-perf-marks') === '1';

export function perfMarkStart(name: string) {
  if (!ENABLED) return;
  try {
    performance.clearMarks(`${name}:start`);
    performance.mark(`${name}:start`);
  } catch {
    /* ignore */
  }
}

export function perfMarkEnd(name: string) {
  if (!ENABLED) return;
  try {
    performance.clearMarks(`${name}:end`);
    performance.mark(`${name}:end`);
    try {
      performance.clearMeasures(name);
    } catch {
      /* ignore */
    }
    performance.measure(name, `${name}:start`, `${name}:end`);
  } catch {
    /* ignore */
  }
}
