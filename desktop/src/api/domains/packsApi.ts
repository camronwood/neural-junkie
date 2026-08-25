import type {
  InstallPackLoRAsResponse,
  PackCatalogEntry,
  PackUpdatesResponse,
  PacksAPIResponse,
} from '../chatAPI';

export type HubFetchFn = (path: string, init?: RequestInit) => Promise<Response>;

function parsePacksMutationResponse(data: unknown): PacksAPIResponse {
  if (data && typeof data === 'object') {
    return data as PacksAPIResponse;
  }
  return { packs: [] };
}

/** Pack install/catalog/layout HTTP surface (composed by ChatAPI). */
export class PacksApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchPacks(): Promise<PacksAPIResponse> {
    const response = await this.hubFetch('/api/packs');
    if (!response.ok) {
      throw new Error(`Failed to fetch packs: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchPackCatalog(): Promise<PackCatalogEntry[]> {
    const response = await this.hubFetch('/api/packs/catalog');
    if (!response.ok) {
      throw new Error(`Failed to fetch pack catalog: ${response.statusText}`);
    }
    const data = await response.json();
    return (data.packs as PackCatalogEntry[]) ?? [];
  }

  async refreshPackCatalog(): Promise<PackCatalogEntry[]> {
    const response = await this.hubFetch('/api/packs/catalog/refresh', { method: 'POST' });
    if (!response.ok) {
      throw new Error(`Failed to refresh pack catalog: ${response.statusText}`);
    }
    const data = await response.json();
    return (data.packs as PackCatalogEntry[]) ?? [];
  }

  async fetchPackUpdates(): Promise<PackUpdatesResponse> {
    const response = await this.hubFetch('/api/packs/updates');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async setLayoutOwner(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch('/api/packs/layout-owner', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ layout_owner: packId }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return parsePacksMutationResponse(await response.json());
  }

  async installPack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}/install`, {
      method: 'POST',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return parsePacksMutationResponse(await response.json());
  }

  async upgradePack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}/upgrade`, {
      method: 'POST',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return parsePacksMutationResponse(await response.json());
  }

  async installPackLoRAs(packId: string): Promise<InstallPackLoRAsResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}/install-loras`, {
      method: 'POST',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }
}
