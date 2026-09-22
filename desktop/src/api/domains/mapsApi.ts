/** MapsApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class MapsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async publishDeviceLocation(body: {
    lat: number;
    lon: number;
    accuracy_m?: number;
    display_name?: string;
    captured_at?: string;
    session_id?: string;
    shared?: boolean;
    source?: string;
  }): Promise<{ ok: boolean; location?: Record<string, unknown> }> {
    const response = await this.hubFetch('/api/maps/device-location', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`Failed to publish device location: ${response.statusText}`);
    }
    return response.json();
  }

  async clearDeviceLocation(): Promise<void> {
    const response = await this.hubFetch('/api/maps/device-location', { method: 'DELETE' });
    if (!response.ok) {
      throw new Error(`Failed to clear device location: ${response.statusText}`);
    }
  }

  async fetchPendingLocationRequests(): Promise<
    Array<{
      id: string;
      agent_id?: string;
      agent_name?: string;
      channel?: string;
      created_at: string;
      status: string;
    }>
  > {
    const response = await this.hubFetch('/api/maps/location-requests/pending');
    if (!response.ok) {
      throw new Error(`Failed to list location requests: ${response.statusText}`);
    }
    return response.json();
  }

  async fulfillLocationRequest(
    requestId: string,
    body: {
      lat: number;
      lon: number;
      accuracy_m?: number;
      display_name?: string;
      captured_at?: string;
    },
  ): Promise<void> {
    const response = await this.hubFetch(`/api/maps/location-requests/${requestId}/fulfill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`Failed to fulfill location request: ${response.statusText}`);
    }
  }

  async rejectLocationRequest(requestId: string, reason?: string): Promise<void> {
    const response = await this.hubFetch(`/api/maps/location-requests/${requestId}/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: reason ?? 'User declined to share location' }),
    });
    if (!response.ok) {
      throw new Error(`Failed to reject location request: ${response.statusText}`);
    }
  }

  async reverseGeocode(lat: number, lon: number): Promise<{ display_name?: string }> {
    const response = await this.hubFetch('/api/maps/reverse', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ lat, lon }),
    });
    if (!response.ok) {
      return {};
    }
    return response.json();
  }

}
