import type { TaskActionSpec } from '../types/protocol';

export const RUNBOOK_ACTION_TYPES = [
  { value: 'http_get', label: 'HTTP GET' },
  { value: 'http_post', label: 'HTTP POST' },
  { value: 'webhook', label: 'Webhook' },
  { value: 'web_search', label: 'Web search' },
  { value: 'slack_message', label: 'Slack message' },
  { value: 'sms', label: 'SMS' },
  { value: 'mcp_tool', label: 'MCP tool' },
  { value: 'shell', label: 'Shell command' },
  { value: 'wait_human', label: 'Wait for human approval' },
  { value: 'git_status', label: 'Git status' },
  { value: 'git_diff', label: 'Git diff' },
] as const;

export function defaultActionConfig(type: string): Record<string, unknown> {
  switch (type) {
    case 'http_get':
      return { url: '' };
    case 'http_post':
      return { url: '', body: {} };
    case 'webhook':
      return { url: '', payload: {} };
    case 'web_search':
      return { query: '' };
    case 'slack_message':
      return { channel_id: '', text: '' };
    case 'sms':
      return { to: '', body: '' };
    case 'shell':
      return { command: '' };
    case 'mcp_tool':
      return { tool: '', arguments: {} };
    case 'wait_human':
      return { prompt: 'Approve to continue' };
    case 'git_diff':
      return { path: '' };
    default:
      return {};
  }
}

export function defaultActionSpec(type: string): TaskActionSpec {
  return { type, config: defaultActionConfig(type) };
}

export function actionConfigString(config: Record<string, unknown> | undefined, key: string): string {
  const v = config?.[key];
  if (v == null) return '';
  return typeof v === 'string' ? v : String(v);
}
