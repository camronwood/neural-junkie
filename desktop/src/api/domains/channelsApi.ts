import type { Channel } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Channel list HTTP surface (composed by ChatAPI). */
export class ChannelsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchChannels(): Promise<Channel[]> {
    const response = await this.hubFetch('/api/channels');
    if (!response.ok) {
      throw new Error(`Failed to fetch channels: ${response.statusText}`);
    }
    return response.json();
  }
}
