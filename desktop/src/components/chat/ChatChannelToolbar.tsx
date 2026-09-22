import type { Channel } from '../../types/protocol';
import type { ConnectionStatus } from '../../hooks/useWebSocket';
import { LeftSidebarIcon, RightSidebarIcon, ChatPanelIcon } from '../Icons';
import { ChatToolbarActions, type ChatToolbarActionsProps } from '../ChatToolbarActions';
import {
  isSlackMirrorChannelName,
  showSlackHubChannelIdInHeader,
  slackChannelDisplayName,
} from '../../utils/slackChannelDisplay';
import { formatChord } from '../../shortcuts/format';

function statusColor(status: ConnectionStatus): string {
  switch (status) {
    case 'connected':
      return 'bg-green-500';
    case 'connecting':
      return 'bg-yellow-500';
    case 'error':
      return 'bg-red-500';
    default:
      return 'bg-gray-500';
  }
}

function statusText(status: ConnectionStatus): string {
  switch (status) {
    case 'connected':
      return 'Connected';
    case 'connecting':
      return 'Connecting...';
    case 'error':
      return 'Connection Error';
    default:
      return 'Disconnected';
  }
}

export interface ChatChannelToolbarProps {
  channel: string;
  channels: Channel[];
  status: ConnectionStatus;
  channelSidebarOpen: boolean;
  onToggleChannelSidebar: () => void;
  chatPanelVisible: boolean;
  onToggleChatPanel: () => void;
  useSidebarChips: boolean;
  toolbarSidebarOpen: boolean;
  onToggleToolbarSidebar: () => void;
  showTopToolbarChips: boolean;
  toolbarActionsProps: Omit<ChatToolbarActionsProps, 'layout'>;
}

/** Top channel title + connection status + sidebar toggles (extracted from ChatWindow). */
export function ChatChannelToolbar({
  channel,
  channels,
  status,
  channelSidebarOpen,
  onToggleChannelSidebar,
  chatPanelVisible,
  onToggleChatPanel,
  useSidebarChips,
  toolbarSidebarOpen,
  onToggleToolbarSidebar,
  showTopToolbarChips,
  toolbarActionsProps,
}: ChatChannelToolbarProps) {
  const ch = channels.find((c) => c.name === channel);
  const isDM = ch?.type === 'dm';
  const agentCount = ch?.agents?.length ?? 0;

  return (
    <div className="flex items-center justify-between px-3 py-1.5 border-b border-slack-border bg-slack-bgHover flex-shrink-0">
      <div className="flex items-center gap-2">
        <h1 className="text-sm font-bold text-slack-text">
          {isDM
            ? `@ ${ch?.agents?.[0]?.name ?? channel}`
            : ch && isSlackMirrorChannelName(ch.name)
              ? slackChannelDisplayName(ch)
              : `# ${channel}`}
        </h1>
        {ch && showSlackHubChannelIdInHeader(ch.name) && (
          <span
            className="text-xs text-slack-textMuted hidden sm:inline truncate max-w-[200px] font-mono"
            title="Hub channel id"
          >
            {ch.name}
          </span>
        )}
        {ch?.description && !isSlackMirrorChannelName(ch.name) && (
          <span
            className="text-xs text-slack-textMuted hidden sm:inline truncate max-w-[200px]"
            title={ch.description}
          >
            {ch.description}
          </span>
        )}
        {agentCount > 0 && !isDM && (
          <span className="text-xs text-slack-textMuted bg-slack-bgHover px-1.5 py-0.5 rounded">
            {agentCount} agent{agentCount !== 1 ? 's' : ''}
          </span>
        )}
        <div className="flex items-center gap-1.5 text-xs">
          <div className={`w-1.5 h-1.5 rounded-full ${statusColor(status)}`} />
          <span className="text-slack-textMuted">{statusText(status)}</span>
        </div>
      </div>

      <div
        className="flex items-center gap-1.5 shrink min-w-0 max-w-[min(100%,72rem)] justify-end"
        aria-label="Sidebar toggles"
      >
        <div className="flex shrink-0 items-center gap-1.5">
          <button
            type="button"
            onClick={onToggleChannelSidebar}
            className={`w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 ${
              channelSidebarOpen
                ? 'bg-slack-accent text-white'
                : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text hover:bg-slack-border'
            }`}
            title="Toggle channels sidebar (⌘B)"
            aria-label="Toggle channels sidebar"
            aria-pressed={channelSidebarOpen}
          >
            <LeftSidebarIcon className="w-3.5 h-3.5" />
          </button>
          <button
            type="button"
            onClick={onToggleChatPanel}
            className={`w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 ${
              chatPanelVisible
                ? 'bg-slack-accent text-white'
                : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text hover:bg-slack-border'
            }`}
            title={`${chatPanelVisible ? 'Hide main chat' : 'Show main chat'} (${formatChord('mod+shift+c')})`}
            aria-label={chatPanelVisible ? 'Hide main chat panel' : 'Show main chat panel'}
            aria-pressed={chatPanelVisible}
          >
            <ChatPanelIcon className="w-3.5 h-3.5" />
          </button>
          {useSidebarChips && (
            <button
              type="button"
              onClick={onToggleToolbarSidebar}
              className={`w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 ${
                toolbarSidebarOpen
                  ? 'bg-slack-accent text-white'
                  : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text hover:bg-slack-border'
              }`}
              title={toolbarSidebarOpen ? 'Close toolbar panel' : 'Open toolbar panel'}
              aria-label={toolbarSidebarOpen ? 'Close toolbar panel' : 'Open toolbar panel'}
              aria-pressed={toolbarSidebarOpen}
            >
              <RightSidebarIcon className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
        {showTopToolbarChips && (
          <>
            <div className="w-px h-5 bg-slack-border shrink-0" />
            <div className="flex min-w-0 overflow-x-auto overflow-y-visible">
              <ChatToolbarActions layout="horizontal" {...toolbarActionsProps} />
            </div>
          </>
        )}
      </div>
    </div>
  );
}
