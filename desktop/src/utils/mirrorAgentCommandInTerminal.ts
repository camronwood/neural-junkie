import type { Message } from '../types/protocol';
import type { CommandOutput } from '../types/protocol';
import { terminalAPI } from '../api/terminalAPI';
import { useTerminalStore } from '../stores/terminalStore';
import { resolveTerminalCwd } from './terminalCwd';

export const METADATA_AGENT_RUN_COMMAND = 'agent_run_command';
export const METADATA_MIRROR_TERMINAL = 'mirror_terminal';

function mirrorLines(command: string, output: CommandOutput): string {
  const lines = [
    `# @agent ran:`,
    `$ ${command}`,
    `# exit ${output.exit_code}`,
    output.stdout?.trimEnd() ?? '',
    output.stderr?.trimEnd() ?? '',
  ].filter((l) => l.length > 0);
  return lines.join('\n') + '\n\n';
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

  const store = useTerminalStore.getState();
  store.setPanelOpen(true);
  const cwd = resolveTerminalCwd({});
  store.alignActiveTabCwd(cwd);
  void terminalAPI.writePtySession(store.activeTabId, mirrorLines(command, output));
}
