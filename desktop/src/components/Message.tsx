import { useState, memo, useEffect } from 'react';
import type { Message as MessageType, CommandOutput as CommandOutputType, MessageErrorMetadata, ThreadMetadata } from '../types/protocol';
import {
  getAgentColor,
  getReasoningText,
  isSystemMessage,
  isCollaborationMessage,
} from '../types/protocol';
import { MessageContent } from './MessageContent';
import { CommandOutput } from './CommandOutput';
import { DesignOutput } from './DesignOutput';
import { ToolApprovalCard } from './ToolApprovalCard';
import { ChatAPI } from '../api/chatAPI';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { useToastStore } from '../stores/toastStore';
import { useChatStore } from '../stores/chatStore';
import { USER_IMAGES_METADATA_KEY } from '../constants/promptMetadata';

function MessageUserImages({ metadata }: { metadata?: Record<string, unknown> }) {
  const raw = metadata?.[USER_IMAGES_METADATA_KEY];
  if (!Array.isArray(raw) || raw.length === 0) return null;
  return (
    <div className="flex flex-wrap gap-2 mb-2">
      {raw.map((item, i) => {
        const obj = item as Record<string, unknown>;
        if (obj.redacted === true) {
          return (
            <span
              key={i}
              className="text-xs px-2 py-1 rounded bg-slack-bgHover border border-slack-border text-slack-textMuted"
              title="Image redacted in history fetch"
            >
              Image ({String(obj.mime || 'unknown')})
              {typeof obj.approx_bytes === 'number' ? ` ~${obj.approx_bytes}B` : ''}
            </span>
          );
        }
        const mime = String(obj.mime || 'image/png');
        const data = String(obj.data || '');
        if (!data) return null;
        return (
          <img
            key={i}
            src={`data:${mime};base64,${data}`}
            className="max-h-36 rounded border border-slack-border object-contain bg-slack-bgHover"
            alt=""
          />
        );
      })}
    </div>
  );
}

function MessageReasoningBlock({
  reasoningText,
  isStreaming,
}: {
  reasoningText: string;
  isStreaming?: boolean;
}) {
  const [expanded, setExpanded] = useState(() => !!isStreaming);
  useEffect(() => {
    if (!isStreaming) {
      setExpanded(false);
    }
  }, [isStreaming]);
  if (!reasoningText.trim()) return null;

  const showExpanded = expanded;

  return (
    <div className="mb-2 rounded border border-slack-border/80 bg-slack-bgHover/60 text-sm">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-slack-textMuted hover:text-slack-text transition-colors"
        aria-expanded={showExpanded}
      >
        <span className="text-xs select-none">{showExpanded ? '▼' : '▶'}</span>
        <span className="font-medium text-xs uppercase tracking-wide">Reasoning</span>
        {isStreaming && !showExpanded && (
          <span className="text-xs italic">(in progress)</span>
        )}
      </button>
      {showExpanded && (
        <div className="px-3 pb-3 text-slack-textMuted whitespace-pre-wrap break-words border-t border-slack-border/50">
          {reasoningText}
        </div>
      )}
    </div>
  );
}

function MessageGeneratedImage({ metadata }: { metadata?: Record<string, unknown> }) {
  const g = metadata?.generated_image as Record<string, unknown> | undefined;
  if (!g) return null;
  if (g.data_redacted === true) {
    return (
      <span className="text-xs px-2 py-1 rounded bg-slack-bgHover border border-slack-border text-slack-textMuted mb-2 inline-block">
        Generated image (redacted in history)
      </span>
    );
  }
  const mime = String(g.mime || 'image/png');
  const data = String(g.data || '');
  if (!data) return null;
  return (
    <div className="mb-2">
      <img
        src={`data:${mime};base64,${data}`}
        className="max-h-48 rounded border border-slack-border object-contain bg-slack-bgHover"
        alt="Generated"
      />
    </div>
  );
}

interface MessageProps {
  message: MessageType;
  threadMetadata?: ThreadMetadata;
  onOpenThread?: (threadId: string) => void;
  isStreaming?: boolean;
}

