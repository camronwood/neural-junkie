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
  if (theme === 'roving') {
    mermaid.initialize({
      startOnLoad: false,
      theme: 'base',
      securityLevel: 'strict',
      fontFamily: 'ui-monospace, monospace',
      themeVariables: {
        darkMode: false,
        background: '#f9f7f2',
        primaryColor: '#f0ebe3',
        primaryTextColor: '#3b2f3d',
        primaryBorderColor: '#e5ddd4',
        lineColor: '#9b7fa8',
        secondaryColor: '#f5d5c3',
        tertiaryColor: '#ffffff',
        nodeTextColor: '#3b2f3d',
        mainBkg: '#ffffff',
        nodeBorder: '#9b7fa8',
        clusterBkg: '#f0ebe3',
        titleColor: '#856b91',
        edgeLabelBackground: '#f9f7f2',
      },
    });
    return;
  }
  if (theme === 'brand') {
    mermaid.initialize({
      startOnLoad: false,
      theme: 'base',
      securityLevel: 'strict',
      fontFamily: 'ui-monospace, monospace',
      themeVariables: {
        darkMode: true,
        background: '#1a161a',
        primaryColor: '#252028',
        primaryTextColor: '#ffffff',
        primaryBorderColor: '#2d262d',
        lineColor: '#f44a69',
        secondaryColor: '#120d11',
        tertiaryColor: '#252028',
        nodeTextColor: '#ffffff',
        mainBkg: '#252028',
        nodeBorder: '#f44a69',
        clusterBkg: '#120d11',
        titleColor: '#ff5a79',
        edgeLabelBackground: '#1a161a',
      },
    });
    return;
  }
  if (theme === 'retro') {
    mermaid.initialize({
      startOnLoad: false,
      theme: 'base',
      securityLevel: 'strict',
      fontFamily: 'ui-monospace, monospace',
      themeVariables: {
        darkMode: true,
        background: '#0c1a2e',
        primaryColor: '#142847',
        primaryTextColor: '#f1f5f9',
        primaryBorderColor: '#2a4a7a',
        lineColor: '#ffd447',
        secondaryColor: '#0a1220',
        tertiaryColor: '#142847',
        nodeTextColor: '#f1f5f9',
        mainBkg: '#142847',
        nodeBorder: '#ffd447',
        clusterBkg: '#0a1220',
        titleColor: '#3a86ff',
        edgeLabelBackground: '#0c1a2e',
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
