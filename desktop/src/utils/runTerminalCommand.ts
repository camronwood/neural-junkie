import type { ChatAPI } from '../api/chatAPI';
import { terminalAPI } from '../api/terminalAPI';
import type { CommandSuggestion } from '../stores/terminalStore';
import { useTerminalStore } from '../stores/terminalStore';
import { resolveTerminalCwd } from './terminalCwd';
import { postCommandOutputToHub } from './postCommandOutput';
import type { Collaboration } from '../types/protocol';

export function shellQuote(path: string): string {
  if (/^[a-zA-Z0-9_./-]+$/.test(path)) {
    return path;
  }
  return `'${path.replace(/'/g, `'\\''`)}'`;
}

function mirrorResultInPty(tabId: string, command: string, result: Awaited<ReturnType<typeof terminalAPI.executeCommand>>): void {
  const lines = [
    `$ ${command}`,
    `# exit ${result.exit_code}`,
    result.stdout?.trimEnd() ?? '',
    result.stderr?.trimEnd() ?? '',
  ].filter((l) => l.length > 0);
  void terminalAPI.writePtySession(tabId, lines.join('\n') + '\n\n');
}

export interface RunAgentTerminalCommandOptions {
  collaboration?: Collaboration | null;
  channel: string;
  api: ChatAPI;
  /** Mirror output in the visible terminal panel. */
  mirrorInPty?: boolean;
}

/**
 * Runs an agent-suggested shell command, posts results to the hub for agents, and optionally mirrors in the PTY.
 */
export async function runAgentTerminalCommand(
  suggestion: CommandSuggestion,
  options: RunAgentTerminalCommandOptions
): Promise<void> {
  const cwd =
    suggestion.cwd?.trim() ||
    resolveTerminalCwd({ collaboration: options.collaboration ?? null });

  const store = useTerminalStore.getState();
  store.alignActiveTabCwd(cwd);

  const result = await terminalAPI.executeCommand(suggestion.command, cwd === '~' ? undefined : cwd);

  await postCommandOutputToHub(options.api, options.channel, result, {
    agentName: suggestion.agent_name,
    replyToMessageId: suggestion.message_id,
  });

  if (options.mirrorInPty !== false) {
    mirrorResultInPty(store.activeTabId, suggestion.command, result);
  }
}

/** @deprecated Use runAgentTerminalCommand */
export async function runSafeTerminalSuggestion(
  suggestion: CommandSuggestion,
  options?: { collaboration?: Collaboration | null; channel?: string; api?: ChatAPI }
): Promise<void> {
  if (!options?.channel || !options?.api) {
    throw new Error('runSafeTerminalSuggestion requires channel and api');
  }
  await runAgentTerminalCommand(suggestion, {
    collaboration: options.collaboration,
    channel: options.channel,
    api: options.api,
  });
}