function MessageImpl({ message, threadMetadata, onOpenThread, isStreaming }: MessageProps) {
  const [proposing, setProposing] = useState(false);
  const isSystem = isSystemMessage(message.type);
  const isCommandOutput = message.type === 'command_output';
  const isDesignOutput = message.type === 'design_output';
  const isToolApproval = message.type === 'tool_approval';
  const isCollab = isCollaborationMessage(message);
  const agentColor = getAgentColor(message.from.type);
  const timestamp = new Date(message.timestamp).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
  const errorMeta = (message.metadata ?? {}) as MessageErrorMetadata;
  const hasErrorMeta = typeof errorMeta.error_code === 'string';
  const canRetry = errorMeta.retryable === true;
  const username = useChatStore(s => s.username);
  const addToast = useToastStore(s => s.addToast);
  const fetchPendingChanges = useFileChangeStore(s => s.fetchPendingChanges);

  const suggestsFileChange = shouldShowProposeAction(message);
  const canShowProposeButton = suggestsFileChange && !isStreaming;
  const reasoningText = getReasoningText(message.metadata as Record<string, unknown> | undefined);

  // Parse command output from metadata if present
  let commandOutput: CommandOutputType | null = null;
  if (isCommandOutput && message.metadata?.command_output) {
    try {
      commandOutput = JSON.parse(message.metadata.command_output as string);
    } catch (e) {
      console.error('Failed to parse command output:', e);
    }
  }

  const handleProposeFromMessage = async () => {
    if (proposing) return;

    const api = new ChatAPI();
    const activeTab = useEditorStore.getState().getActiveTab();
    const explorer = useFileExplorerStore.getState();
    const activeWorkspace = explorer.getActiveWorkspace();

    const workspaceId = activeTab?.workspaceId || activeWorkspace?.id;
    const targetPath = activeTab?.path || (explorer.selectedPath || '');

    if (!workspaceId || !targetPath) {
      addToast({
        type: 'warning',
        title: 'Open a target file first',
        message: 'Open the file you want to update, then click "Propose it?" again.',
      });
      return;
    }

    setProposing(true);
    try {
      await api.proposeFileChangeFromMessage({
        channel: message.channel,
        messageId: message.id,
        workspaceId,
        targetPath,
        userId: username || 'default',
      });
      await fetchPendingChanges(username || 'default');
      addToast({
        type: 'success',
        title: 'Proposal created',
        message: 'A new file-change proposal is ready for review in Pending Changes.',
      });
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to create proposal from message';
      addToast({
        type: 'error',
        title: 'Propose failed',
        message: msg,
      });
    } finally {
      setProposing(false);
    }
  };

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
        borderLeft: isSystem ? 'none' : `3px solid ${isCollab ? '#8b5cf6' : agentColor}`,
        backgroundColor: isCollab ? 'rgba(139, 92, 246, 0.04)' : undefined,
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
        {isCollab && (
          <span
            className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded"
            style={{ backgroundColor: 'rgba(139, 92, 246, 0.15)', color: '#a78bfa' }}
          >
            🤝 collab
          </span>
        )}
        {hasErrorMeta && (
          <span
            className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded bg-amber-500/15 text-amber-400"
            title={canRetry ? 'The operation can usually be retried.' : 'The operation likely needs configuration changes.'}
          >
            error: {errorMeta.error_code}
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
            <MessageUserImages metadata={message.metadata as Record<string, unknown> | undefined} />
            <MessageGeneratedImage metadata={message.metadata as Record<string, unknown> | undefined} />
            <MessageReasoningBlock reasoningText={reasoningText} isStreaming={isStreaming} />
            <MessageContent content={message.content} isStreaming={isStreaming} />
            {isStreaming && (
              <span className="inline-block w-2 h-4 ml-0.5 bg-slack-text animate-pulse rounded-sm align-text-bottom" />
            )}
            {commandOutput && <CommandOutput output={commandOutput} />}
            {canShowProposeButton && (
              <div className="mt-3">
                <button
                  type="button"
                  onClick={handleProposeFromMessage}
                  disabled={proposing}
                  className="px-3 py-1.5 text-xs rounded border border-slack-border bg-slack-bgHover hover:bg-slack-accent hover:text-white transition-colors disabled:opacity-50"
                  title="Create a pending file change proposal from this message"
                >
                  {proposing ? 'Creating proposal...' : 'Propose it?'}
                </button>
              </div>
            )}
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

export const Message = memo(MessageImpl, (prev, next) => {
  return (
    prev.message === next.message &&
    prev.isStreaming === next.isStreaming &&
    prev.threadMetadata === next.threadMetadata &&
    prev.onOpenThread === next.onOpenThread
  );
});

function shouldShowProposeAction(message: MessageType): boolean {
  if (message.type !== 'chat') return false;
  if (message.from.type === 'human' || message.from.type === 'general') return false;

  const content = (message.content || '').toLowerCase();
  if (!content.trim()) return false;
  if (content.includes('i submitted a file change proposal')) return false;

  // Only show when the agent suggests making/proposing changes to a file.
  const suggestivePhrase = /would you like me to|want me to|i can submit|i can propose|should i propose|propose.*approval|submit.*approval/.test(content);
  const fileChangeContext = /(readme|file|edit|update|change|proposal)/.test(content);

  return suggestivePhrase && fileChangeContext;
}

