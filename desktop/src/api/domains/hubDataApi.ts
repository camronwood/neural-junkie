/** HubDataApi — hub-local data access HTTP surface. */
import type { HubFetchFn } from './packsApi';

export class HubDataApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async readHubDataAccess(
    targets: Array<{ kind: 'file' | 'directory'; relative_path: string }>
  ): Promise<{ root: string; entries: unknown[] }> {
    const response = await this.hubFetch(`/api/hub-data/read`, {
      method: 'POST',
      body: JSON.stringify({ targets }),
    });
    if (!response.ok) {
      const t = await response.text();
      if (response.status === 404) {
        throw new Error(
          'Hub does not expose /api/hub-data/read (404). Restart the hub (`make server`) or rebuild the packaged sidecar (`make build-sidecar`).'
        );
      }
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }
}
