import type { ChatAPI } from '../api/chatAPI';
import { terminalAPI, type CommandResult } from '../api/terminalAPI';
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

/** Dev servers and watchers never exit — run them in the PTY instead of blocking execute_command. */
export function isLongRunningShellCommand(command: string): boolean {
  const c = command.toLowerCase().replace(/\s+/g, ' ').trim();
  if (!c) return false;
  const patterns = [
    /\btauri\s+dev\b/,
    /\bnpm\s+run\s+tauri(\s+dev)?\b/,
    /\bnpm\s+run\s+dev\b/,
    /\bnpm\s+start\b/,
    /\byarn\s+(dev|start)\b/,
    /\bpnpm\s+(dev|start)\b/,
    /\bbun\s+(dev|start)\b/,
    /\bcargo\s+run\b/,
    /\bcargo\s+watch\b/,
    /\bvite\b/,
    /\bnext\s+dev\b/,
    /\bwebpack(-dev-server)?\b/,
    /\bnodemon\b/,
  ];
  return patterns.some((re) => re.test(c));
}

function mirrorResultInPty(tabId: string, command: string, result: CommandResult): void {
  const lines = [
    `$ ${command}`,
    `# exit ${result.exit_code}`,
    result.stdout?.trimEnd() ?? '',
    result.stderr?.trimEnd() ?? '',
  ].filter((l) => l.length > 0);
  void terminalAPI.writePtySession(tabId, lines.join('\n') + '\n\n');
}

function failureResult(command: string, err: unknown): CommandResult {
  const message = err instanceof Error ? err.message : String(err);
  return {
    id: `err-${Date.now()}`,
    command,
    exit_code: 1,
    stdout: '',
    stderr: message,
    duration_ms: 0,
    success: false,
  };
}

function startedLongRunningResult(command: string): CommandResult {
  return {
    id: `pty-${Date.now()}`,
    command,
    exit_code: -1,
    stdout:
      'Started in the interactive terminal (long-running process; no exit code yet).\n' +
      'Watch the terminal panel for Vite/Tauri readiness and the desktop window.\n' +
      'Continue once startup output looks healthy; do not claim the command failed solely because it has not exited.',
    stderr: '',
    duration_ms: 0,
    success: true,
  };
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
 * User clicking Run counts as approval — long-running commands are started in the PTY; failures still post command_output.
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
  store.setPanelOpen(true);

  const tabId = store.activeTabId;
  let result: CommandResult;

  if (isLongRunningShellCommand(suggestion.command)) {
    try {
      await terminalAPI.writePtySession(tabId, `${suggestion.command}\n`);
      result = startedLongRunningResult(suggestion.command);
    } catch (err) {
      result = failureResult(suggestion.command, err);
    }
  } else {
    try {
      result = await terminalAPI.executeCommand(suggestion.command, cwd === '~' ? undefined : cwd, {
        userApproved: true,
      });
      if (options.mirrorInPty !== false) {
        mirrorResultInPty(tabId, suggestion.command, result);
      }
    } catch (err) {
      result = failureResult(suggestion.command, err);
      if (options.mirrorInPty !== false) {
        try {
          await terminalAPI.writePtySession(tabId, `${suggestion.command}\n`);
          result = {
            ...result,
            stdout:
              (result.stdout ? result.stdout + '\n' : '') +
              'Fell back to interactive terminal after execute_command failed. Watch the terminal panel for output.',
            success: true,
            exit_code: -1,
          };
        } catch {
          // keep failure result
        }
      }
    }
  }

  await postCommandOutputToHub(options.api, options.channel, result, {
    agentName: suggestion.agent_name,
    replyToMessageId: suggestion.message_id,
  });
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
