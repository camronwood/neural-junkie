import { useEffect, useState } from 'react';
import type { ThinkingAgent } from '../types/protocol';
import {
  THINKING_ACTIVITY_GENERATING_IMAGE,
  THINKING_ACTIVITY_GENERATING_MUSIC,
  getAgentColor,
} from '../types/protocol';
import { formatThinkingActivityLabel, formatToolStepLabel } from '../utils/thinkingActivityLabel';

interface TypingIndicatorProps {
  agents: ThinkingAgent[];
  showStop?: boolean;
  onStop?: () => void;
  stopDisabled?: boolean;
}

const LONG_RUNNING_ACTIVITIES = new Set([
  THINKING_ACTIVITY_GENERATING_IMAGE,
  THINKING_ACTIVITY_GENERATING_MUSIC,
  'implementation',
  'verifying',
]);

/** Collapse when more than this many agents are thinking at once. */
const COLLAPSE_THRESHOLD = 2;

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

function elapsedLabel(startedAt?: number): string | null {
  if (!startedAt) return null;
  const sec = Math.max(0, Math.floor((Date.now() - startedAt) / 1000));
  if (sec < 5) return null;
  return `${sec}s`;
}

function AgentTypingRow({ agent }: { agent: ThinkingAgent }) {
  const [, tick] = useState(0);
  const showElapsed =
    agent.startedAt &&
    (LONG_RUNNING_ACTIVITIES.has(agent.activity ?? '') || (agent.toolSteps?.length ?? 0) > 0);

  useEffect(() => {
    if (!showElapsed) return;
    const id = window.setInterval(() => tick((n) => n + 1), 1000);
    return () => window.clearInterval(id);
  }, [showElapsed, agent.startedAt]);

  const elapsed = showElapsed ? elapsedLabel(agent.startedAt) : null;
  const suppressDots =
    agent.activity === THINKING_ACTIVITY_GENERATING_IMAGE ||
    agent.activity === THINKING_ACTIVITY_GENERATING_MUSIC;

  return (
    <div className="animate-fadeIn min-w-0">
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
          {elapsed ? (
            <span className="text-xs text-slack-textMuted font-mono tabular-nums">({elapsed})</span>
          ) : null}
          {!suppressDots && (
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
  );
}

function AvatarStack({ agents }: { agents: ThinkingAgent[] }) {
  const shown = agents.slice(0, 3);
  return (
    <div className="flex items-center -space-x-1.5 flex-shrink-0" aria-hidden>
      {shown.map((agent, i) => (
        <div
          key={agent.id}
          className="flex items-center justify-center w-7 h-7 rounded-full text-white text-xs font-bold shadow-md ring-2 ring-slack-bg"
          style={{ backgroundColor: getAgentColor(agent.type), zIndex: shown.length - i }}
        >
          {agent.name.charAt(0).toUpperCase()}
        </div>
      ))}
    </div>
  );
}

function collapsedSummary(agents: ThinkingAgent[]): string {
  const lead = agents[0]?.name ?? 'Agent';
  const extra = agents.length - 1;
  if (extra <= 0) return `${lead} responding`;
  return `${lead} + ${extra} more responding`;
}

export function TypingIndicator({ agents, showStop, onStop, stopDisabled }: TypingIndicatorProps) {
  const [expanded, setExpanded] = useState(false);
  const multi = agents.length >= COLLAPSE_THRESHOLD;

  // Collapse again when the crowd clears so the next @here starts compact.
  useEffect(() => {
    if (agents.length < COLLAPSE_THRESHOLD) {
      setExpanded(false);
    }
  }, [agents.length]);

  if (agents.length === 0 && !showStop) {
    return null;
  }

  const showDetails = !multi || expanded;

  return (
    <div className="px-6 py-2.5 bg-gradient-to-r from-slack-bgHover to-slack-bg border-t border-slack-border shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-2 min-w-0 flex-1">
          {multi && !expanded ? (
            <div className="flex items-center gap-2 min-w-0 animate-fadeIn">
              <AvatarStack agents={agents} />
              <button
                type="button"
                onClick={() => setExpanded(true)}
                className="flex items-center gap-1.5 min-w-0 text-left group"
                aria-expanded={false}
                aria-label={`Show all ${agents.length} agents responding`}
              >
                <span className="text-sm font-medium text-slack-text truncate">
                  {collapsedSummary(agents)}
                </span>
                <span className="inline-flex" aria-hidden>
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
                <span className="text-xs text-slack-accent group-hover:underline shrink-0">Show</span>
              </button>
            </div>
          ) : null}

          {showDetails ? (
            <div className="flex flex-col gap-2 min-w-0 max-h-36 overflow-y-auto pr-1">
              {multi && expanded ? (
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs text-slack-textMuted">
                    {agents.length} agents responding
                  </span>
                  <button
                    type="button"
                    onClick={() => setExpanded(false)}
                    className="text-xs text-slack-accent hover:underline shrink-0"
                    aria-expanded={true}
                    aria-label="Collapse agent activity list"
                  >
                    Hide
                  </button>
                </div>
              ) : null}
              {agents.map((agent) => (
                <AgentTypingRow key={agent.id} agent={agent} />
              ))}
            </div>
          ) : null}
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
