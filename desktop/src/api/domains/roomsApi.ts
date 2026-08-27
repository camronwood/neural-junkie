import {
  hubAuthHeaders,
  normalizeHubBaseURL,
  setHubSessionToken,
} from '../../config/hubUrl';
import type { HubFetchFn } from './packsApi';

/** Session, room lifecycle, and API-key HTTP surface. */
export class RoomsApi {
  constructor(
    private readonly hubFetch: HubFetchFn,
    private readonly baseURL: string,
  ) {}

  /** Create or refresh a hub user session (channel ACL). */
  async createSession(username: string): Promise<{ token: string; username: string; role?: string }> {
    const response = await this.hubFetch('/api/auth/session', {
      method: 'POST',
      body: JSON.stringify({ username }),
    });
    if (!response.ok) {
      throw new Error(`Failed to create session: ${response.statusText}`);
    }
    const data = (await response.json()) as { token: string; username: string; role?: string };
    if (data.token) {
      setHubSessionToken(data.token);
    }
    return data;
  }

  async createRoom(params?: {
    name?: string;
    ttl_hours?: number;
    max_members?: number;
  }): Promise<{ room: any; channel: any }> {
    const response = await this.hubFetch('/api/room/create', {
      method: 'POST',
      body: JSON.stringify({
        name: params?.name ?? '',
        ttl_hours: params?.ttl_hours ?? 0,
        max_members: params?.max_members ?? 0,
      }),
    });
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to create room: ${response.statusText}`);
    }
    return response.json();
  }

  async leaveRoom(roomId: string): Promise<void> {
    const response = await this.hubFetch('/api/room/leave', {
      method: 'POST',
      body: JSON.stringify({ room_id: roomId }),
    });
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to leave room: ${response.statusText}`);
    }
  }

  async endRoom(roomId: string): Promise<void> {
    const response = await this.hubFetch('/api/room/end', {
      method: 'POST',
      body: JSON.stringify({ room_id: roomId }),
    });
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to end room: ${response.statusText}`);
    }
  }

  async getRoom(roomId: string): Promise<any> {
    const response = await this.hubFetch(`/api/room/${encodeURIComponent(roomId)}`);
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to fetch room: ${response.statusText}`);
    }
    return response.json();
  }

  async getRoomPresence(roomId: string): Promise<{ room_id: string; members: any[] }> {
    const response = await this.hubFetch(`/api/room/${encodeURIComponent(roomId)}/presence`);
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to fetch room presence: ${response.statusText}`);
    }
    return response.json();
  }

  async joinRoom(
    hostHubUrl: string,
    joinCode: string,
    username: string
  ): Promise<{ room: any; session: { token: string; username: string }; hub_url: string; hub_token: string; room_channel: string }> {
    const base = normalizeHubBaseURL(hostHubUrl);
    const response = await fetch(`${base}/api/room/join`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ join_code: joinCode, username }),
    });
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to join room: ${response.statusText}`);
    }
    return response.json();
  }

  /** Mint an admin session using the hub bootstrap secret (API keys, ACL admin). */
  async createAdminSession(
    username: string,
    bootstrapToken: string
  ): Promise<{ token: string; username: string; role: string }> {
    const boot = bootstrapToken.trim();
    if (!boot) {
      throw new Error('Bootstrap token is required for admin session');
    }
    const response = await fetch(`${this.baseURL}/api/auth/session`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...hubAuthHeaders(),
        'X-NJ-Bootstrap': boot,
      },
      body: JSON.stringify({ username, role: 'admin' }),
    });
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(
        detail ? `Admin unlock failed: ${detail}` : `Admin unlock failed: ${response.statusText}`
      );
    }
    const data = (await response.json()) as { token: string; username: string; role: string };
    if (data.role !== 'admin') {
      throw new Error('Hub did not grant admin role — check bootstrap token');
    }
    if (data.token) {
      setHubSessionToken(data.token);
    }
    return data;
  }

  async listAPIKeys(): Promise<Array<Record<string, unknown>>> {
    const response = await this.hubFetch('/api/auth/api-keys');
    if (!response.ok) {
      throw new Error(`Failed to list API keys: ${response.statusText}`);
    }
    return response.json();
  }

  async createAPIKey(name: string, role: string): Promise<{ api_key: string; record: Record<string, unknown> }> {
    const response = await this.hubFetch('/api/auth/api-keys', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, role }),
    });
    if (!response.ok) {
      throw new Error(`Failed to create API key: ${response.statusText}`);
    }
    return response.json();
  }

  async revokeAPIKey(id: string): Promise<void> {
    const response = await this.hubFetch(`/api/auth/api-keys/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error(`Failed to revoke API key: ${response.statusText}`);
    }
  }
}
