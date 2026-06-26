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

const IMAGE_FILE_EXT = /\.(png|jpe?g|gif|webp|svg|bmp|ico)$/i;

/**
 * When a line is only an absolute path to an image file (common when an agent
 * "saved" a file and pasted the path), rewrite to markdown image syntax so the
 * chat can render a preview (see MessageContent + resolveChatImageSrc).
 * Skips lines that look like inline code or already contain markdown images.
 */
export function promoteStandaloneImageFilePaths(text: string): string {
  return text
    .split(/\r?\n/)
    .map((line) => {
      const t = line.trim();
      if (!t) return line;
      if (t.includes('`')) return line;
      if (/^!\[/.test(t)) return line;

      const abs = t.startsWith('/') || /^[A-Za-z]:[\\/]/.test(t);
      if (!abs || !IMAGE_FILE_EXT.test(t)) return line;
      if (/\s/.test(t)) return line;

      return `![](${t})`;
    })
    .join('\n');
}

type FencePart = { type: 'code' | 'text'; content: string };

/** Split markdown so prose normalizers skip fenced code blocks. */
function splitPreservingCodeFences(raw: string): FencePart[] {
  const parts: FencePart[] = [];
  const re = /```[\s\S]*?```/g;
  let last = 0;
  let match: RegExpExecArray | null;
  while ((match = re.exec(raw)) !== null) {
    if (match.index > last) {
      parts.push({ type: 'text', content: raw.slice(last, match.index) });
    }
    parts.push({ type: 'code', content: match[0] });
    last = match.index + match[0].length;
  }
  if (last < raw.length) {
    parts.push({ type: 'text', content: raw.slice(last) });
  }
  if (parts.length === 0) {
    parts.push({ type: 'text', content: raw });
  }
  return parts;
}

/**
 * LLMs often emit headings, HRs, and lists inline in one paragraph.
 * Insert block breaks so marked/GFM can render articles and long replies cleanly.
 */
export function normalizeProseMarkdownBlocks(raw: string): string {
  return splitPreservingCodeFences(raw)
    .map((part) => (part.type === 'code' ? part.content : normalizeProseText(part.content)))
    .join('');
}

/** Agent review / report section labels often glued to titles or prior sentences. */
const REVIEW_SECTION_LABEL =
  /(?:^|\s)((?:What went wrong|What could have been better|What (?:worked|didn't work|did not work)|Recommendation(?:s)?|Summary|Root cause|Next steps|Key (?:findings|points)|Action items|Follow[- ]up)(?:[^:\n]{0,40})?):\s+/gi;

function normalizeProseText(text: string): string {
  let s = text;

  // "Review of X Response What went wrong:" → title + first section
  s = s.replace(
    /^(Review of\s+.+?)\s+(What went wrong):\s*/i,
    '## $1\n\n### What went wrong\n\n'
  );

  // Horizontal rules on their own line
  s = s.replace(/\s+---\s+/g, '\n\n---\n\n');
  s = s.replace(/\s+\*\*\*\s+/g, '\n\n***\n\n');

  // Headings after sentence punctuation
  s = s.replace(/([.!?])\s+(#{1,6}\s+)/g, '$1\n\n$2');

  // Headings mid-line (e.g. "... manner. --- ### Title")
  s = s.replace(/([^\n#])\s+(#{1,6}\s+\S)/g, '$1\n\n$2');

  // Review section labels ("What could have been better:", "Recommendation:", …)
  s = s.replace(REVIEW_SECTION_LABEL, (_m, label: string) => `\n\n### ${label.trim()}\n\n`);

  // Sentence-case section labels after period ("Details. Background: The …")
  s = s.replace(
    /([.!?])\s+([A-Z][A-Za-z]+(?:\s+[a-z]+){1,6}):\s+(?=[A-Z"'])/g,
    '$1\n\n### $2\n\n'
  );

  // Parenthesized inline enumerations: "(1) foo, (2) bar" → markdown ordered list
  s = s.replace(/\s+\((\d+)\)\s+/g, '\n\n$1. ');

  // Numbered list after a heading line
  s = s.replace(/(#{1,6}\s+[^\n]+)\s+(\d+\.\s+)/g, '$1\n\n$2');

  // Numbered list after sentence end or colon intro ("Workspace: 1. First step")
  s = s.replace(/([.!?])\s+(?=\d+\.\s+)/g, '$1\n\n');
  s = s.replace(/:\s+(?=\d+\.\s+)/g, ':\n\n');

  // Numbered list after closing paren ("(admin.google.com) 2. Next item")
  s = s.replace(/\)\s+(?=\d+\.\s+)/g, ')\n\n');

  // Subsequent numbered items glued to prior list text (2., 3., … — avoids "Section 1. The")
  s = s.replace(/(\S)\s+(?=[2-9]\d*\.\s+)/g, '$1\n\n');

  // Sub-bullets after list-item colons: "1. Foo: - Bar"
  s = s.replace(/:\s+-\s+/g, ':\n\n- ');

  // Sub-bullets after closing paren: "(admin.google.com) - Navigate"
  s = s.replace(/\)\s+-\s+/g, ')\n\n- ');

  // Inline dash sub-bullets mid paragraph: "… - Log into … - Navigate"
  s = s.replace(/\s+-\s+(?=[A-Z])/g, '\n\n- ');

  // Bold subsection labels inline: "#### Benefits 1. Enhanced"
  s = s.replace(/(#{4,6}\s+[A-Za-z][^\n]*?)\s+(\d+\.\s+)/g, '$1\n\n$2');

  return s.replace(/\n{4,}/g, '\n\n\n');
}

/** True when inline chat text likely needs full GFM rendering (headings, lists, HR). */
export function looksLikeBlockMarkdown(text: string): boolean {
  return /(^|\n|\s)(#{1,6}\s|[-*]\s|\d+\.\s|\(\d+\)\s|---\s*$)/m.test(text);
}

export function normalizeAgentMessageMarkdown(raw: string): string {
  let s = raw.replace(/\r\n/g, '\n');
  s = normalizeMarkdownFences(s);
  s = normalizeInlineFenceOpeners(s);
  s = normalizeProseMarkdownBlocks(s);
  return s;
}
