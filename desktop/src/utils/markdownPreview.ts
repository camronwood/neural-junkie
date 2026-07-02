/** Plain-text preview of markdown for narrow sidebar/status strips. */
export function markdownPreviewLine(text: string, maxLen = 120): string {
  let s = text.trim();
  if (!s) return '';
  s = s.replace(/^#{1,6}\s+/gm, '');
  s = s.replace(/^\s*[-*]\s+/gm, '');
  s = s.replace(/\*\*([^*]+)\*\*/g, '$1');
  s = s.replace(/\*([^*]+)\*/g, '$1');
  s = s.replace(/`([^`]+)`/g, '$1');
  s = s.replace(/\s+/g, ' ').trim();
  if (s.length > maxLen) {
    return `${s.slice(0, maxLen - 1)}…`;
  }
  return s;
}
