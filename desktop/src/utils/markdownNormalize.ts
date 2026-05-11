/**
 * Normalizes sloppy LLM markdown so chat rendering matches GitHub-style fences.
 * Common issues: ``bash instead of ```bash, single-backtick close.
 */

/** Two-backtick fence opener: ``lang … ` → proper ``` fences */
export function normalizeMarkdownFences(raw: string): string {
  const lines = raw.split(/\r?\n/);
  const out: string[] = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];
    const langMatch = line.match(/^``([\w-]+)$/);
    const bareDouble = line === '``';

    if (langMatch || bareDouble) {
      out.push(bareDouble ? '```' : '```' + langMatch![1]);
      i++;
      let closed = false;
      while (i < lines.length) {
        const L = lines[i];
        if (L === '```') {
          out.push('```');
          i++;
          closed = true;
          break;
        }
        if (L === '`' || L === '``') {
          out.push('```');
          i++;
          closed = true;
          break;
        }
        out.push(L);
        i++;
      }
      if (!closed) {
        out.push('```');
      }
      continue;
    }

    out.push(line);
    i++;
  }

  return out.join('\n');
}

/** Same-line opener ```bash mv foo → split onto next line for the fence regex */
export function normalizeInlineFenceOpeners(raw: string): string {
  return raw.replace(/^```([\w-]+)\s+(.+)$/gm, (_m, lang: string, rest: string) => {
    const t = String(rest).trim();
    if (!t) return '```' + lang;
    return '```' + lang + '\n' + t;
  });
}

/** Prism / react-syntax-highlighter language id */
export function mapHighlighterLanguage(lang: string): string {
  const l = (lang || 'text').toLowerCase().trim();
  const map: Record<string, string> = {
    sh: 'bash',
    shell: 'bash',
    zsh: 'bash',
    console: 'bash',
    terminal: 'bash',
    ts: 'typescript',
    js: 'javascript',
    py: 'python',
  };
  return map[l] ?? l;
}

export function normalizeAgentMessageMarkdown(raw: string): string {
  let s = raw.replace(/\r\n/g, '\n');
  s = normalizeMarkdownFences(s);
  s = normalizeInlineFenceOpeners(s);
  return s;
}
