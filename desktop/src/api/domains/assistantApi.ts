import type {
  AssistantStateResponse,
  DesktopListenChunkResponse,
  DesktopListenStartResponse,
  DesktopListenStatus,
  DesktopListenStopResult,
  GoogleMeetNotesAppConfig,
  GoogleMeetNotesStatus,
} from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Assistant state, Google Meet notes, and Desktop Listen HTTP surface. */
export class AssistantApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchAssistantState(channel?: string): Promise<AssistantStateResponse> {
    const params = new URLSearchParams();
    if (channel) {
      params.set('channel', channel);
    }
    const query = params.toString();
    const response = await this.hubFetch(`/api/assistant/state${query ? `?${query}` : ''}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch assistant state: ${response.statusText}`);
    }
    return response.json();
  }

  async markAssistantTaskDone(taskID: string): Promise<void> {
    const response = await this.hubFetch(`/api/assistant/task-done`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ task_id: taskID }),
    });
    if (!response.ok) {
      throw new Error(`Failed to mark task done: ${response.statusText}`);
    }
  }

  async dismissAssistantReminder(reminderID: string): Promise<void> {
    const response = await this.hubFetch(`/api/assistant/reminder-dismiss`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reminder_id: reminderID }),
    });
    if (!response.ok) {
      throw new Error(`Failed to dismiss reminder: ${response.statusText}`);
    }
  }

  async getGoogleMeetNotesAppConfig(): Promise<GoogleMeetNotesAppConfig> {
    const response = await this.hubFetch(`/api/assistant/google/config`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Google OAuth config: ${response.statusText}`);
    }
    return response.json();
  }

  async saveGoogleMeetNotesAppConfig(
    clientId: string,
    clientSecret: string,
    redirectUrl?: string
  ): Promise<GoogleMeetNotesAppConfig> {
    const response = await this.hubFetch(`/api/assistant/google/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        client_id: clientId,
        client_secret: clientSecret,
        redirect_url: redirectUrl ?? '',
      }),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to save Google OAuth config: ${response.statusText}`);
    }
    return data;
  }

  async getGoogleMeetNotesStatus(): Promise<GoogleMeetNotesStatus> {
    const response = await this.hubFetch(`/api/assistant/google/status`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Google meet notes status: ${response.statusText}`);
    }
    return response.json();
  }

  async getGoogleMeetNotesAuthURL(): Promise<string> {
    const response = await this.hubFetch(`/api/assistant/google/auth?json=1`);
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to get auth URL: ${response.statusText}`);
    }
    return data.url;
  }

  async disconnectGoogleMeetNotes(): Promise<void> {
    const response = await this.hubFetch(`/api/assistant/google/disconnect`, {
      method: 'POST',
    });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || `Disconnect failed: ${response.statusText}`);
    }
  }

  async syncGoogleMeetNotes(): Promise<number> {
    const response = await this.hubFetch(`/api/assistant/google/sync`, {
      method: 'POST',
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Sync failed: ${response.statusText}`);
    }
    return data.ingested ?? 0;
  }

  async getListenStatus(): Promise<DesktopListenStatus> {
    const response = await this.hubFetch(`/api/assistant/listen/status`);
    if (!response.ok) {
      throw new Error(`Failed to fetch listen status: ${response.statusText}`);
    }
    return response.json();
  }

  async startListen(title?: string): Promise<DesktopListenStartResponse> {
    const response = await this.hubFetch(`/api/assistant/listen/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: title ?? '' }),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Listen start failed: ${response.statusText}`);
    }
    return data;
  }

  async sendListenChunk(sessionId: string, wav: ArrayBuffer): Promise<DesktopListenChunkResponse> {
    const response = await this.hubFetch(
      `/api/assistant/listen/chunk?session_id=${encodeURIComponent(sessionId)}`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/octet-stream',
          'X-NJ-Listen-Session': sessionId,
        },
        body: wav,
      }
    );
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Listen chunk failed: ${response.statusText}`);
    }
    return data;
  }

  async stopListen(sessionId: string): Promise<DesktopListenStopResult> {
    const response = await this.hubFetch(`/api/assistant/listen/stop`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session_id: sessionId }),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Listen stop failed: ${response.statusText}`);
    }
    return data;
  }

  async cancelListen(sessionId: string): Promise<void> {
    const response = await this.hubFetch(`/api/assistant/listen/cancel`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session_id: sessionId }),
    });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || `Listen cancel failed: ${response.statusText}`);
    }
  }
}
