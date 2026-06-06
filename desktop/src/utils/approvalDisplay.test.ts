import { describe, expect, it } from 'vitest';
import type { PendingToolApproval } from '../stores/approvalStore';
import type { CommandSuggestion } from '../stores/terminalStore';
import {
  formatCommandSuggestionSummary,
  formatToolApprovalSummary,
  isShellToolApproval,
} from './approvalDisplay';

describe('approvalDisplay', () => {
  it('formats shell tool approval from command field', () => {
    const approval: PendingToolApproval = {
      id: 'a1',
      agentId: 'cursor',
      agentName: 'Cursor',
      toolName: 'run_shell_command',
      toolInput: { command: 'cat README.md' },
      channel: 'general',
      createdAt: new Date().toISOString(),
    };
    expect(formatToolApprovalSummary(approval)).toBe('cat README.md');
    expect(isShellToolApproval(approval)).toBe(true);
  });

  it('formats file tool approval from path field', () => {
    const approval: PendingToolApproval = {
      id: 'a2',
      agentId: 'cursor',
      agentName: 'Cursor',
      toolName: 'edit_file',
      toolInput: { path: 'src/App.tsx' },
      channel: 'general',
      createdAt: new Date().toISOString(),
    };
    expect(formatToolApprovalSummary(approval)).toBe('src/App.tsx');
    expect(isShellToolApproval(approval)).toBe(false);
  });

  it('formats command suggestion summary', () => {
    const suggestion: CommandSuggestion = {
      id: 's1',
      command: 'grep schema resource-api/',
      plugin: 'shell',
      description: 'inspect schema',
      is_safe: true,
      agent_name: 'BackendEngineer',
      message_id: 'm1',
      created_at: new Date().toISOString(),
    };
    expect(formatCommandSuggestionSummary(suggestion)).toBe('grep schema resource-api/');
  });
});
