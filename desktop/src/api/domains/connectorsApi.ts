import type { ConnectorProfile } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Connector profile HTTP surface. */
export class ConnectorsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async listConnectors(): Promise<ConnectorProfile[]> {
    const response = await this.hubFetch('/api/connectors');
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async saveConnector(
    profile: ConnectorProfile & { secret?: string },
    isNew: boolean
  ): Promise<ConnectorProfile> {
    const response = await this.hubFetch(
      isNew ? '/api/connectors' : `/api/connectors/${encodeURIComponent(profile.id)}`,
      {
        method: isNew ? 'POST' : 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(profile),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async deleteConnector(id: string): Promise<void> {
    const response = await this.hubFetch(`/api/connectors/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
  }
}
