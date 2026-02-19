import { useState, useEffect } from 'react';
import { useChatStore } from '../stores/chatStore';
import { ChatAPI } from '../api/chatAPI';
import type { AgentInfo } from '../types/protocol';

interface RemovedAgentsPanelProps {
  onClose: () => void;
}

export function RemovedAgentsPanel({ onClose }: RemovedAgentsPanelProps) {
  const { 
    serverAddr, 
    channel, 
    username, 
    removedAgents, 
    setRemovedAgents 
  } = useChatStore();
  
  const [loading, setLoading] = useState(true);
  const [recallingAgent, setRecallingAgent] = useState<string | null>(null);

  const [api] = useState(() => new ChatAPI(serverAddr));

  // Load removed agents on mount
  useEffect(() => {
    const loadRemovedAgents = async () => {
      try {
        setLoading(true);
        const agents = await api.fetchRemovedAgents();
        setRemovedAgents(agents);
      } catch (error) {
        console.error('Failed to load removed agents:', error);
      } finally {
        setLoading(false);
      }
    };

    loadRemovedAgents();
  }, [api, setRemovedAgents]);

  const handleRecallAgent = async (agent: AgentInfo) => {
    try {
      setRecallingAgent(agent.id);
      
      // Send recall command via API
      await api.recallAgent(
        channel,
        agent.name,
        { name: username, type: 'human' }
      );

      // Close panel after successful recall
      onClose();
    } catch (error) {
      console.error('Failed to recall agent:', error);
    } finally {
      setRecallingAgent(null);
    }
  };

  const formatLastActive = (lastActive: string): string => {
    const date = new Date(lastActive);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMinutes = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    
    if (diffMinutes < 1) return 'Just now';
    if (diffMinutes < 60) return `${diffMinutes}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  const getAgentTypeColor = (type: string): string => {
    switch (type) {
      case 'repo': return '#52b6ef';
      case 'helper': return '#af77ca';
      case 'confluence': return '#f09348';
      case 'frontend': return '#61dafb';
      case 'backend': return '#68d391';
      case 'devops': return '#f6ad55';
      case 'database': return '#9f7aea';
      case 'security': return '#f56565';
      default: return '#a9b9ba';
    }
  };

  const getAgentTypeIcon = (type: string): string => {
    switch (type) {
      case 'repo': return '📁';
      case 'helper': return '🤖';
      case 'confluence': return '📚';
      case 'frontend': return '🎨';
      case 'backend': return '⚙️';
      case 'devops': return '🚀';
      case 'database': return '🗄️';
      case 'security': return '🔒';
      default: return '❓';
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex">
      {/* Backdrop */}
      <div 
        className="flex-1 bg-black bg-opacity-50" 
        onClick={onClose}
      />
      
      {/* Panel */}
      <div className="w-96 bg-slack-bg border-l border-slack-border flex flex-col animate-slide-in-right">
        {/* Header */}
        <div className="p-4 border-b border-slack-border bg-slack-bgHover">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-bold text-slack-text flex items-center gap-2">
              <span>🚪</span>
              <span>Removed Agents</span>
            </h2>
            <button
              onClick={onClose}
              className="text-slack-textMuted hover:text-slack-text transition-colors"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-2">
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <div className="text-slack-textMuted">Loading removed agents...</div>
            </div>
          ) : removedAgents.length === 0 ? (
            <div className="p-6 text-center">
              <div className="text-4xl mb-3">🚪</div>
              <h3 className="text-lg font-medium text-slack-text mb-2">No Removed Agents</h3>
              <p className="text-sm text-slack-textMuted mb-4">
                When you remove agents from the conversation, they appear here so you can recall them later.
              </p>
              <div className="bg-slack-bgHover rounded p-4 text-left text-sm">
                <p className="font-medium text-slack-text mb-2">To remove an agent:</p>
                <code className="block bg-slack-bg p-2 rounded mb-3">/remove-agent &lt;agent-name&gt;</code>
                <p className="font-medium text-slack-text mb-2">To recall an agent:</p>
                <code className="block bg-slack-bg p-2 rounded">/recall-agent &lt;agent-name&gt;</code>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {removedAgents.map((agent, index) => (
                <div
                  key={`${agent.type}-${agent.name}-${index}`}
                  className="p-3 rounded bg-slack-bgHover border border-slack-border hover:border-slack-accent transition-colors"
                >
                  <div className="flex items-start gap-3">
                    {/* Type Icon */}
                    <div 
                      className="w-8 h-8 rounded flex items-center justify-center text-white text-sm flex-shrink-0"
                      style={{ backgroundColor: getAgentTypeColor(agent.type) }}
                    >
                      {getAgentTypeIcon(agent.type)}
                    </div>
                    
                    {/* Agent Info */}
                    <div className="flex-1 min-w-0">
                      <div className="font-medium text-slack-text text-sm truncate">
                        {agent.name}
                      </div>
                      
                      {/* Type Badge */}
                      <div className="mt-1">
                        <span 
                          className="text-xs px-2 py-1 rounded"
                          style={{
                            backgroundColor: `${getAgentTypeColor(agent.type)}20`,
                            color: getAgentTypeColor(agent.type),
                          }}
                        >
                          {agent.type}
                        </span>
                      </div>
                      
                      {/* Last Active */}
                      {agent.last_active_time && (
                        <div className="text-xs text-slack-textMuted mt-1">
                          Last active: {formatLastActive(agent.last_active_time)}
                        </div>
                      )}
                      
                      {/* Removed From */}
                      {agent.removed_from && agent.removed_from.length > 0 && (
                        <div className="text-xs text-slack-textMuted mt-1">
                          Removed from: {agent.removed_from.join(', ')}
                        </div>
                      )}
                    </div>
                    
                    {/* Recall Button */}
                    <button
                      onClick={() => handleRecallAgent(agent)}
                      disabled={recallingAgent === agent.id}
                      className="px-3 py-1 bg-slack-accent hover:bg-slack-accentHover text-white text-xs rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-shrink-0"
                    >
                      {recallingAgent === agent.id ? 'Recalling...' : 'Recall'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-3 border-t border-slack-border text-xs text-slack-textMuted text-center">
          {removedAgents.length} removed agent{removedAgents.length !== 1 ? 's' : ''}
        </div>
      </div>
    </div>
  );
}
