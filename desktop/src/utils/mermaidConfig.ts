import mermaid from 'mermaid';
import type { ColorTheme } from '../stores/settingsStore';
import { normalizeMermaidSource } from './mermaidNormalize';

function mermaidInitForTheme(theme: ColorTheme): void {
  if (theme === 'flat') {
    mermaid.initialize({
      startOnLoad: false,
      theme: 'base',
      securityLevel: 'strict',
      fontFamily: 'ui-monospace, monospace',
      themeVariables: {
        darkMode: true,
        background: '#0d0d1a',
        primaryColor: '#16162d',
        primaryTextColor: '#e4e4ef',
        primaryBorderColor: '#3d3d5c',
        lineColor: '#a78bfa',
        secondaryColor: '#252b45',
        tertiaryColor: '#16162d',
        nodeTextColor: '#e4e4ef',
        mainBkg: '#16162d',
        nodeBorder: '#a78bfa',
        clusterBkg: '#252b45',
        titleColor: '#7dd3fc',
        edgeLabelBackground: '#0d0d1a',
      },
    });
    return;
  }
  mermaid.initialize({
    startOnLoad: false,
    theme: 'dark',
    securityLevel: 'strict',
    fontFamily: 'ui-monospace, monospace',
  });
}

mermaidInitForTheme('slack');

/** Re-initialize Mermaid when the user switches color theme. */
export function applyMermaidTheme(theme: ColorTheme): void {
  mermaidInitForTheme(theme);
}

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
