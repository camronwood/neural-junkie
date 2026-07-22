import type { RefObject } from 'react';
import type { AgentInfo, ThinkingAgent } from '../../types/protocol';
import type { ComposerMode } from '../../constants/composerMode';
import { TypingIndicator } from '../TypingIndicator';
import { TurnTelemetryDrawer } from '../TurnTelemetryDrawer';
import { ComposerModeControl } from '../ComposerModeControl';
import { RichTextInput } from '../RichTextInput';
import { useSettingsStore } from '../../stores/settingsStore';

interface ChatInputAreaProps {
  channel: string;
  channelHeld: boolean;
  hasPendingUserQuestion?: boolean;
  thinkingAgentsForChannel: ThinkingAgent[];
  showAgentStop: boolean;
  onChannelInterject: () => void;
  composerMode: ComposerMode;
  composerModeDisabled: boolean;
  onComposerModeChange: (mode: ComposerMode) => void;
  onSend: (content: string, metadata?: Record<string, unknown>) => void | Promise<void | boolean>;
  inputDisabled: boolean;
  inputPlaceholder: string;
  agents: AgentInfo[];
  inputRef: RefObject<HTMLTextAreaElement>;
  composerDraft: string;
  onDraftChange: (draft: string) => void;
  onAttachmentStateChange?: (hasAttachments: boolean) => void;
  showContextIndicator: boolean;
  contextIndicatorLabel: string;
  contextScopeReason?: string;
  ideRoutingLabel?: string | null;
}

export function ChatInputArea({
  channel,
  channelHeld,
  hasPendingUserQuestion = false,
  thinkingAgentsForChannel,
  showAgentStop,
  onChannelInterject,
  composerMode,
  composerModeDisabled,
  onComposerModeChange,
  onSend,
  inputDisabled,
  inputPlaceholder,
  agents,
  inputRef,
  composerDraft,
  onDraftChange,
  onAttachmentStateChange,
  showContextIndicator,
  contextIndicatorLabel,
  contextScopeReason,
  ideRoutingLabel,
}: ChatInputAreaProps) {
  const showTurnTelemetryDrawer = useSettingsStore(
    (s) => s.layoutSettings.showTurnTelemetryDrawer === true,
  );

  return (
    <div className="flex-shrink-0 flex flex-col min-h-0">
      <TurnTelemetryDrawer channel={channel} enabled={showTurnTelemetryDrawer} />

      <TypingIndicator
        agents={thinkingAgentsForChannel}
        showStop={showAgentStop}
        onStop={onChannelInterject}
      />

      {hasPendingUserQuestion && (
        <div
          className="mx-3 mb-1 px-3 py-2 rounded-md text-sm border border-sky-700/50 bg-sky-950/40 text-sky-100"
          role="status"
        >
          Agent question pending — type your answer in the composer (or use the card options) to continue.
        </div>
      )}

      {channelHeld && !hasPendingUserQuestion && (
        <div
          className="mx-3 mb-1 px-3 py-2 rounded-md text-sm border border-amber-700/50 bg-amber-950/40 text-amber-100"
          role="status"
        >
          Agents paused — send a message to continue.
        </div>
      )}

      <ComposerModeControl
        mode={composerMode}
        disabled={composerModeDisabled}
        onChange={onComposerModeChange}
      />

      <RichTextInput
        onSend={onSend}
        disabled={inputDisabled}
        placeholder={inputPlaceholder}
        agents={agents}
        ref={inputRef}
        onDraftChange={onDraftChange}
        onAttachmentStateChange={onAttachmentStateChange}
      />

      {showContextIndicator && composerDraft.trim() && (
        <div
          className="px-3 py-1 text-xs text-slack-textMuted border-t border-slack-border"
          title={contextScopeReason}
        >
          Context: <span className="text-slack-text">{contextIndicatorLabel}</span>
          {ideRoutingLabel ? (
            <span className="ml-2 text-slack-accent">{ideRoutingLabel}</span>
          ) : null}
        </div>
      )}
    </div>
  );
}
