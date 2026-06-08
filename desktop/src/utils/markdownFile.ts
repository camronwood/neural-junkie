/** True for workspace markdown files that support inline edit/preview in the editor. */
export function isMarkdownPath(path: string): boolean {
  if (!path) return false;
  const lower = path.toLowerCase();
  return lower.endsWith('.md') || lower.endsWith('.markdown');
}
