import type { Collaboration } from '../../types/protocol';
import { markdownPreviewLine } from '../../utils/markdownPreview';
import { ChatFindBar } from '../ChatFindBar';
import { MessageList } from '../MessageList';
import type { FirstWinCoachActions } from '../FirstWinCoach';
import { RoutingTracePanel } from '../RoutingTracePanel';
import { useSettingsStore } from '../../stores/settingsStore';
import { useChatStore } from '../../stores/chatStore';

interface ChatMessageListProps extends FirstWinCoachActions {
  channel: string;
  messageSearchQuery: string;
  chatFindOpen: boolean;
  onMessageSearchQueryChange: (query: string) => void;
  onCloseFind: () => void;
  isClosedCollaborationChannel: boolean;
  collaborationForChannel: Collaboration | null;
  channelAwaitingWorkspaceCollab: Collaboration | null | undefined;
  onOpenWorkspaceGate: () => void;
}

export function ChatMessageList({
  channel,
  messageSearchQuery,
  chatFindOpen,
  onMessageSearchQueryChange,
  onCloseFind,
  isClosedCollaborationChannel,
  collaborationForChannel,
  channelAwaitingWorkspaceCollab,
  onOpenWorkspaceGate,
  onOpenFiles,
  onOpenCommandPalette,
  onOpenAgentDM,
  onPrefillComposer,
  onOpenModelLibrary,
}: ChatMessageListProps) {
  const showRoutingOnMessages = useSettingsStore(
    (s) => s.layoutSettings.showRoutingOnMessages !== false,
  );
  const highlightMessageId = useChatStore((s) => s.highlightMessageId);

  return (
    <>
      {isClosedCollaborationChannel && collaborationForChannel && (
        <div
          className={`mx-3 mt-2 px-3 py-2 rounded-md text-sm border ${
            collaborationForChannel.phase === 'cancelled'
              ? 'border-red-700/50 bg-red-950/40 text-red-100'
              : 'border-emerald-700/50 bg-emerald-950/40 text-emerald-100'
          }`}
          role="status"
        >
          {collaborationForChannel.phase === 'cancelled' ? (
            <>Collaboration cancelled — this channel is read-only.</>
          ) : (
            <>
              Collaboration complete —{' '}
              {collaborationForChannel.tasks?.filter((t) => t.status === 'completed').length ?? 0}/
              {collaborationForChannel.tasks?.length ?? 0} tasks done.
              {collaborationForChannel.session_recap?.trim() ? (
                <>
                  {' '}
                  {markdownPreviewLine(collaborationForChannel.session_recap, 160)}
                </>
              ) : null}{' '}
              This channel is read-only.
            </>
          )}
        </div>
      )}

      {chatFindOpen && (
        <ChatFindBar
          query={messageSearchQuery}
          onQueryChange={onMessageSearchQueryChange}
          onClose={onCloseFind}
        />
      )}

      {channelAwaitingWorkspaceCollab && (
        <div
          data-testid="chat-workspace-gate-strip"
          className="mx-3 mt-2 mb-1 px-3 py-2 rounded-md border border-amber-600/50 bg-amber-950/30 text-sm text-amber-100 flex items-center justify-between gap-3"
        >
          <span>
            <strong>Confirm workspace</strong> — agents on #{channel} are waiting for your approval before
            task prompts are sent.
          </span>
          <button
            type="button"
            className="shrink-0 px-3 py-1 rounded bg-amber-700 hover:bg-amber-600 text-white text-xs font-semibold"
            onClick={onOpenWorkspaceGate}
          >
            Continue
          </button>
        </div>
      )}

      {highlightMessageId && (
        <RoutingTracePanel
          channel={channel}
          messageId={highlightMessageId}
          query={messageSearchQuery}
          enabled={showRoutingOnMessages}
        />
      )}

      <MessageList
        key={channel}
        searchQuery={messageSearchQuery}
        onOpenFiles={onOpenFiles}
        onOpenCommandPalette={onOpenCommandPalette}
        onOpenAgentDM={onOpenAgentDM}
        onPrefillComposer={onPrefillComposer}
        onOpenModelLibrary={onOpenModelLibrary}
      />
    </>
  );
}
