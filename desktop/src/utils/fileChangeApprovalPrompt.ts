import type { Collaboration, Message } from '../types/protocol';

/** True when a file_change message should surface the approval modal. */
export function shouldPromptFileChangeApproval(
  message: Message,
  activeChannel: string,
  collaborations: Record<string, Collaboration>,
): boolean {
  if (message.type !== 'file_change') {
    return false;
  }
  if (message.metadata?.file_change_auto_approved === true) {
    return false;
  }
  const channel = message.channel?.trim();
  if (!channel) {
    return false;
  }
  if (channel === activeChannel.trim()) {
    return true;
  }
  for (const collab of Object.values(collaborations)) {
    if (collab.phase === 'executing' && collab.channel?.trim() === channel) {
      return true;
    }
  }
  return false;
}

export function registeredFileChangeId(message: Message): string {
  const id = message.metadata?.registered_change_id;
  return typeof id === 'string' ? id.trim() : '';
}
