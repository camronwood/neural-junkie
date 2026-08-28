import type {
  StreamDispatchResult,
  StreamManagerStatus,
  StreamSubscription,
} from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Stream manager and subscription HTTP surface. */
export class StreamsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async getStreamStatus(): Promise<StreamManagerStatus> {
    const response = await this.hubFetch('/api/stream/status');
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async restartStreamManager(): Promise<void> {
    const response = await this.hubFetch('/api/stream/restart', { method: 'POST' });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
  }

  async listStreamSubscriptions(): Promise<StreamSubscription[]> {
    const response = await this.hubFetch('/api/stream/subscriptions');
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    const data = await response.json();
    return Array.isArray(data) ? data : [];
  }

  async saveStreamSubscription(
    sub: StreamSubscription,
    isNew: boolean
  ): Promise<StreamSubscription> {
    const response = await this.hubFetch(
      isNew ? '/api/stream/subscriptions' : `/api/stream/subscriptions/${encodeURIComponent(sub.id)}`,
      {
        method: isNew ? 'POST' : 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(sub),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async deleteStreamSubscription(id: string): Promise<void> {
    const response = await this.hubFetch(`/api/stream/subscriptions/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
  }

  async testStreamSubscription(
    id: string,
    payload: string,
    topic?: string
  ): Promise<StreamDispatchResult> {
    const response = await this.hubFetch(
      `/api/stream/subscriptions/${encodeURIComponent(id)}/test`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ payload, topic }),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }
}
