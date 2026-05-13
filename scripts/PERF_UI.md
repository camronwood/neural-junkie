# Neural Junkie — UI performance profiling

## Dev perf marks (WebSocket + markdown parse)

1. Run the desktop app in dev (`make gui` or `npm run tauri:dev` from repo root).
2. In the webview DevTools console: `localStorage.setItem('nj-perf-marks', '1')` then reload.
3. Open **Performance** → record while agents stream / collaborate.
4. Look for measures: `ws.onmessage`, `messageContent.parse`.

Disable: `localStorage.removeItem('nj-perf-marks')`.

## Stream coalescing (automated)

From `neural-junkie/desktop`:

```bash
npm test -- --run src/stores/chatStreamCoalesce.test.ts
```

## Stress scenario (manual)

Use `/collaborate` with multiple agents on a busy channel; compare UI responsiveness before/after changes. With `nj-perf-marks` enabled, confirm fewer long tasks on the main thread during token bursts.
