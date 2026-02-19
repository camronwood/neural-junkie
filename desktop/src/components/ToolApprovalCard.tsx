import { useState, useRef } from 'react';
import type { Message } from '../types/protocol';
import { ChatAPI } from '../api/chatAPI';

interface ToolApprovalCardProps {
  message: Message;
}

const toolIcons: Record<string, string> = {
  read_file: '📖',
  write_file: '📝',
  edit_file: '✏️',
  list_directory: '📁',
  search_files: '🔍',
  run_shell_command: '💻',
  shell: '💻',
  read_many_files: '📚',
};

export function ToolApprovalCard({ message }: ToolApprovalCardProps) {
  const [loading, setLoading] = useState(false);
  const apiRef = useRef(new ChatAPI());

  const approvalId = message.metadata?.approval_id as string;
  const toolName = message.metadata?.tool_name as string;
  const toolInput = message.metadata?.tool_input as Record<string, any> | undefined;
  const status = message.metadata?.status as string;
  const reason = message.metadata?.reason as string | undefined;

  const isPending = status === 'pending';
  const isApproved = status === 'approved';
  const isRejected = status === 'rejected' || status === 'expired';
  const icon = toolIcons[toolName] || '🔧';

  const handleApprove = async () => {
    if (!approvalId) return;
    setLoading(true);
    try {
      await apiRef.current.approveToolCall(approvalId);
    } catch (err) {
      console.error('Failed to approve tool call:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleReject = async () => {
    if (!approvalId) return;
    setLoading(true);
    try {
      await apiRef.current.rejectToolCall(approvalId);
    } catch (err) {
      console.error('Failed to reject tool call:', err);
    } finally {
      setLoading(false);
    }
  };

  const formatInput = () => {
    if (!toolInput) return null;

    if (toolName === 'read_file' || toolName === 'write_file' || toolName === 'edit_file') {
      return toolInput.path as string;
    }
    if (toolName === 'run_shell_command' || toolName === 'shell') {
      return toolInput.command as string;
    }
    if (toolName === 'list_directory') {
      return toolInput.path as string;
    }
    if (toolName === 'search_files') {
      return `"${toolInput.query || toolInput.pattern}" in ${toolInput.path || '.'}`;
    }

    const firstKey = Object.keys(toolInput)[0];
    return firstKey ? `${firstKey}: ${toolInput[firstKey]}` : null;
  };

  const inputDisplay = formatInput();

  return (
    <div className={`my-2 rounded-lg border ${
      isPending ? 'border-yellow-500/40 bg-yellow-500/5' :
      isApproved ? 'border-green-500/30 bg-green-500/5' :
      'border-red-500/30 bg-red-500/5'
    } p-3`}>
      <div className="flex items-start gap-3">
        <span className="text-xl">{icon}</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-sm font-semibold text-slack-text">{toolName}</span>
            {isPending && (
              <span className="text-xs px-1.5 py-0.5 rounded bg-yellow-500/20 text-yellow-400 animate-pulse">
                awaiting approval
              </span>
            )}
            {isApproved && (
              <span className="text-xs px-1.5 py-0.5 rounded bg-green-500/20 text-green-400">
                approved
              </span>
            )}
            {isRejected && (
              <span className="text-xs px-1.5 py-0.5 rounded bg-red-500/20 text-red-400">
                {status === 'expired' ? 'expired' : 'rejected'}
              </span>
            )}
          </div>

          {inputDisplay && (
            <code className="text-xs text-slack-textMuted bg-slack-bgHover px-1.5 py-0.5 rounded break-all">
              {inputDisplay}
            </code>
          )}

          {reason && isRejected && (
            <p className="text-xs text-red-400 mt-1">{reason}</p>
          )}
        </div>

        {isPending && (
          <div className="flex gap-1.5 flex-shrink-0">
            <button
              onClick={handleApprove}
              disabled={loading}
              className="px-3 py-1 text-xs font-medium rounded bg-green-600 hover:bg-green-500 text-white disabled:opacity-50 transition-colors"
            >
              {loading ? '...' : 'Approve'}
            </button>
            <button
              onClick={handleReject}
              disabled={loading}
              className="px-3 py-1 text-xs font-medium rounded bg-red-600 hover:bg-red-500 text-white disabled:opacity-50 transition-colors"
            >
              {loading ? '...' : 'Reject'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
