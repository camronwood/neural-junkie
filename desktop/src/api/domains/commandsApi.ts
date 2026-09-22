/** CommandsApi — slash command definitions with in-memory cache. */
import type { CommandDefinition } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

export class CommandsApi {
  private commandsCache: CommandDefinition[] | null = null;

  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchCommands(forceRefresh: boolean = false): Promise<CommandDefinition[]> {
    if (!forceRefresh && this.commandsCache) {
      return this.commandsCache;
    }

    const response = await this.hubFetch(`/api/commands`);

    if (!response.ok) {
      throw new Error(`Failed to fetch commands: ${response.statusText}`);
    }

    this.commandsCache = await response.json();
    return this.commandsCache!;
  }

  clearCommandsCache(): void {
    this.commandsCache = null;
  }
}
