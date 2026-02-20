import { useState, useEffect, useRef } from 'react';
import type { AgentInfo, AgentType } from '../types/protocol';
import { getAgentColor } from '../types/protocol';
import { useChatStore } from '../stores/chatStore';
import { useSettingsStore } from '../stores/settingsStore';
import { AgentInfoModal } from './AgentInfoModal';

interface AgentListProps {
  agents: AgentInfo[];
  onRefresh: () => void;
  onAgentClick?: (agentName: string) => void;
  onRemoveAgent?: (agentId: string, agentName: string) => void;
  onExportAgent?: (agentName: string) => void;
}

const MIN_WIDTH = 150; // Minimum usable width
const DEFAULT_WIDTH = 320;
const STORAGE_KEY = 'agent-list-width';

export function AgentList({ agents, onRefresh, onAgentClick, onRemoveAgent, onExportAgent }: AgentListProps) {
  const { switchAgentProvider, loadingAgents } = useChatStore();
  const { fetchOllamaModels, fetchLMStudioModels } = useSettingsStore();
  const [switchingProvider, setSwitchingProvider] = useState<string | null>(null);
  const [infoAgentId, setInfoAgentId] = useState<string | null>(null);
  const [availableOllamaModels, setAvailableOllamaModels] = useState<string[]>([]);
  const [availableLMStudioModels, setAvailableLMStudioModels] = useState<string[]>([]);

  // Resize state
  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    const savedWidth = saved ? parseInt(saved, 10) : DEFAULT_WIDTH;
    // Sanity check: ensure saved width is reasonable (not larger than 35% of screen)
    const maxReasonableWidth = Math.min(window.innerWidth * 0.35, 500);
    if (savedWidth > maxReasonableWidth || savedWidth < MIN_WIDTH || isNaN(savedWidth)) {
      localStorage.removeItem(STORAGE_KEY);
      return DEFAULT_WIDTH;
    }
    return savedWidth;
  });
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartX = useRef<number>(0);
  const resizeStartWidth = useRef<number>(0);
  const currentWidthRef = useRef<number>(width);
  
  // Keep ref in sync with state
  useEffect(() => {
    currentWidthRef.current = width;
  }, [width]);

  // Clamp width if window is resized smaller
  useEffect(() => {
    const handleWindowResize = () => {
      const maxAllowed = Math.min(window.innerWidth * 0.35, 500);
      if (currentWidthRef.current > maxAllowed) {
        const clamped = Math.max(MIN_WIDTH, maxAllowed);
        setWidth(clamped);
        localStorage.setItem(STORAGE_KEY, clamped.toString());
      }
    };
    window.addEventListener('resize', handleWindowResize);
    return () => window.removeEventListener('resize', handleWindowResize);
  }, []);

  // Fetch available Ollama and LM Studio models on component mount
  useEffect(() => {
    const loadOllamaModels = async () => {
      try {
        const models = await fetchOllamaModels();
        setAvailableOllamaModels(models);
      } catch (error) {
        console.error('Failed to fetch Ollama models:', error);
        setAvailableOllamaModels([]);
      }
    };
    const loadLMStudioModels = async () => {
      try {
        const models = await fetchLMStudioModels();
        setAvailableLMStudioModels(models);
      } catch (error) {
        console.error('Failed to fetch LM Studio models:', error);
        setAvailableLMStudioModels([]);
      }
    };
    loadOllamaModels();
    loadLMStudioModels();
  }, [fetchOllamaModels, fetchLMStudioModels]);

  // Resize handlers
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      
      const delta = resizeStartX.current - e.clientX; // Inverted for left edge resize
      const newWidth = resizeStartWidth.current + delta;
      // Allow free resizing, but limit to reasonable maximum
      // Sidebar should not take more than 35% of screen or 500px
      const maxWidth = Math.min(window.innerWidth * 0.35, 500);
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
  
  // Filter out removed agents from the main sidebar
  const activeAgents = agents.filter(agent => agent.status === 'active');
  
  // Create loading agents for display
  const loadingAgentsList: AgentInfo[] = Array.from(loadingAgents).map(agentName => ({
    id: `loading-${agentName}`,
    name: agentName,
    type: 'loading' as AgentType,
    status: 'loading',
    expertise: [] as string[],
    model: '',
    ai_provider: '',
    ai_model: '',
    is_paused: false,
    supports_vision: false,
    indexing_status: 'loading',
    index_progress: 0,
    repository_path: '',
    knowledge_path: '',
    confluence_space_key: '',
    last_active_time: new Date().toISOString(),
    removed_from: undefined
  }));
  
  // Combine active agents with loading agents
  const allAgents = [...activeAgents, ...loadingAgentsList];
  
  // Debug: Log the props
  console.log('AgentList props:', { onExportAgent: !!onExportAgent, agentsCount: agents.length });
  console.log('Active agents:', activeAgents.map(a => ({ name: a.name, status: a.status })));
  console.log('All agents:', allAgents.map(a => ({ name: a.name, status: a.status })));
  
  // Debug: Log raw agent data to check for missing names
  console.log('Raw agents data:', agents.map(a => ({ 
    id: a.id, 
    name: a.name, 
    type: a.type, 
    hasName: !!a.name,
    nameLength: a.name ? a.name.length : 0
  })));

  const handleProviderSwitch = async (agentId: string, provider: string, model: string) => {
    setSwitchingProvider(agentId);
    try {
      await switchAgentProvider(agentId, provider, model);
      onRefresh();
    } catch (error) {
      console.error('Failed to switch provider:', error);
    } finally {
      setSwitchingProvider(null);
    }
  };

  const handleApprovalModeChange = async (agentId: string, mode: 'interactive' | 'auto_edit' | 'yolo') => {
    try {
      const { ChatAPI } = await import('../api/chatAPI');
      const api = new ChatAPI();
      await api.setAgentApprovalMode(agentId, mode);
      onRefresh();
    } catch (error) {
      console.error('Failed to set approval mode:', error);
    }
  };

  
  return (
    <div className="flex h-full bg-slack-bg border-l border-slack-border relative flex-shrink-0" style={{ width: `${width}px`, minWidth: `${MIN_WIDTH}px` }}>
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
      
      <div className="flex flex-col h-full w-full">
      {/* Header */}
      <div className="p-4 border-b border-slack-border">
        <h2 className="text-lg font-bold text-slack-text mb-3 flex items-center gap-2">
          <span>🤖</span>
          <span>Active Agents</span>
        </h2>
        <button
          onClick={onRefresh}
          className="w-full px-3 py-2 bg-slack-accent hover:bg-slack-accentHover text-white rounded transition-colors text-sm font-medium"
        >
          Refresh
        </button>
      </div>

      {/* Agent List */}
      <div className="flex-1 overflow-y-auto p-2" style={{ scrollbarWidth: 'thin' }}>
        {allAgents.length === 0 ? (
          <div className="text-slack-textMuted text-sm p-4 text-center">
            No agents connected
          </div>
        ) : (
          <div className="space-y-1">
            {allAgents.map((agent) => {
              const agentColor = agent.type === 'loading' ? '#3b82f6' : getAgentColor(agent.type);
              const isActive = agent.status === 'active';
              const isLoading = agent.status === 'loading';
              
              return (
                <div
                  key={agent.id}
                  className={`p-3 rounded transition-colors group ${
                    isLoading 
                      ? 'bg-blue-50 border border-blue-200 cursor-wait' 
                      : 'hover:bg-slack-bgHover cursor-pointer'
                  }`}
                  onClick={isLoading ? undefined : () => onAgentClick?.(agent.name)}
                  title={isLoading ? `${agent.name} is loading...` : `Click to mention ${agent.name}`}
                >
                  <div className="flex items-center gap-2">
                    {/* Status Indicator */}
                    <div
                      className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${
                        isActive ? 'animate-pulse' : isLoading ? 'animate-spin' : ''
                      }`}
                      style={{ backgroundColor: isLoading ? '#3b82f6' : agentColor }}
                    />
                    
                    {/* Agent Name */}
                    <div className="flex-1 min-w-0">
                      <span className="font-medium text-slack-text text-sm truncate block" title={agent.name || 'Unknown Agent'}>
                        {agent.name || agent.id || 'Unknown Agent'}
                      </span>
                    </div>
                    
                    {/* Info Icon - Always Visible */}
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        setInfoAgentId(agent.id);
                        fetchOllamaModels().then(setAvailableOllamaModels).catch(() => {});
                        fetchLMStudioModels().then(setAvailableLMStudioModels).catch(() => {});
                      }}
                      className="text-slack-textMuted hover:text-slack-text transition-colors flex-shrink-0"
                      title={`View ${agent.name} details`}
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="p-3 border-t border-slack-border text-xs text-slack-textMuted text-center">
        {activeAgents.length} agent{activeAgents.length !== 1 ? 's' : ''} online
      </div>

      {/* Agent Info Modal */}
      {infoAgentId && (
        <AgentInfoModal
          agent={agents.find(a => a.id === infoAgentId) || allAgents.find(a => a.id === infoAgentId)}
          isOpen={!!infoAgentId}
          onClose={() => setInfoAgentId(null)}
          onProviderSwitch={handleProviderSwitch}
          onExport={onExportAgent}
          onRemove={onRemoveAgent}
          onApprovalModeChange={handleApprovalModeChange}
          switchingProvider={switchingProvider}
          availableOllamaModels={availableOllamaModels}
          availableLMStudioModels={availableLMStudioModels}
        />
      )}
      </div>
    </div>
  );
}

