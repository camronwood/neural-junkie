/** User affirms a prior implementation/theme request without repeating code verbs. */
const IMPLEMENTATION_CONTINUATION_RE =
  /\b(go ahead|do it( now)?|yes please|please do|proceed|make (the |those )?changes|apply (that|it|your plan)|do that now|ok please|sure,?\s*please|let's do it|please implement|sounds good[,!]?\s*(go|do)|that works[,!]?\s*(go|do)|you can (start|begin))\b/i;

export function hasImplementationContinuationSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text || text.length > 120) return false;
  return IMPLEMENTATION_CONTINUATION_RE.test(text);
}
