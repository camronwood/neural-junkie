import { parseContentParts } from '../utils/markdownParts';

self.onmessage = (ev: MessageEvent<string>) => {
  try {
    const parts = parseContentParts(ev.data);
    self.postMessage(parts);
  } catch (err) {
    console.error('[markdownParts.worker]', err);
    try {
      self.postMessage(parseContentParts(ev.data));
    } catch {
      self.postMessage([{ type: 'text' as const, content: ev.data }]);
    }
  }
};

export {};
