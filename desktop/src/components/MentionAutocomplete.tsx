import { useRef, useEffect } from 'react';
import type { AgentInfo } from '../types/protocol';
import { getAgentColor } from '../types/protocol';

interface MentionAutocompleteProps {
  agents: AgentInfo[];
  query: string;
  selectedIndex: number;
  onSelect: (agent: AgentInfo) => void;
  position: { top: number; left: number };
}

export function MentionAutocomplete({
  agents,
  query,
  selectedIndex,
  onSelect,
  position,
}: MentionAutocompleteProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  // Filter agents based on query
  const filteredAgents = agents.filter((agent) =>
    agent.name.toLowerCase().includes(query.toLowerCase())
  );

  // Scroll selected item into view
  useEffect(() => {
    if (containerRef.current) {
      const selectedElement = containerRef.current.children[selectedIndex] as HTMLElement;
      if (selectedElement) {
        selectedElement.scrollIntoView({ block: 'nearest' });
      }
    }
  }, [selectedIndex]);

  if (filteredAgents.length === 0) {
    return null;
  }

  // Highlight matching text
  const highlightMatch = (text: string, query: string) => {
    if (!query) return text;
    
    const index = text.toLowerCase().indexOf(query.toLowerCase());
    if (index === -1) return text;
    
    return (
      <>
        {text.substring(0, index)}
        <span className="bg-slack-accent/30 font-semibold">
          {text.substring(index, index + query.length)}
        </span>
        {text.substring(index + query.length)}
      </>
    );
  };

  return (
    <div
      ref={containerRef}
      className="absolute z-50 bg-slack-bgHover border border-slack-border rounded-lg shadow-xl max-h-64 overflow-y-auto"
      style={{
        bottom: position.top,
        left: position.left,
        minWidth: '280px',
        maxWidth: '400px',
      }}
    >
      {filteredAgents.map((agent, index) => {
        const agentColor = getAgentColor(agent.type);
        const isSelected = index === selectedIndex;
        const isActive = agent.status === 'active';

        return (
          <div
            key={agent.id}
            onClick={() => onSelect(agent)}
            className={`px-3 py-2 cursor-pointer transition-colors ${
              isSelected ? 'bg-slack-accent/20' : 'hover:bg-slack-accent/10'
            }`}
          >
            <div className="flex items-center gap-2">
              {/* Status Indicator */}
              <div
                className={`w-2 h-2 rounded-full flex-shrink-0 ${
                  isActive ? 'animate-pulse' : ''
                }`}
                style={{ backgroundColor: agentColor }}
              />

              {/* Agent Info */}
              <div className="flex-1 min-w-0">
                <div className="font-medium text-slack-text text-sm truncate">
                  {highlightMatch(agent.name, query)}
                </div>
              </div>

              {/* Type Badge */}
              <span
                className="text-xs px-1.5 py-0.5 rounded flex-shrink-0"
                style={{
                  backgroundColor: `${agentColor}20`,
                  color: agentColor,
                }}
              >
                {agent.type}
              </span>

              {/* Paused Badge */}
              {agent.is_paused && (
                <span className="text-xs px-1.5 py-0.5 rounded bg-yellow-500/20 text-yellow-500 flex-shrink-0">
                  paused
                </span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

