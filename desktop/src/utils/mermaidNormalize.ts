/**
 * Mermaid 11+ treats `@` inside unquoted flowchart node labels as link syntax (LINK_ID).
 * Agent plans often use @AgentName in labels — quote them before render.
 */

function escapeMermaidLabel(label: string): string {
  return label.replace(/"/g, '#quot;');
}

/** Quote [...] labels that contain @ and are not already quoted. */
function quoteBracketLabelsWithAt(source: string): string {
  return source.replace(
    /(\b[\w-]+)\[([^\]"\n]*@[^\]"\n]*)\]/g,
    (_match, id: string, label: string) => {
      const trimmed = label.trim();
      return `${id}["${escapeMermaidLabel(trimmed)}"]`;
    }
  );
}

/** Quote (...) node labels that contain @ (flowchart round nodes). */
function quoteParenLabelsWithAt(source: string): string {
  return source.replace(
    /(\b[\w-]+)\(([^)"\n]*@[^)"\n]*)\)/g,
    (_match, id: string, label: string) => {
      const trimmed = label.trim();
      return `${id}("${escapeMermaidLabel(trimmed)}")`;
    }
  );
}

const FLOWCHART_HEADER = /^(flowchart|graph)\s/im;

/**
 * Normalize agent-generated flowcharts for Mermaid 11 parsers.
 * Safe to call on any diagram type — only mutates flowchart/graph bodies.
 */
export function normalizeMermaidSource(raw: string): string {
  const s = raw.replace(/\r\n/g, '\n');
  const trimmed = s.trim();
  if (!trimmed) return trimmed;

  const firstLine = trimmed.split('\n')[0] ?? '';
  if (!FLOWCHART_HEADER.test(firstLine)) {
    return trimmed;
  }

  let out = trimmed;
  out = quoteBracketLabelsWithAt(out);
  out = quoteParenLabelsWithAt(out);
  return out;
}
