import type { Message, ThreadMetadata } from '../../types/protocol';
import type { SendMessageResponse } from '../chatAPI';
import type { ContextRequestPayload } from '../../utils/contextRequestAttach';
import type { HubFetchFn } from './packsApi';

/** Messages, turns, threads, and channel interject HTTP surface. */
export class MessagesApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchMessages(channel: string, limit: number = 50, beforeId?: string): Promise<Message[]> {
    const params = new URLSearchParams({ channel, limit: String(limit) });
    if (beforeId?.trim()) {
      params.set('before', beforeId.trim());
    }
    const response = await this.hubFetch(`/api/messages?${params}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch messages: ${response.statusText}`);
    }
    return response.json();
  }

  async sendMessage(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    credentials?: Record<string, unknown>
  ): Promise<SendMessageResponse> {
    const body: Record<string, unknown> = { channel, content, type, from };
    if (credentials) {
      body.metadata = { ...credentials };
      const replyTo = credentials.reply_to;
      if (typeof replyTo === 'string' && replyTo.trim()) {
        body.reply_to = replyTo.trim();
      }
    }
    const response = await this.hubFetch('/api/send', {
      method: 'POST',
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`Failed to send message: ${response.statusText}`);
    }
    const text = await response.text();
    if (!text.trim()) return { status: 'ok' };
    try {
      return JSON.parse(text) as SendMessageResponse;
    } catch {
      return { status: 'ok' };
    }
  }

  async prepareTurn(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    metadata?: Record<string, unknown>
  ): Promise<{
    prepare_token: string;
    context_request: ContextRequestPayload;
    decision?: Record<string, unknown>;
  }> {
    const body: Record<string, unknown> = { channel, content, type, from };
    if (metadata) {
      body.metadata = { ...metadata };
      const replyTo = metadata.reply_to;
      if (typeof replyTo === 'string' && replyTo.trim()) {
        body.reply_to = replyTo.trim();
      }
    }
    const response = await this.hubFetch('/api/turn/prepare', {
      method: 'POST',
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to prepare turn: ${response.statusText}`);
    }
    return response.json();
  }

  async dispatchTurn(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    metadata?: Record<string, unknown>
  ): Promise<SendMessageResponse> {
    const body: Record<string, unknown> = { channel, content, type, from };
    if (metadata) {
      body.metadata = { ...metadata };
      const replyTo = metadata.reply_to;
      if (typeof replyTo === 'string' && replyTo.trim()) {
        body.reply_to = replyTo.trim();
      }
    }
    const response = await this.hubFetch('/api/turn/dispatch', {
      method: 'POST',
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`Failed to dispatch turn: ${response.statusText}`);
    }
    const text = await response.text();
    if (!text.trim()) return { status: 'ok' };
    try {
      return JSON.parse(text) as SendMessageResponse;
    } catch {
      return { status: 'ok' };
    }
  }

  async sendThreadReply(
    threadId: string,
    channel: string,
    content: string,
    from: { name: string; type: string },
    metadata?: Record<string, unknown>
  ): Promise<void> {
    const body: Record<string, unknown> = { channel, content, from };
    if (metadata && Object.keys(metadata).length > 0) {
      body.metadata = metadata;
    }
    const response = await this.hubFetch(`/api/threads/${encodeURIComponent(threadId)}/reply`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`Failed to send thread reply: ${response.statusText}`);
    }
  }

  async fetchThreadMetadata(threadId: string): Promise<ThreadMetadata> {
    const response = await this.hubFetch(`/api/threads/${encodeURIComponent(threadId)}/metadata`);
    if (!response.ok) {
      throw new Error(`Failed to fetch thread metadata: ${response.statusText}`);
    }
    return response.json();
  }

  async channelInterject(
    channel: string,
    heldBy?: string
  ): Promise<{ channel: string; held: boolean }> {
    const response = await this.hubFetch(`/api/channels/${encodeURIComponent(channel)}/interject`, {
      method: 'POST',
      body: JSON.stringify({ held_by: heldBy ?? '' }),
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async answerUserQuestion(questionId: string, answer: string): Promise<void> {
    const response = await this.hubFetch(`/api/user-questions/answer/${questionId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ answer }),
    });
    if (!response.ok) {
      throw new Error(`Failed to answer question: ${response.statusText}`);
    }
  }
}
