/**
 * Mermaid 11+ treats many characters inside unquoted flowchart node labels as syntax
 * (@ → LINK_ID, / and . in paths, nested brackets, etc.). Agent diagrams often use
 * file paths and package names — quote unsafe labels before render.
 */

function escapeMermaidLabel(label: string): string {
  return label.replace(/"/g, '#quot;');
}

function labelNeedsQuotes(label: string): boolean {
  const trimmed = label.trim();
  if (!trimmed || trimmed.startsWith('"')) return false;
  return /[\/\\.@:\s()[\]{},|]/.test(trimmed) || trimmed.includes(']');
}

/** Quote [...] labels that contain unsafe characters and are not already quoted. */
function quoteUnsafeBracketLabels(source: string): string {
  return source.replace(
    /(\b[\w-]+)\[([^\]"\n]*)\]/g,
    (match, id: string, label: string) => {
      const trimmed = label.trim();
      if (!labelNeedsQuotes(trimmed)) return match;
      return `${id}["${escapeMermaidLabel(trimmed)}"]`;
    }
  );
}

/** Quote (...) node labels that contain unsafe characters. */
function quoteUnsafeParenLabels(source: string): string {
  return source.replace(
    /(\b[\w-]+)\(([^)"\n]*)\)/g,
    (match, id: string, label: string) => {
      const trimmed = label.trim();
      if (!labelNeedsQuotes(trimmed)) return match;
      return `${id}("${escapeMermaidLabel(trimmed)}")`;
    }
  );
}

const FLOWCHART_HEADER = /^(flowchart|graph)\s/im;

function flowchartBodyStart(source: string): number {
  const lines = source.split('\n');
  let offset = 0;
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed && !trimmed.startsWith('%%')) {
      return FLOWCHART_HEADER.test(trimmed) ? offset : -1;
    }
    offset += line.length + 1;
  }
  return -1;
}

/**
 * Normalize agent-generated flowcharts for Mermaid 11 parsers.
 * Safe to call on any diagram type — only mutates flowchart/graph bodies.
 */
export function normalizeMermaidSource(raw: string): string {
  const s = raw.replace(/\r\n/g, '\n');
  const trimmed = s.trim();
  if (!trimmed) return trimmed;

  if (flowchartBodyStart(trimmed) < 0) {
    return trimmed;
  }

  let out = trimmed;
  out = quoteUnsafeBracketLabels(out);
  out = quoteUnsafeParenLabels(out);
  return out;
}
