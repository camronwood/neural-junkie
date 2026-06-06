import { useCallback, useMemo, useState } from 'react';
import { useApprovalStore } from '../stores/approvalStore';
import { useTerminalStore, type CommandSuggestion } from '../stores/terminalStore';
import { useChatStore } from '../stores/chatStore';
import { ChatAPI } from '../api/chatAPI';
import {
  formatCommandSuggestionSummary,
  formatToolApprovalSummary,
  isShellToolApproval,
} from '../utils/approvalDisplay';
import { runAgentTerminalCommand } from '../utils/runTerminalCommand';
import type { Collaboration } from '../types/protocol';

interface PendingApprovalsBarProps {
  channel: string;
  api: ChatAPI;
  collaboration?: Collaboration | null;
  onOpenTerminal: () => void;
  onScrollToApproval: (approvalId: string) => void;
}

type QueueItem =
  | { kind: 'tool'; key: string; approvalId: string }
  | { kind: 'command'; key: string; suggestionId: string };

export function PendingApprovalsBar({
  channel,
  api,
  collaboration,
  onOpenTerminal,
  onScrollToApproval,
}: PendingApprovalsBarProps) {
  const pendingTools = useApprovalStore((s) => s.pendingTools);
  const removePendingTool = useApprovalStore((s) => s.removePendingTool);
  const suggestedCommands = useTerminalStore((s) => s.suggestedCommands);
  const removeSuggestedCommand = useTerminalStore((s) => s.removeSuggestedCommand);
  const setChannel = useChatStore((s) => s.setChannel);
  const [busyKey, setBusyKey] = useState<string | null>(null);

  const channelTools = useMemo(
    () => pendingTools.filter((a) => a.channel === channel),
    [pendingTools, channel],
  );
  const otherChannelTools = useMemo(
    () => pendingTools.filter((a) => a.channel !== channel),
    [pendingTools, channel],
  );

  const queue: QueueItem[] = useMemo(() => {
    const items: QueueItem[] = [];
    for (const s of suggestedCommands) {
      items.push({ kind: 'command', key: `cmd:${s.id}`, suggestionId: s.id });
    }
    for (const a of channelTools) {
      items.push({ kind: 'tool', key: `tool:${a.id}`, approvalId: a.id });
    }
    return items;
  }, [suggestedCommands, channelTools]);

  const totalCount = suggestedCommands.length + pendingTools.length;
  if (totalCount === 0) return null;

  const active = queue[0];
  const activeTool =
    active?.kind === 'tool' ? channelTools.find((a) => a.id === active.approvalId) : undefined;
  const activeCommand =
    active?.kind === 'command'
      ? suggestedCommands.find((s) => s.id === active.suggestionId)
      : undefined;

  const runCommand = useCallback(
    async (suggestion: CommandSuggestion) => {
      setBusyKey(`cmd:${suggestion.id}`);
      removeSuggestedCommand(suggestion.id);
      try {
        await runAgentTerminalCommand(suggestion, { collaboration, channel, api });
        onOpenTerminal();
      } finally {
        setBusyKey(null);
      }
    },
    [api, channel, collaboration, onOpenTerminal, removeSuggestedCommand],
  );

  const approveTool = useCallback(
    async (approvalId: string) => {
      setBusyKey(`tool:${approvalId}`);
      try {
        await api.approveToolCall(approvalId);
        removePendingTool(approvalId);
      } finally {
        setBusyKey(null);
      }
    },
    [api, removePendingTool],
  );

  const rejectTool = useCallback(
    async (approvalId: string) => {
      setBusyKey(`tool:${approvalId}`);
      try {
        await api.rejectToolCall(approvalId);
        removePendingTool(approvalId);
      } finally {
        setBusyKey(null);
      }
    },
    [api, removePendingTool],
  );

  const summary = activeTool
    ? formatToolApprovalSummary(activeTool)
    : activeCommand
      ? formatCommandSuggestionSummary(activeCommand)
      : '';

  const agentName = activeTool?.agentName ?? activeCommand?.agent_name ?? 'Agent';
  const isCommand = active?.kind === 'command';
  const isShell = activeTool ? isShellToolApproval(activeTool) : isCommand;

  return (
    <div
      className="shrink-0 border-b border-amber-500/40 bg-amber-500/10 px-3 py-2"
      data-testid="pending-approvals-bar"
      role="status"
      aria-live="polite"
    >
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <span className="relative flex h-2.5 w-2.5 shrink-0">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-60" />
            <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-amber-400" />
          </span>
          <div className="min-w-0">
            <p className="text-xs font-semibold text-amber-100">
              {isShell ? 'Command needs your approval' : 'Tool call needs your approval'}
              {totalCount > 1 ? ` (${totalCount} waiting)` : ''}
            </p>
            <p className="text-[11px] text-amber-200/80 truncate">
              <span className="font-medium text-amber-100">{agentName}</span>
              {summary ? (
                <>
                  {' '}
                  — <code className="font-mono text-amber-50/90">{summary}</code>
                </>
              ) : null}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-1.5 shrink-0">
          {activeTool && (
            <button
              type="button"
              className="px-2 py-1 text-[11px] rounded border border-slack-border text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover"
              onClick={() => onScrollToApproval(activeTool.id)}
            >
              View in chat
            </button>
          )}
          {isCommand && (
            <button
              type="button"
              className="px-2 py-1 text-[11px] rounded border border-slack-border text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover"
              onClick={onOpenTerminal}
            >
              Open terminal
            </button>
          )}
          {activeTool ? (
            <>
              <button
                type="button"
                disabled={busyKey === active.key}
                onClick={() => void rejectTool(activeTool.id)}
                className="px-2.5 py-1 text-[11px] font-medium rounded bg-red-700/80 hover:bg-red-600 text-white disabled:opacity-50"
              >
                Reject
              </button>
              <button
                type="button"
                disabled={busyKey === active.key}
                onClick={() => void approveTool(activeTool.id)}
                className="px-2.5 py-1 text-[11px] font-medium rounded bg-emerald-700 hover:bg-emerald-600 text-white disabled:opacity-50"
              >
                {busyKey === active.key ? '…' : 'Approve'}
              </button>
            </>
          ) : activeCommand ? (
            <>
              <button
                type="button"
                disabled={busyKey === active.key}
                onClick={() => removeSuggestedCommand(activeCommand.id)}
                className="px-2.5 py-1 text-[11px] font-medium rounded bg-slack-bgHover hover:bg-slack-border text-slack-text disabled:opacity-50"
              >
                Dismiss
              </button>
              <button
                type="button"
                disabled={busyKey === active.key}
                onClick={() => void runCommand(activeCommand)}
                className="px-2.5 py-1 text-[11px] font-medium rounded bg-emerald-700 hover:bg-emerald-600 text-white disabled:opacity-50"
              >
                {busyKey === active.key ? 'Running…' : 'Run command'}
              </button>
            </>
          ) : null}
        </div>
      </div>

      {otherChannelTools.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-2">
          {otherChannelTools.map((approval) => (
            <button
              key={approval.id}
              type="button"
              onClick={() => setChannel(approval.channel)}
              className="text-[11px] px-2 py-0.5 rounded bg-amber-500/20 text-amber-100 hover:bg-amber-500/30"
            >
              {approval.agentName} in #{approval.channel}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
