/** Pending ask_user cards on a channel timeline. */
export function pendingUserQuestionMessages<T extends { type?: string; metadata?: Record<string, unknown> | null }>(
  messages: T[] | undefined | null
): T[] {
  if (!messages?.length) return [];
  return messages.filter(
    (m) => m.type === 'user_question' && m.metadata?.status === 'pending' && !!m.metadata?.question_id
  );
}

export function pendingUserQuestionIds(
  messages: { type?: string; metadata?: Record<string, unknown> | null }[] | undefined | null
): string[] {
  const ids: string[] = [];
  const seen = new Set<string>();
  for (const m of pendingUserQuestionMessages(messages)) {
    const id = String(m.metadata?.question_id ?? '');
    if (!id || seen.has(id)) continue;
    seen.add(id);
    ids.push(id);
  }
  return ids;
}
