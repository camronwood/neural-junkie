import type { WebSearchConfigResponse } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Web search config and connection test HTTP surface. */
export class WebSearchApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async getWebSearchConfig(): Promise<WebSearchConfigResponse> {
    const response = await this.hubFetch(`/api/web-search/config`);
    if (!response.ok) {
      throw new Error(`Failed to fetch web search config: ${response.statusText}`);
    }
    return response.json();
  }

  async saveWebSearchConfig(body: {
    enabled?: boolean;
    provider?: string;
    api_key?: string;
    max_results?: number;
    keyless?: boolean;
  }): Promise<{ status: string }> {
    const response = await this.hubFetch(`/api/web-search/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to save web search config: ${response.statusText}`);
    }
    return data;
  }

  async testWebSearchConnection(): Promise<{
    status: string;
    results?: Array<{ title: string; url: string; description: string }>;
  }> {
    const response = await this.hubFetch(`/api/web-search/test`, { method: 'POST' });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Web search test failed: ${response.statusText}`);
    }
    return data;
  }
}
