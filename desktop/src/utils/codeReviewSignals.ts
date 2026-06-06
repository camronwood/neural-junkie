/** Read-only project/codebase review — not implementation/file-edit tasks. */
const CODE_REVIEW_RE =
  /\b(code review|review (this|the|my) (project|codebase|repository|repo|app|workspace)|review (the )?code in (the )?(workspace|project|repo|codebase)|review (the )?workspace|audit (this|the) (project|codebase|code))\b/i;

export function hasCodeReviewSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (CODE_REVIEW_RE.test(text)) return true;
  const lower = text.toLowerCase();
  if (/\breview\b/.test(lower) && /\bworkspace\b/.test(lower)) return true;
  if (/\breview\b/.test(lower) && /\b(the )?code\b/.test(lower) && !/\b(for issues|for bugs|and fix)\b/.test(lower)) {
    if (/\b(workspace|project|codebase|repository|repo|app)\b/.test(lower)) return true;
  }
  return false;
}
