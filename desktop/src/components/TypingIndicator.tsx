import type { ThinkingAgent } from '../types/protocol';
import { getAgentColor } from '../types/protocol';

interface TypingIndicatorProps {
  agents: ThinkingAgent[];
  showStop?: boolean;
  onStop?: () => void;
  stopDisabled?: boolean;
}

export function TypingIndicator({ agents, showStop, onStop, stopDisabled }: TypingIndicatorProps) {
  if (agents.length === 0 && !showStop) {
    return null;
  }

  return (
    <div className="px-6 py-3 bg-gradient-to-r from-slack-bgHover to-slack-bg border-t border-slack-border shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3 min-w-0">
          {agents.map((agent) => (
            <div
              key={agent.id}
              className="flex items-center gap-2 animate-fadeIn"
            >
              <div
                className="flex items-center justify-center w-7 h-7 rounded-full text-white text-xs font-bold shadow-md animate-pulse ring-2 ring-white/20"
                style={{ backgroundColor: getAgentColor(agent.type) }}
              >
                {agent.name.charAt(0).toUpperCase()}
              </div>
              <div className="flex items-center gap-1">
                <span className="text-sm font-medium text-slack-text">{agent.name}</span>
                <span className="text-sm text-slack-textMuted">is thinking</span>
                <span className="inline-flex">
                  <span className="animate-bounce text-slack-accent font-bold" style={{ animationDelay: '0ms' }}>.</span>
                  <span className="animate-bounce text-slack-accent font-bold" style={{ animationDelay: '150ms' }}>.</span>
                  <span className="animate-bounce text-slack-accent font-bold" style={{ animationDelay: '300ms' }}>.</span>
                </span>
              </div>
            </div>
          ))}
        </div>
        {showStop && onStop ? (
          <button
            type="button"
            onClick={onStop}
            disabled={stopDisabled}
            className="flex-shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium bg-slack-bg border border-slack-border text-slack-text hover:bg-slack-bgHover disabled:opacity-50 disabled:cursor-not-allowed"
            title="Stop agents (Esc)"
            aria-label="Stop agents"
          >
            <span className="inline-block w-3 h-3 bg-red-500 rounded-sm" aria-hidden />
            Stop
          </button>
        ) : null}
      </div>
    </div>
  );
}
