import mermaid from 'mermaid';
import { normalizeMermaidSource } from './mermaidNormalize';

mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  securityLevel: 'loose',
  fontFamily: 'ui-monospace, monospace',
});

let renderCounter = 0;

// Serialize mermaid.render() calls -- it manipulates document.body globally
// and concurrent calls corrupt each other's temp containers.
let renderQueue: Promise<string> = Promise.resolve('');

/**
 * Render a mermaid diagram with a guaranteed-unique ID.
 * Calls are serialized so concurrent invocations don't collide.
 */
export function renderMermaidSvg(content: string): Promise<string> {
  renderQueue = renderQueue
    .catch(() => {})
    .then(async () => {
      const id = `mermaid-${++renderCounter}-${Math.random().toString(36).slice(2, 7)}`;
      document.getElementById('d' + id)?.remove();
      const { svg } = await mermaid.render(id, normalizeMermaidSource(content));
      return svg;
    });
  return renderQueue;
}

export default mermaid;
