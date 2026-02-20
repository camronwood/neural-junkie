import { useState, useEffect, useRef } from 'react';
import { useChatStore } from '../stores/chatStore';
import { ChatAPI } from '../api/chatAPI';
import type { CachedAgentInfo, AgentCategory, AgentInfo } from '../types/protocol';

interface MyAgentsPanelProps {
  onClose: () => void;
}

type TabType = 'my-agents' | 'removed';

const MIN_WIDTH = 250; // Minimum usable width
const DEFAULT_WIDTH = 384; // w-96 = 384px
const STORAGE_KEY = 'my-agents-panel-width';

export function MyAgentsPanel({ onClose }: MyAgentsPanelProps) {
  const { 
    serverAddr, 
    channel, 
    username, 
    myAgents, 
    setMyAgents,
    removedAgents,
    setRemovedAgents
  } = useChatStore();
  
  const [activeTab, setActiveTab] = useState<TabType>('my-agents');
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<AgentCategory>('all');
  const [loading, setLoading] = useState(true);
  const [loadingAgent, setLoadingAgent] = useState<string | null>(null);
  const [recallingAgent, setRecallingAgent] = useState<string | null>(null);

  // Resize state
  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    const savedWidth = saved ? parseInt(saved, 10) : DEFAULT_WIDTH;
    // Sanity check: ensure saved width is reasonable (not larger than screen)
    const maxReasonableWidth = window.innerWidth * 0.6; // Max 60% of screen
    return savedWidth > maxReasonableWidth ? DEFAULT_WIDTH : savedWidth;
  });
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartX = useRef<number>(0);
  const resizeStartWidth = useRef<number>(0);
  const currentWidthRef = useRef<number>(width);
  
  // Keep ref in sync with state
  useEffect(() => {
    currentWidthRef.current = width;
  }, [width]);

  const [api] = useState(() => new ChatAPI(serverAddr));

  // Load agents on mount
  useEffect(() => {
    const loadAgents = async () => {
      try {
        setLoading(true);
        const [myAgentsData, removedAgentsData] = await Promise.all([
          api.fetchMyAgents(),
          api.fetchRemovedAgents()
        ]);
        setMyAgents(myAgentsData);
        setRemovedAgents(removedAgentsData);
      } catch (error) {
        console.error('Failed to load agents:', error);
      } finally {
        setLoading(false);
      }
    };

    loadAgents();
  }, [api, setMyAgents, setRemovedAgents]);

  // Resize handlers
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      
      const delta = resizeStartX.current - e.clientX; // Inverted for left edge resize
      const newWidth = resizeStartWidth.current + delta;
      // Allow free resizing, but limit to reasonable maximum
      // My Agents panel should not take more than 50% of screen
      const maxWidth = Math.min(window.innerWidth * 0.5, 800); // Max 50% of screen or 800px
      const clampedWidth = Math.max(MIN_WIDTH, Math.min(maxWidth, newWidth));
      
      setWidth(clampedWidth);
    };

    const handleMouseUp = () => {
      if (isResizing) {
        setIsResizing(false);
        localStorage.setItem(STORAGE_KEY, currentWidthRef.current.toString());
      }
    };

    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing]);

  const handleResizeStart = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsResizing(true);
    resizeStartX.current = e.clientX;
    resizeStartWidth.current = currentWidthRef.current;
  };

  // Filter and search agents based on active tab
  const filteredAgents = (activeTab === 'my-agents' ? myAgents : removedAgents).filter(agent => {
    // Type filter
    if (filterType !== 'all' && agent.type !== filterType) {
      return false;
    }

    // Search filter
    if (searchTerm) {
      const searchLower = searchTerm.toLowerCase();
      const hasPath = 'path' in agent && agent.path;
      return (
        agent.name.toLowerCase().includes(searchLower) ||
        (hasPath && agent.path.toLowerCase().includes(searchLower))
      );
    }

    return true;
  });

  // Sort agents based on tab
  const sortedAgents = [...filteredAgents].sort((a, b) => {
    if (activeTab === 'my-agents') {
      // Sort by last used (descending), then by cache size (descending)
      const aLastUsed = 'last_used' in a ? a.last_used : '';
      const bLastUsed = 'last_used' in b ? b.last_used : '';
      const dateA = new Date(aLastUsed).getTime();
      const dateB = new Date(bLastUsed).getTime();
      
      if (dateA !== dateB) {
        return dateB - dateA; // Most recent first
      }
      
      const aCacheSize = 'cache_size' in a ? a.cache_size : 0;
      const bCacheSize = 'cache_size' in b ? b.cache_size : 0;
      return bCacheSize - aCacheSize; // Larger cache first
    } else {
      // For removed agents, sort by last active time
      const aLastActive = 'last_active_time' in a ? a.last_active_time : '';
      const bLastActive = 'last_active_time' in b ? b.last_active_time : '';
      const dateA = new Date(aLastActive || 0).getTime();
      const dateB = new Date(bLastActive || 0).getTime();
      return dateB - dateA; // Most recent first
    }
  });

  const handleLoadAgent = async (agent: CachedAgentInfo) => {
    try {
      setLoadingAgent(agent.name);
      // Add to global loading state
      const { addLoadingAgent } = useChatStore.getState();
      addLoadingAgent(agent.name);
      
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
        case 'cli':
          const cliType = agent.metadata?.cli_type || 'cursor';
          const workDir = agent.path || '';
          command = `/create-cli-agent ${cliType} ${agent.name}${workDir ? ' ' + workDir : ''}`;
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
      // Remove from global loading state
      const { removeLoadingAgent } = useChatStore.getState();
      removeLoadingAgent(agent.name);
    }
  };

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
      case 'moderator': return '#3b82f6';
      case 'assistant': return '#10b981';
      case 'rust': return '#dea584';
      case 'frontend': return '#52b6ef';
      case 'backend': return '#af77ca';
      case 'devops': return '#f09348';
      case 'database': return '#fbd837';
      case 'security': return '#f16a5a';
      default: return '#a9b9ba';
    }
  };

  const getAgentTypeIcon = (type: string): string => {
    switch (type) {
      case 'repo': return '📁';
      case 'helper': return '🤖';
      case 'confluence': return '📚';
      case 'moderator': return '🛡️';
      case 'assistant': return '✨';
      case 'rust': return '🦀';
      case 'frontend': return '🎨';
      case 'backend': return '⚙️';
      case 'devops': return '🔧';
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
      <div className="bg-slack-bg border-l border-slack-border flex flex-col animate-slide-in-right relative flex-shrink-0" style={{ width: `${width}px`, minWidth: `${MIN_WIDTH}px` }}>
        {/* Resize handle */}
        <div
          onMouseDown={handleResizeStart}
          className="absolute left-0 top-0 bottom-0 cursor-col-resize z-[100] group"
          style={{ 
            width: '6px', 
            marginLeft: '-3px',
            pointerEvents: 'auto',
          }}
        >
          <div className="absolute inset-0 bg-transparent group-hover:bg-slack-accent/30 transition-colors" />
          <div className="absolute left-1/2 top-1/2 -translate-y-1/2 -translate-x-1/2 w-1 h-8 bg-gray-400 group-hover:bg-slack-accent rounded-full opacity-0 group-hover:opacity-100 transition-opacity" />
        </div>
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
          
          {/* Tabs */}
          <div className="flex gap-1 mb-3">
            <button
              onClick={() => setActiveTab('my-agents')}
              className={`px-3 py-1 text-sm rounded transition-colors ${
                activeTab === 'my-agents'
                  ? 'bg-slack-accent text-white'
                  : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
              }`}
            >
              My Agents ({myAgents.length})
            </button>
            <button
              onClick={() => setActiveTab('removed')}
              className={`px-3 py-1 text-sm rounded transition-colors ${
                activeTab === 'removed'
                  ? 'bg-slack-accent text-white'
                  : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
              }`}
            >
              Removed ({removedAgents.length})
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
              <div className="text-slack-textMuted">Loading agents...</div>
            </div>
          ) : sortedAgents.length === 0 ? (
            searchTerm || filterType !== 'all' ? (
              <div className="text-slack-textMuted text-sm p-4 text-center">
                No agents match your filters
              </div>
            ) : activeTab === 'my-agents' ? (
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
            ) : (
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
                      {'path' in agent && agent.path && (
                        <div className="text-xs text-slack-textMuted truncate mt-1">
                          {agent.path}
                        </div>
                      )}
                      
                      {/* Metadata */}
                      <div className="flex items-center gap-3 mt-2 text-xs text-slack-textMuted">
                        {activeTab === 'my-agents' ? (
                          <>
                            {'last_used' in agent && (
                              <span>{formatLastUsed(agent.last_used)}</span>
                            )}
                            {'cache_size' in agent && (
                              <>
                                <span>•</span>
                                <span>{formatCacheSize(agent.cache_size)}</span>
                              </>
                            )}
                          </>
                        ) : (
                          <>
                            {'last_active_time' in agent && agent.last_active_time && (
                              <span>Last active: {formatLastActive(agent.last_active_time)}</span>
                            )}
                            {'removed_from' in agent && agent.removed_from && agent.removed_from.length > 0 && (
                              <>
                                <span>•</span>
                                <span>Removed from: {agent.removed_from.join(', ')}</span>
                              </>
                            )}
                          </>
                        )}
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
                    
                    {/* Action Button */}
                    {activeTab === 'my-agents' ? (
                      <button
                        onClick={() => handleLoadAgent(agent as CachedAgentInfo)}
                        disabled={loadingAgent === agent.name}
                        className="px-3 py-1 bg-slack-accent hover:bg-slack-accentHover text-white text-xs rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-shrink-0"
                      >
                        {loadingAgent === agent.name ? 'Loading...' : 'Load'}
                      </button>
                    ) : (
                      <button
                        onClick={() => handleRecallAgent(agent as AgentInfo)}
                        disabled={recallingAgent === ('id' in agent ? agent.id : '')}
                        className="px-3 py-1 bg-slack-accent hover:bg-slack-accentHover text-white text-xs rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-shrink-0"
                      >
                        {recallingAgent === ('id' in agent ? agent.id : '') ? 'Recalling...' : 'Recall'}
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-3 border-t border-slack-border text-xs text-slack-textMuted text-center">
          {activeTab === 'my-agents' 
            ? `${sortedAgents.length} of ${myAgents.length} agents`
            : `${sortedAgents.length} of ${removedAgents.length} removed agents`
          }
        </div>
      </div>
    </div>
  );
}
