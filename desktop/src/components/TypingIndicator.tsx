import type { ThinkingAgent } from '../types/protocol';
import { THINKING_ACTIVITY_GENERATING_IMAGE, getAgentColor } from '../types/protocol';
import { formatThinkingActivityLabel, formatToolStepLabel } from '../utils/thinkingActivityLabel';

interface TypingIndicatorProps {
  agents: ThinkingAgent[];
  showStop?: boolean;
  onStop?: () => void;
  stopDisabled?: boolean;
}

function AgentActivityHistory({ steps }: { steps: NonNullable<ThinkingAgent['toolSteps']> }) {
  const recent = steps.slice(-5);
  if (recent.length === 0) return null;
  return (
    <ul className="mt-1 ml-9 space-y-0.5 text-xs text-slack-textMuted list-none">
      {recent.map((step, i) => (
        <li key={i} className="truncate font-mono">
          <span className="text-slack-accent/80">▸</span> {formatToolStepLabel(step)}
        </li>
      ))}
    </ul>
  );
}

export function TypingIndicator({ agents, showStop, onStop, stopDisabled }: TypingIndicatorProps) {
  if (agents.length === 0 && !showStop) {
    return null;
  }

  return (
    <div className="px-6 py-3 bg-gradient-to-r from-slack-bgHover to-slack-bg border-t border-slack-border shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-2 min-w-0 flex-1">
          {agents.map((agent) => (
            <div key={agent.id} className="animate-fadeIn min-w-0">
              <div className="flex items-center gap-2">
                <div
                  className="flex-shrink-0 flex items-center justify-center w-7 h-7 rounded-full text-white text-xs font-bold shadow-md animate-pulse ring-2 ring-white/20"
                  style={{ backgroundColor: getAgentColor(agent.type) }}
                >
                  {agent.name.charAt(0).toUpperCase()}
                </div>
                <div className="flex items-center gap-1 min-w-0 flex-wrap">
                  <span className="text-sm font-medium text-slack-text">{agent.name}</span>
                  <span className="text-sm text-slack-textMuted truncate">
                    {formatThinkingActivityLabel(agent.activity, agent.activityDetail)}
                  </span>
                  {agent.activity !== THINKING_ACTIVITY_GENERATING_IMAGE && (
                    <span className="inline-flex">
                      <span className="animate-bounce text-slack-accent font-bold" style={{ animationDelay: '0ms' }}>
                        .
                      </span>
                      <span className="animate-bounce text-slack-accent font-bold" style={{ animationDelay: '150ms' }}>
                        .
                      </span>
                      <span className="animate-bounce text-slack-accent font-bold" style={{ animationDelay: '300ms' }}>
                        .
                      </span>
                    </span>
                  )}
                </div>
              </div>
              {agent.toolSteps && agent.toolSteps.length > 0 ? (
                <AgentActivityHistory steps={agent.toolSteps} />
              ) : null}
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
