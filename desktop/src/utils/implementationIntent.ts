/**
 * Lightweight mirrors of internal/intent LooksLikeDirectImplementationAsk /
 * PrefersWorkspaceImplementOverOpenCanvas for desktop Agent-mode metadata.
 * Keep in sync when those Go helpers change.
 */

export function looksLikeExecuteImplementationAsk(text: string): boolean {
  const c = (text ?? '').trim().toLowerCase();
  if (!c || c.includes('?')) return false;
  if (
    c.includes('propose a plan') ||
    c.includes('how should we') ||
    c.includes("what's the approach") ||
    c.includes('what is the approach') ||
    c.includes('draft a plan') ||
    c.startsWith('plan ')
  ) {
    return false;
  }
  const verbs = [
    'add a ',
    'add an ',
    'create a ',
    'create an ',
    'implement ',
    'write a ',
    'write an ',
    'build a ',
    'build an ',
    'insert a ',
    'insert an ',
  ];
  // Canvas / diagram / mermaid asks are artifact delivery, not workspace implement.
  if (c.includes('canvas') || c.includes('mermaid') || c.includes('diagram')) {
    return false;
  }
  for (const v of verbs) {
    if (c.startsWith(v) || c.includes(` ${v}`)) return true;
  }
  const hasImplement =
    c.includes('implement') ||
    c.includes('make the change') ||
    c.includes('apply the fix') ||
    c.includes('ship it') ||
    c.includes('do the edit') ||
    c.includes('write the code');
  const hasPlanCue =
    c.includes('plan') ||
    c.includes('approve') ||
    c.includes('go ahead') ||
    c.includes('do it now');
  return hasImplement && hasPlanCue;
}

export function looksLikeImplementGoAhead(text: string): boolean {
  const c = (text ?? '').trim().toLowerCase();
  if (!c || c.includes('?')) return false;
  const fields = c.split(/[\s,!.]+/).filter(Boolean);
  if (fields.length === 0 || fields.length > 10) return false;
  const exact = new Set([
    'ok',
    'okay',
    'k',
    'yes',
    'yep',
    'yeah',
    'y',
    'sure',
    'do it',
    'ship it',
    'go ahead',
    'go for it',
    'lgtm',
    'sounds good',
    'that sounds good',
    'ok that sounds good',
    'okay that sounds good',
    'looks good',
    'proceed',
  ]);
  if (exact.has(c)) return true;
  if (
    (c.startsWith('ok ') || c.startsWith('okay ')) &&
    (c.includes('sound') || c.includes('good') || c.includes('go ahead'))
  ) {
    return true;
  }
  return false;
}

/** Whether Agent-mode should stamp implementation_session like the scenario harness. */
export function shouldStampAgentImplementationSession(
  content: string,
  recentHasImplementationThread: boolean
): boolean {
  if (looksLikeExecuteImplementationAsk(content)) return true;
  if (looksLikeImplementGoAhead(content) && recentHasImplementationThread) return true;
  return false;
}
