import type {
  AssistantStateResponse,
  GoogleMeetNotesAppConfig,
  GoogleMeetNotesStatus,
} from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Assistant state and Google Meet notes HTTP surface. */
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
}
