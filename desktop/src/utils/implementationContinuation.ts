/** Structural channel-history helpers for implementation threads (no NL phrase banks). */

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

/** @deprecated NL phrase banks removed — always false. Hub semantic routing owns intent. */
export function hasContentDeliverySignals(_message: string): boolean {
  return false;
}

/** @deprecated Prefer explicit composer export mode. */
export function hasFileExportSignals(_message: string): boolean {
  return false;
}

/** @deprecated Prefer stamped prior_reference + export mode. */
export function hasPriorReferenceExportSignals(_message: string): boolean {
  return false;
}

/** @deprecated */
export function hasCombinedContentDeliveryExport(_message: string): boolean {
  return false;
}

/** @deprecated */
export function hasBareWorkspaceDirectiveOnly(_message: string): boolean {
  return false;
}

/** @deprecated Continuations are stamped ActionContinue. */
export function hasImplementationContinuationSignals(_message: string): boolean {
  return false;
}

/** @deprecated */
export function hasImplementationStatusCheckSignals(_message: string): boolean {
  return false;
}

/** @deprecated */
export function hasErrorLogFollowUpSignals(_message: string): boolean {
  return false;
}

/** @deprecated Implementation intent is hub-stamped. */
export function hasImplementationRequestSignals(_message: string): boolean {
  return false;
}

export const WORKSPACE_DIRECTIVE_RE = /\b(use|read|from)\s+(the\s+)?(open\s+)?workspace\b/i;
