import type { PendingToolApproval } from '../stores/approvalStore';
import type { CommandSuggestion } from '../stores/terminalStore';

export function formatToolApprovalSummary(approval: PendingToolApproval): string {
  const input = approval.toolInput ?? {};
  if (approval.toolName === 'run_shell_command' || approval.toolName === 'shell') {
    const cmd = typeof input.command === 'string' ? input.command : '';
    return cmd || approval.toolName;
  }
  if (
    approval.toolName === 'read_file' ||
    approval.toolName === 'write_file' ||
    approval.toolName === 'edit_file' ||
    approval.toolName === 'list_directory'
  ) {
    const path = typeof input.path === 'string' ? input.path : '';
    return path || approval.toolName;
  }
  const firstKey = Object.keys(input)[0];
  if (firstKey) {
    return `${firstKey}: ${String(input[firstKey])}`;
  }
  return approval.toolName;
}

export function formatCommandSuggestionSummary(suggestion: CommandSuggestion): string {
  return suggestion.command;
}

export function isShellToolApproval(approval: PendingToolApproval): boolean {
  return approval.toolName === 'run_shell_command' || approval.toolName === 'shell';
}
