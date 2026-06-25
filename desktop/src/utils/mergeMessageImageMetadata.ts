import type { Message } from '../types/protocol';

type GeneratedImageMeta = Record<string, unknown>;

function generatedImage(meta: Record<string, unknown> | undefined): GeneratedImageMeta | undefined {
  const g = meta?.generated_image;
  return g && typeof g === 'object' ? (g as GeneratedImageMeta) : undefined;
}

/**
 * Keep inline image bytes when a history refetch redacts metadata but we already
 * received the full payload over WebSocket.
 */
export function mergeMessageImageMetadata(
  existing: Record<string, unknown> | undefined,
  incoming: Record<string, unknown> | undefined,
): Record<string, unknown> | undefined {
  if (!incoming) return existing;
  if (!existing) return incoming;

  const inG = generatedImage(incoming);
  if (!inG) {
    return { ...existing, ...incoming };
  }

  const inRedacted = inG.data_redacted === true;
  const inHasData = typeof inG.data === 'string' && inG.data.length > 0;
  const inHasPath = typeof inG.path === 'string' && inG.path.trim().length > 0;
  if (!inRedacted || inHasData || inHasPath) {
    return { ...existing, ...incoming };
  }

  const exG = generatedImage(existing);
  const exData = typeof exG?.data === 'string' ? exG.data : '';
  if (!exData) {
    return { ...existing, ...incoming };
  }

  const mergedG: GeneratedImageMeta = { ...inG, data: exData };
  delete mergedG.data_redacted;
  return {
    ...existing,
    ...incoming,
    generated_image: mergedG,
  };
}

export function mergeMessagePreservingImages(existing: Message, incoming: Message): Message {
  return {
    ...incoming,
    metadata: mergeMessageImageMetadata(
      existing.metadata as Record<string, unknown> | undefined,
      incoming.metadata as Record<string, unknown> | undefined,
    ),
  };
}
