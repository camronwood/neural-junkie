import type { ThinkingAgent } from '../types/protocol';
import { getAgentColor } from '../types/protocol';

interface TypingIndicatorProps {
  agents: ThinkingAgent[];
}

export function TypingIndicator({ agents }: TypingIndicatorProps) {
  if (agents.length === 0) {
    return null;
  }

  return (
    <div className="px-6 py-3 bg-gradient-to-r from-slack-bgHover to-slack-bg border-t border-slack-border shadow-sm">
      <div className="flex flex-wrap items-center gap-3">
        {agents.map((agent) => (
          <div
            key={agent.id}
            className="flex items-center gap-2 animate-fadeIn"
          >
            {/* Agent Badge with Pulse */}
            <div
              className="flex items-center justify-center w-7 h-7 rounded-full text-white text-xs font-bold shadow-md animate-pulse ring-2 ring-white/20"
              style={{ backgroundColor: getAgentColor(agent.type) }}
            >
              {agent.name.charAt(0).toUpperCase()}
            </div>
            
            {/* Agent Name with Animated Dots */}
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
    </div>
  );
}

