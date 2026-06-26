import type { Message } from '../types/protocol';
import type { CommandOutput } from '../types/protocol';
import { terminalAPI } from '../api/terminalAPI';
import { createNewTab, useTerminalStore } from '../stores/terminalStore';
import { resolveTerminalCwd } from './terminalCwd';

export const METADATA_AGENT_RUN_COMMAND = 'agent_run_command';
export const METADATA_MIRROR_TERMINAL = 'mirror_terminal';

function mirrorLines(
  command: string,
  output: CommandOutput,
  channel?: string,
  agentName?: string
): string {
  const who = agentName?.trim() || 'agent';
  const header = channel ? `# @${who} ran (${channel}):` : `# @${who} ran:`;
  const lines = [
    header,
    `$ ${command}`,
    `# exit ${output.exit_code}`,
    output.stdout?.trimEnd() ?? '',
    output.stderr?.trimEnd() ?? '',
  ].filter((l) => l.length > 0);
  return lines.join('\n') + '\n\n';
}

function resolveMirrorTabId(agentName: string): string {
  const store = useTerminalStore.getState();
  const existing = store.tabs.find((t) => t.type === 'agent' && t.agentName === agentName);
  if (existing) {
    store.setActiveTab(existing.id);
    return existing.id;
  }
  const cwd = resolveTerminalCwd({});
  const tab = createNewTab('agent', agentName, cwd);
  store.addTab(tab);
  return tab.id;
}

/** Mirror hub-posted agent run_command output into the visible terminal panel. */
export function mirrorAgentCommandInTerminal(message: Message): void {
  if (message.type !== 'command_output') return;
  const meta = message.metadata;
  if (!meta?.[METADATA_MIRROR_TERMINAL] && !meta?.[METADATA_AGENT_RUN_COMMAND]) return;
  const raw = meta.command_output;
  if (typeof raw !== 'string') return;
  let output: CommandOutput;
  try {
    output = JSON.parse(raw) as CommandOutput;
  } catch {
    return;
  }
  const command = output.command?.trim();
  if (!command) return;

  const agentName = message.from?.name?.trim() || 'Agent';
  const channel = message.channel?.trim();

  const store = useTerminalStore.getState();
  store.setPanelOpen(true);
  const tabId = resolveMirrorTabId(agentName);
  const cwd = resolveTerminalCwd({});
  store.alignActiveTabCwd(cwd);
  void terminalAPI.writePtySession(tabId, mirrorLines(command, output, channel, agentName));
}
