/** User affirms a prior implementation/theme request without repeating code verbs. */
const IMPLEMENTATION_CONTINUATION_RE =
  /\b(approved|approve(d| it)?|keep going|please continue|continue(?: with (it|this|that|the work))?|looks good|that sounds good|sounds good|go[- ]?ahead|goadhead|do it( now)?|yes please|please do|proceed|make (the |those )?(changes|them)|apply (that|it|your plan)|do that now|ok please|sure,?\s*please|let's do it|please implement|sounds good[,!]?\s*(go|do)|that works[,!]?\s*(go|do)?|you can (start|begin|proceed)|yeah go ahead|yes[,!]?\s*(keep going|that sounds good|use that|please))\b/i;

/** Bare acknowledgements — not approval to ship file changes. */
const WEAK_AFFIRMATION_ONLY_RE =
  /^(?:@\w+\s+)?(?:ok|okay|looks good|that works|sounds good|nice|great|cool|perfect)[!.?\s]*$/i;

/** Coding/build asks where the user expects file changes, not generic chat. */
const IMPLEMENTATION_REQUEST_RE =
  /\b(settings modal|font size|pick up where|finish (?:that |the )?work|theme support|dark[/ ]light|light[/ ]dark|dark mode|light mode|settings page|wire up|hook up|not working|does(?:n't| not) work|not booting|won't boot|broken|debug this|blank screen|white screen|can you fix)\b/i;

/** Short status check after a fix attempt in the same thread. */
const IMPLEMENTATION_STATUS_CHECK_RE =
  /^(?:@\w+\s+)?(?:is it fixed|did (?:that|it) fix|does it work(?: now)?|is it working(?: now)?|still broken|still not (?:booting|working)|working now)\??[!.?\s]*$/i;

/** Error-log / stack-trace paste while debugging in an active implementation thread. */
const ERROR_LOG_FOLLOW_UP_RE =
  /\b(still getting|also got this|got this error|seeing this|getting this)\b/i;

const ERROR_LOG_MARKERS_RE =
  /\b(?:VITE v?\d|exit_code=\d+|error TS\d+|TS\d{4,5}:|npm ERR!|UnhandledPromiseRejection|stack trace|node_modules\/|at file:\/\/|process\.processTicksAndRejections|Waiting for your frontend dev server)\b/i;

/** User directs the agent to use the shared workspace instead of asking for pasted files. */
export const WORKSPACE_DIRECTIVE_RE =
  /\b(use|read|from)\s+(the\s+)?(open\s+)?workspace\b/i;

/** Writing/marketing tasks — not code implementation sessions. */
const CONTENT_DELIVERY_RE =
  /\b(linkedin|blog post|blog article|article about|write (?:me )?(?:a |an )?article|marketing copy|press release|social media post|whitepaper|writeup|newsletter)\b/i;

const BARE_WORKSPACE_WRAPPER_RE =
  /\b(can you|could you|please|for this|for that|to do this|now)\b/gi;

export function hasContentDeliverySignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  return CONTENT_DELIVERY_RE.test(text);
}

/** Save/store/create/fill markdown file — route as code + implementation, not chat re-ask loops. */
/** @deprecated Prefer explicit composer Mode: Export; kept for Auto migration fallback. */
export function hasFileExportSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  const lower = text.toLowerCase();
  const phrases = [
    'store that',
    'store it',
    'save it',
    'save that',
    'save it as',
    'save it in',
    'fill the file',
    'create that file',
    'please create that file',
    'markdown file',
  ];
  for (const p of phrases) {
    if (lower.includes(p)) return true;
  }
  if (/\bfill\b/i.test(text) && /\bwith\b/i.test(text)) return true;
  const hasFileTarget =
    /\.md\b/i.test(text) || lower.includes('markdown file') || lower.includes('the file') || lower.includes('to a file');
  const hasExportVerb = /\b(store|save|create|fill|write)\b/i.test(text);
  return hasFileTarget && hasExportVerb;
}

/** Prior-reference + save/export intent — auto-routes as Export mode in the background. */
export function hasPriorReferenceExportSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  const prior =
    /\b(few messages back|what you wrote|you created|that art(?:icle|ical)|artical content|earlier you|from before|you wrote|message(s)? back|article you wrote)\b/i;
  const exportIntent =
    /\b(save|store|write|export|markdown|\.md\b|to a file|the file)\b/i;
  return prior.test(text) && exportIntent.test(text);
}

/** Write + save in one message — chat generates first; do not auto-route as Export metadata. */
export function hasCombinedContentDeliveryExport(message: string): boolean {
  return hasContentDeliverySignals(message) && hasFileExportSignals(message);
}

/** Short "use the workspace" messages without a concrete code deliverable. */
export function hasBareWorkspaceDirectiveOnly(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text || !WORKSPACE_DIRECTIVE_RE.test(text)) return false;
  let stripped = text.replace(WORKSPACE_DIRECTIVE_RE, '');
  stripped = stripped.replace(BARE_WORKSPACE_WRAPPER_RE, '');
  stripped = stripped.replace(/^[?,.\s]+|[?,.\s]+$/g, '').trim();
  if (!stripped) return true;
  const lower = stripped.toLowerCase();
  const codeVerbs = [
    'implement',
    'fix',
    'debug',
    'build',
    'theme',
    'settings',
    'modal',
    'patch',
    'refactor',
    'broken',
    'not working',
  ];
  for (const v of codeVerbs) {
    if (lower.includes(v)) return false;
  }
  return stripped.length < 40;
}

export function hasImplementationContinuationSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text || text.length > 120) return false;
  if (WEAK_AFFIRMATION_ONLY_RE.test(text)) return false;
  return IMPLEMENTATION_CONTINUATION_RE.test(text);
}

export function hasImplementationStatusCheckSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  return IMPLEMENTATION_STATUS_CHECK_RE.test(text);
}

/** User pasted build/runtime output or a stack trace while iterating on a fix. */
export function hasErrorLogFollowUpSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (ERROR_LOG_FOLLOW_UP_RE.test(text) && ERROR_LOG_MARKERS_RE.test(text)) return true;
  if (ERROR_LOG_MARKERS_RE.test(text) && text.length >= 40) return true;
  return false;
}

export type ChannelMessageRef = {
  type?: string;
  metadata?: Record<string, unknown>;
};

/** Recent channel history shows an active or recently failed implementation session. */
export function channelHasImplementationThread(
  messages: ChannelMessageRef[] | undefined,
  lookback = 24
): boolean {
  if (!messages?.length) return false;
  const recent = messages.slice(-lookback);
  for (const m of recent) {
    const meta = m.metadata ?? {};
    if (meta.implementation_session === true) return true;
    if (meta.implementation_session_complete === true) return true;
    if (meta.file_change_approved === true) return true;
    if (meta.can_run_impl_session === true) return true;
    if (m.type === 'file_change') return true;
    const outcome = meta.implementation_session_outcome;
    if (
      outcome &&
      typeof outcome === 'object' &&
      (outcome as { verify_failed?: boolean }).verify_failed === true
    ) {
      return true;
    }
  }
  return false;
}

export function hasImplementationRequestSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (hasImplementationStatusCheckSignals(text)) return true;
  if (WORKSPACE_DIRECTIVE_RE.test(text)) return true;
  if (IMPLEMENTATION_REQUEST_RE.test(text)) return true;
  if (/\badd(?:ing)?\b/i.test(text) && /\b(theme|themes|modal|settings)\b/i.test(text)) {
    return true;
  }
  return false;
}
