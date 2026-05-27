import type { ChatAPI } from '../api/chatAPI';
import type { CommandResult } from '../api/terminalAPI';
import type { CommandOutput } from '../types/protocol';

const MAX_STREAM_CHARS = 12_000;

function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.slice(0, max) + '\n... (truncated)';
}

/** Text shown in chat and included in agent LLM history. */
export function formatCommandOutputContent(
  result: CommandResult,
  agentName?: string
): string {
  const who = agentName?.trim() ? `@${agentName.trim()}` : 'An agent';
  let body = `${who} ran a terminal command.\n\n`;
  body += `Command: \`${result.command}\`\n`;
  body += `Exit code: ${result.exit_code} (${result.success ? 'success' : 'failed'})\n`;
  if (result.stdout?.trim()) {
    body += `\nstdout:\n\`\`\`\n${truncate(result.stdout.trim(), MAX_STREAM_CHARS)}\n\`\`\`\n`;
  }
  if (result.stderr?.trim()) {
    body += `\nstderr:\n\`\`\`\n${truncate(result.stderr.trim(), MAX_STREAM_CHARS)}\n\`\`\`\n`;
  }
  if (!result.stdout?.trim() && !result.stderr?.trim()) {
    body += '\n(no output)\n';
  }
  return body.trim();
}

export function commandResultToProtocolOutput(result: CommandResult): CommandOutput {
  return {
    command: result.command,
    plugin: 'shell',
    exit_code: result.exit_code,
    stdout: result.stdout,
    stderr: result.stderr,
    duration: result.duration_ms * 1_000_000,
    success: result.success,
  };
}

/** Posts command_output to the hub so agents and humans see results in channel history. */
export async function postCommandOutputToHub(
  api: ChatAPI,
  channel: string,
  result: CommandResult,
  options?: { agentName?: string; replyToMessageId?: string }
): Promise<void> {
  const content = formatCommandOutputContent(result, options?.agentName);
  const metadata: Record<string, unknown> = {
    command_output: JSON.stringify(commandResultToProtocolOutput(result)),
  };
  if (options?.replyToMessageId) {
    metadata.reply_to = options.replyToMessageId;
  }
  if (options?.agentName) {
    metadata.suggested_by_agent = options.agentName;
  }

  await api.sendMessage(
    channel,
    content,
    { name: 'Terminal', type: 'general' },
    'command_output',
    metadata
  );
}
