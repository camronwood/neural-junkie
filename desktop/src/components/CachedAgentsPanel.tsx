import { useState, useEffect } from 'react';
import { useChatStore } from '../stores/chatStore';
import { ChatAPI } from '../api/chatAPI';
import type { CachedAgentInfo, AgentCategory } from '../types/protocol';

interface MyAgentsPanelProps {
  onClose: () => void;
}

export function MyAgentsPanel({ onClose }: MyAgentsPanelProps) {
  const { 
    serverAddr, 
    channel, 
    username, 
    myAgents, 
    setMyAgents
  } = useChatStore();
  
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<AgentCategory>('all');
  const [loading, setLoading] = useState(true);
  const [loadingAgent, setLoadingAgent] = useState<string | null>(null);

  const [api] = useState(() => new ChatAPI(serverAddr));

  // Load my agents on mount
  useEffect(() => {
    const loadMyAgents = async () => {
      try {
        setLoading(true);
        const agents = await api.fetchMyAgents();
        setMyAgents(agents);
      } catch (error) {
        console.error('Failed to load my agents:', error);
      } finally {
        setLoading(false);
      }
    };

    loadMyAgents();
  }, [api, setMyAgents]);

  // Filter and search agents
  const filteredAgents = myAgents.filter(agent => {
    // Type filter
    if (filterType !== 'all' && agent.type !== filterType) {
      return false;
    }

    // Search filter
    if (searchTerm) {
      const searchLower = searchTerm.toLowerCase();
      return (
        agent.name.toLowerCase().includes(searchLower) ||
        agent.path.toLowerCase().includes(searchLower)
      );
    }

    return true;
  });

  // Sort by last used (descending), then by cache size (descending)
  const sortedAgents = [...filteredAgents].sort((a, b) => {
    const dateA = new Date(a.last_used).getTime();
    const dateB = new Date(b.last_used).getTime();
    
    if (dateA !== dateB) {
      return dateB - dateA; // Most recent first
    }
    
    return b.cache_size - a.cache_size; // Larger cache first
  });

  const handleLoadAgent = async (agent: CachedAgentInfo) => {
    try {
      setLoadingAgent(agent.name);
      
      // Construct appropriate command based on agent type
      let command = '';
      switch (agent.type) {
        case 'repo':
          command = `/create-repo-agent ${agent.path} ${agent.name}`;
          break;
        case 'helper':
          // Map agent names to correct template names
          let templateName = '';
          const agentNameLower = agent.name.toLowerCase();
          if (agentNameLower.includes('day one') || agentNameLower.includes('day-one')) {
            templateName = 'day-one';
          } else if (agentNameLower.includes('testing') || agentNameLower.includes('test')) {
            templateName = 'testing-expert';
          } else if (agentNameLower.includes('docs') || agentNameLower.includes('documentation')) {
            templateName = 'docs-expert';
          } else {
            // Fallback: try to extract from name
            templateName = agent.name.toLowerCase().replace(/\s+/g, '-');
          }
          command = `/create-helper ${templateName}`;
          break;
        case 'confluence':
          // Extract space key from path or metadata
          const spaceKey = agent.metadata?.space_key || agent.path;
          command = `/create-confluence-agent ${spaceKey} ${agent.name}`;
          break;
        default:
          console.error('Unknown agent type:', agent.type);
          return;
      }

      // Send command via API
      await api.sendMessage(
        channel,
        command,
        { name: username, type: 'human' },
        'question'
      );

      // Close panel after successful load
      onClose();
    } catch (error) {
      console.error('Failed to load agent:', error);
    } finally {
      setLoadingAgent(null);
    }
  };

  const formatCacheSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
  };

  const formatLastUsed = (lastUsed: string): string => {
    const date = new Date(lastUsed);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    
    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
    return `${Math.floor(diffDays / 30)} months ago`;
  };

  const getAgentTypeColor = (type: string): string => {
    switch (type) {
      case 'repo': return '#52b6ef';
      case 'helper': return '#af77ca';
      case 'confluence': return '#f09348';
      default: return '#a9b9ba';
    }
  };

  const getAgentTypeIcon = (type: string): string => {
    switch (type) {
      case 'repo': return '📁';
      case 'helper': return '🤖';
      case 'confluence': return '📚';
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
              <span>👤</span>
              <span>My Agents</span>
            </h2>
            <button
              onClick={onClose}
              className="text-slack-textMuted hover:text-slack-text transition-colors"
            >
              ✕
            </button>
          </div>
          
          {/* Search */}
          <div className="mb-3">
            <input
              type="text"
              placeholder="Search agents..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text placeholder-slack-textMuted focus:outline-none focus:border-slack-accent"
            />
          </div>
          
          {/* Filter */}
          <div className="flex gap-1">
            {(['all', 'repo', 'helper', 'confluence'] as AgentCategory[]).map((type) => (
              <button
                key={type}
                onClick={() => setFilterType(type)}
                className={`px-3 py-1 text-xs rounded transition-colors ${
                  filterType === type
                    ? 'bg-slack-accent text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                {type === 'all' ? 'All' : type.charAt(0).toUpperCase() + type.slice(1)}
              </button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-2">
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <div className="text-slack-textMuted">Loading my agents...</div>
            </div>
          ) : sortedAgents.length === 0 ? (
            searchTerm || filterType !== 'all' ? (
              <div className="text-slack-textMuted text-sm p-4 text-center">
                No agents match your filters
              </div>
            ) : (
              <div className="p-6 text-center">
                <div className="text-4xl mb-3">👤</div>
                <h3 className="text-lg font-medium text-slack-text mb-2">No Agents Yet</h3>
                <p className="text-sm text-slack-textMuted mb-4">
                  Your agents are created when you use specialized agents. Once created, they load instantly from cache.
                </p>
                <div className="bg-slack-bgHover rounded p-4 text-left text-sm">
                  <p className="font-medium text-slack-text mb-2">Create agents using:</p>
                  <code className="block bg-slack-bg p-2 rounded mb-1">/create-repo-agent &lt;path&gt; [name]</code>
                  <code className="block bg-slack-bg p-2 rounded mb-1">/create-helper &lt;template&gt;</code>
                  <code className="block bg-slack-bg p-2 rounded">/create-confluence-agent &lt;space-key&gt; [name]</code>
                </div>
              </div>
            )
          ) : (
            <div className="space-y-2">
              {sortedAgents.map((agent, index) => (
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
                      <div className="text-xs text-slack-textMuted truncate mt-1">
                        {agent.path}
                      </div>
                      
                      {/* Metadata */}
                      <div className="flex items-center gap-3 mt-2 text-xs text-slack-textMuted">
                        <span>{formatLastUsed(agent.last_used)}</span>
                        <span>•</span>
                        <span>{formatCacheSize(agent.cache_size)}</span>
                      </div>
                      
                      {/* Type Badge */}
                      <div className="mt-2">
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
                    </div>
                    
                    {/* Load Button */}
                    <button
                      onClick={() => handleLoadAgent(agent)}
                      disabled={loadingAgent === agent.name}
                      className="px-3 py-1 bg-slack-accent hover:bg-slack-accentHover text-white text-xs rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-shrink-0"
                    >
                      {loadingAgent === agent.name ? 'Loading...' : 'Load'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-3 border-t border-slack-border text-xs text-slack-textMuted text-center">
          {sortedAgents.length} agent{sortedAgents.length !== 1 ? 's' : ''}
        </div>
      </div>
    </div>
  );
}
