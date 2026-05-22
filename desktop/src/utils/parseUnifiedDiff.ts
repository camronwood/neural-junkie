export interface DiffHunk {
  id: string;
  oldStart: number;
  newStart: number;
  removedLines: string[];
  addedLines: string[];
}

/** Parses unified diff text into hunks (simplified). */
export function parseUnifiedDiff(diff: string): DiffHunk[] {
  const hunks: DiffHunk[] = [];
  const lines = diff.split('\n');
  let i = 0;
  let hunkIndex = 0;
  while (i < lines.length) {
    const line = lines[i];
    const hdr = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(line);
    if (hdr) {
      const hunk: DiffHunk = {
        id: `h${hunkIndex++}`,
        oldStart: parseInt(hdr[1], 10),
        newStart: parseInt(hdr[2], 10),
        removedLines: [],
        addedLines: [],
      };
      i++;
      while (i < lines.length && !lines[i].startsWith('@@')) {
        const l = lines[i];
        if (l.startsWith('-') && !l.startsWith('---')) {
          hunk.removedLines.push(l.slice(1));
        } else if (l.startsWith('+') && !l.startsWith('+++')) {
          hunk.addedLines.push(l.slice(1));
        }
        i++;
      }
      hunks.push(hunk);
      continue;
    }
    i++;
  }
  return hunks;
}

/** Apply hunk added lines at newStart (1-based) into content. */
export function applyHunkToContent(content: string, hunk: DiffHunk): string {
  const fileLines = content.split('\n');
  const insertAt = Math.max(0, hunk.newStart - 1);
  const removeCount = hunk.removedLines.length;
  fileLines.splice(insertAt, removeCount, ...hunk.addedLines);
  return fileLines.join('\n');
}
