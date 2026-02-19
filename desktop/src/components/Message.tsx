import type { Message as MessageType, CommandOutput as CommandOutputType, ThreadMetadata } from '../types/protocol';
import { getAgentColor, isSystemMessage } from '../types/protocol';
import { MessageContent } from './MessageContent';
import { CommandOutput } from './CommandOutput';
import { DesignOutput } from './DesignOutput';
import { ToolApprovalCard } from './ToolApprovalCard';

interface MessageProps {
  message: MessageType;
  threadMetadata?: ThreadMetadata;
  onOpenThread?: (threadId: string) => void;
  isStreaming?: boolean;
}

export function Message({ message, threadMetadata, onOpenThread, isStreaming }: MessageProps) {
  const isSystem = isSystemMessage(message.type);
  const isCommandOutput = message.type === 'command_output';
  const isDesignOutput = message.type === 'design_output';
  const isToolApproval = message.type === 'tool_approval';
  const agentColor = getAgentColor(message.from.type);
  const timestamp = new Date(message.timestamp).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });

  // Parse command output from metadata if present
  let commandOutput: CommandOutputType | null = null;
  if (isCommandOutput && message.metadata?.command_output) {
    try {
      commandOutput = JSON.parse(message.metadata.command_output as string);
    } catch (e) {
      console.error('Failed to parse command output:', e);
    }
  }

  // Format last reply time for thread indicator
  const formatLastReplyTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (minutes < 1) return 'just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return `${days}d ago`;
  };

  return (
    <div
      className={`group relative px-4 py-2 hover:bg-slack-bgHover transition-colors ${
        isSystem ? 'italic text-slack-textMuted' : ''
      }`}
      style={{
        borderLeft: isSystem ? 'none' : `3px solid ${agentColor}`,
      }}
    >
      {/* Reply in Thread button - shows on hover, only for non-system, non-thread messages */}
      {!isSystem && !message.is_thread_reply && onOpenThread && (
        <div className="absolute top-2 right-4 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={() => onOpenThread(message.id)}
            className="px-2 py-1 text-xs bg-white border border-slack-border rounded shadow-sm hover:shadow-md transition-all text-gray-700 hover:text-slack-link"
            title="Reply in thread"
          >
            💬 Reply in thread
          </button>
        </div>
      )}

      <div className="flex items-baseline gap-2 mb-1">
        <span className="text-xs text-slack-textMuted font-mono">[{timestamp}]</span>
        <span
          className="font-semibold"
          style={{ color: isSystem ? undefined : agentColor }}
        >
          {message.from.name}
        </span>
        {message.from.type && (
          <span className="text-xs px-2 py-0.5 rounded bg-slack-bgHover text-slack-textMuted">
            {message.from.type}
          </span>
        )}
        {message.metadata?.workspace_context && (
          <span
            className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded bg-purple-500/15 text-purple-400"
            title="This message included workspace files"
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
              <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
            </svg>
            files shared
          </span>
        )}
      </div>
      
      <div
        className={`text-slack-text ${
          isSystem ? 'text-sm' : ''
        }`}
      >
        {isToolApproval ? (
          <ToolApprovalCard message={message} />
        ) : isDesignOutput ? (
          <DesignOutput message={message} />
        ) : (
          <>
            <MessageContent content={message.content} />
            {isStreaming && (
              <span className="inline-block w-2 h-4 ml-0.5 bg-slack-text animate-pulse rounded-sm align-text-bottom" />
            )}
            {commandOutput && <CommandOutput output={commandOutput} />}
          </>
        )}
      </div>

      {message.tags && message.tags.length > 0 && (
        <div className="flex gap-1 mt-2">
          {message.tags.map((tag) => (
            <span
              key={tag}
              className="text-xs px-2 py-0.5 rounded bg-agent-default/20 text-agent-default"
            >
              #{tag}
            </span>
          ))}
        </div>
      )}

      {/* Thread indicator - only show for non-thread messages with replies */}
      {!message.is_thread_reply && threadMetadata && threadMetadata.reply_count > 0 && (
        <div className="mt-2 pt-2 border-t border-slack-border">
          <button
            onClick={() => onOpenThread?.(message.id)}
            className="flex items-center gap-2 text-sm text-blue-600 hover:text-blue-700 hover:underline transition-colors"
          >
            <span className="font-semibold">
              {threadMetadata.reply_count} {threadMetadata.reply_count === 1 ? 'reply' : 'replies'}
            </span>
            <span className="text-slack-textMuted">
              Last reply {formatLastReplyTime(threadMetadata.last_reply_time)}
            </span>
            <span className="font-medium">View thread →</span>
          </button>
        </div>
      )}
    </div>
  );
}

