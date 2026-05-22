import type { Message, ThreadMetadata } from '../types/protocol';

function isSlackMirrorChannel(channel: string): boolean {
  return channel.startsWith('slack:');
}

/**
 * Thread id for hub API / ThreadPanel.
 * Slack mirror replies use the parent NJ message id when mapped; legacy threads may use slack_ts.
 */
export function slackThreadOpenId(message: Message, channel: string): string {
  if (!isSlackMirrorChannel(channel)) {
    return message.id;
  }
  if (message.is_thread_reply && message.thread_id) {
    return message.thread_id;
  }
  return message.id;
}

/** Resolve thread metadata for a parent message (NJ id or legacy slack_ts key). */
export function slackThreadMetadataLookup(
  threadMetadata: Map<string, ThreadMetadata>,
  message: Message,
  channel: string
): ThreadMetadata | undefined {
  const byID = threadMetadata.get(message.id);
  if (byID) {
    return byID;
  }
  if (!isSlackMirrorChannel(channel)) {
    return undefined;
  }
  const meta = message.metadata as Record<string, unknown> | undefined;
  const slackTs = meta?.slack_ts;
  if (typeof slackTs === 'string' && slackTs.trim() !== '') {
    const legacy = threadMetadata.get(slackTs);
    if (legacy) {
      return legacy;
    }
  }
  if (message.thread_id) {
    return threadMetadata.get(message.thread_id);
  }
  return undefined;
}

/** Find parent message when opening a thread by NJ id or legacy slack_ts. */
export function findThreadParentMessage(
  messages: Message[],
  openThreadId: string
): Message | null {
  if (!openThreadId) return null;
  return (
    messages.find((m) => m.id === openThreadId) ??
    messages.find((m) => {
      const ts = (m.metadata as Record<string, unknown> | undefined)?.slack_ts;
      return typeof ts === 'string' && ts === openThreadId;
    }) ??
    null
  );
}
