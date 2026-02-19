import { useEffect } from 'react';
import type { AgentInfo } from '../types/protocol';
import { getAgentColor } from '../types/protocol';

interface AgentInfoModalProps {
  agent: AgentInfo | undefined;
  isOpen: boolean;
  onClose: () => void;
  onProviderSwitch?: (agentId: string, provider: string, model: string) => void;
  onExport?: (agentName: string) => void;
  onRemove?: (agentId: string, agentName: string) => void;
  onApprovalModeChange?: (agentId: string, mode: 'interactive' | 'auto_edit' | 'yolo') => void;
  switchingProvider?: string | null;
  availableOllamaModels?: string[];
  availableLMStudioModels?: string[];
}

export function AgentInfoModal({ 
  agent, 
  isOpen, 
  onClose,
  onProviderSwitch,
  onExport,
  onRemove,
  onApprovalModeChange,
  switchingProvider,
  availableOllamaModels = [],
  availableLMStudioModels = []
}: AgentInfoModalProps) {
  // Handle escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      return () => document.removeEventListener('keydown', handleEscape);
    }
  }, [isOpen, onClose]);

  if (!isOpen || !agent) return null;

  const agentColor = agent.type === 'loading' ? '#3b82f6' : getAgentColor(agent.type);
  const isActive = agent.status === 'active';
  const isLoading = agent.status === 'loading';

  const getProviderIcon = (provider?: string) => {
    switch (provider) {
      case 'ollama':
        return '🤖';
      case 'claude':
        return '🧠';
      case 'lmstudio':
        return '🎨';
      default:
        return '❓';
    }
  };

  const getProviderColor = (provider?: string) => {
    switch (provider) {
      case 'ollama':
        return 'text-blue-500';
      case 'claude':
        return 'text-purple-500';
      case 'lmstudio':
        return 'text-green-500';
      default:
        return 'text-gray-500';
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black bg-opacity-50"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="relative bg-slack-bg border border-slack-border rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slack-border">
          <div className="flex items-center gap-3">
            <div
              className={`w-3 h-3 rounded-full ${
                isActive ? 'animate-pulse' : ''
              }`}
              style={{ backgroundColor: agentColor }}
            />
            <h2 className="text-xl font-bold text-slack-text">
              {agent.name}
            </h2>
            {isLoading && (
              <span className="text-sm px-2 py-1 rounded bg-blue-500/20 text-blue-500 font-medium animate-pulse">
                ⏳ Loading...
              </span>
            )}
            {agent.type === 'moderator' && !isLoading && (
              <span className="text-sm px-2 py-1 rounded bg-purple-500/20 text-purple-500 font-medium">
                🔒 System
              </span>
            )}
          </div>
          <button
            onClick={onClose}
            className="text-slack-textMuted hover:text-slack-text transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6 max-h-[calc(90vh-120px)] overflow-y-auto">
          {/* Basic Info */}
          <div className="space-y-4">
            <div>
              <h3 className="text-sm font-medium text-slack-textMuted mb-2">Agent Type</h3>
              <div className="flex items-center gap-2">
                <span
                  className="text-sm px-3 py-1 rounded"
                  style={{
                    backgroundColor: `${agentColor}20`,
                    color: agentColor,
                  }}
                >
                  {agent.type}
                </span>
              </div>
            </div>

            {/* Status */}
            <div>
              <h3 className="text-sm font-medium text-slack-textMuted mb-2">Status</h3>
              <div className="flex items-center gap-2">
                <div
                  className={`w-2 h-2 rounded-full ${
                    isActive ? 'animate-pulse' : ''
                  }`}
                  style={{ backgroundColor: agentColor }}
                />
                <span className="text-sm text-slack-text">
                  {isActive ? 'Active' : agent.status || 'Unknown'}
                </span>
                {agent.is_paused && (
                  <span className="text-xs px-2 py-1 rounded bg-yellow-500/20 text-yellow-500">
                    ⏸️ Paused
                  </span>
                )}
              </div>
            </div>

            {/* Tool Approval Mode -- shown for CLI agents */}
            {(agent.ai_provider === 'cursor-cli' || agent.ai_provider === 'gemini-cli' || agent.type === 'cli') && (
              <div>
                <h3 className="text-sm font-medium text-slack-textMuted mb-2">Tool Approval Mode</h3>
                <p className="text-xs text-slack-textMuted mb-2">
                  Controls whether this agent asks for your permission before using tools.
                </p>
                <select
                  value={agent.approval_mode || 'interactive'}
                  onChange={(e) => {
                    const mode = e.target.value as 'interactive' | 'auto_edit' | 'yolo';
                    onApprovalModeChange?.(agent.id, mode);
                  }}
                  className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text text-sm focus:outline-none focus:ring-1 focus:ring-slack-accent"
                >
                  <option value="interactive">
                    Interactive -- Ask before every tool call
                  </option>
                  <option value="auto_edit">
                    Auto Edit -- Approve file ops, ask for shell commands
                  </option>
                  <option value="yolo">
                    YOLO -- Auto-approve everything
                  </option>
                </select>
                <div className="mt-2 text-xs text-slack-textMuted">
                  {(agent.approval_mode || 'interactive') === 'interactive' && (
                    <span>You will see approve/reject buttons in the chat for each tool call.</span>
                  )}
                  {agent.approval_mode === 'auto_edit' && (
                    <span>File reads and edits run automatically. Shell commands need your approval.</span>
                  )}
                  {agent.approval_mode === 'yolo' && (
                    <span className="text-yellow-400">All tool calls will execute without confirmation.</span>
                  )}
                </div>
              </div>
            )}

            {/* AI Provider & Model -- hidden for agents that use external tools (CLI, etc.) */}
            {agent.ai_provider !== 'cursor-cli' && agent.type !== 'cli' && (
              <div>
                <h3 className="text-sm font-medium text-slack-textMuted mb-2">AI Configuration</h3>
                <div className="space-y-2">
                  <div className="flex items-center gap-2 mb-2">
                    <span className={`text-sm font-medium ${getProviderColor(agent.ai_provider)}`}>
                      {getProviderIcon(agent.ai_provider)} {agent.ai_provider || 'unknown'}
                    </span>
                    <span className="text-sm text-slack-textMuted">•</span>
                    <span className="text-sm text-slack-text">{agent.ai_model || 'unknown'}</span>
                  </div>
                  {onProviderSwitch && (
                    <div className="relative">
                      <select
                        value={`${agent.ai_provider || 'claude'}::${agent.ai_model || 'claude-sonnet'}`}
                        onChange={(e) => {
                          const [provider, ...modelParts] = e.target.value.split('::');
                          const model = modelParts.join('::');
                          onProviderSwitch(agent.id, provider, model);
                        }}
                        disabled={switchingProvider === agent.id}
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-1 focus:ring-slack-accent"
                        title="Switch AI provider"
                      >
                        <optgroup label="Claude">
                          <option value="claude::claude-sonnet">🧠 Claude Sonnet</option>
                          <option value="claude::claude-haiku">🧠 Claude Haiku</option>
                        </optgroup>
                        <optgroup label="Ollama">
                          {availableOllamaModels.length > 0 ? (
                            availableOllamaModels.map((m) => (
                              <option key={m} value={`ollama::${m}`}>
                                🤖 {m}
                              </option>
                            ))
                          ) : (
                            <option value="ollama::none" disabled>
                              🤖 No models available
                            </option>
                          )}
                        </optgroup>
                        <optgroup label="LM Studio">
                          {availableLMStudioModels.length > 0 ? (
                            availableLMStudioModels.map((m) => (
                              <option key={m} value={`lmstudio::${m}`}>
                                🎨 {m}
                              </option>
                            ))
                          ) : (
                            <option value="lmstudio::none" disabled>
                              🎨 No models available
                            </option>
                          )}
                        </optgroup>
                      </select>
                      {switchingProvider === agent.id && (
                        <div className="absolute top-2 right-2 w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
                      )}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Expertise */}
            {agent.expertise && agent.expertise.length > 0 && (
              <div>
                <h3 className="text-sm font-medium text-slack-textMuted mb-2">Expertise</h3>
                <div className="flex flex-wrap gap-2">
                  {agent.expertise.map((skill) => (
                    <span
                      key={skill}
                      className="text-sm px-3 py-1 rounded bg-slack-bgHover text-slack-textMuted"
                    >
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* Indexing Status for Repo Agents */}
            {agent.type === 'repo' && agent.indexing_status && (
              <div>
                <h3 className="text-sm font-medium text-slack-textMuted mb-2">Repository Status</h3>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className={`text-sm font-medium ${
                      agent.indexing_status === 'ready' ? 'text-green-500' :
                      agent.indexing_status === 'error' ? 'text-red-500' :
                      agent.indexing_status === 'indexing' ? 'text-blue-500' :
                      agent.indexing_status === 'reindexing' ? 'text-yellow-500' :
                      'text-slack-textMuted'
                    }`}>
                      {agent.indexing_status === 'indexing' && '📊 '}
                      {agent.indexing_status === 'reindexing' && '🔄 '}
                      {agent.indexing_status === 'ready' && '✅ '}
                      {agent.indexing_status === 'error' && '❌ '}
                      {agent.indexing_status}
                    </span>
                    {agent.index_progress !== undefined && agent.indexing_status !== 'ready' && (
                      <span className="text-sm font-bold text-slack-text">{agent.index_progress}%</span>
                    )}
                  </div>
                  {agent.index_progress !== undefined && agent.indexing_status !== 'ready' && (
                    <div className="w-full h-2 bg-slack-bgHover rounded-full overflow-hidden">
                      <div
                        className={`h-full transition-all duration-300 ${
                          agent.indexing_status === 'indexing' ? 'bg-blue-500' :
                          agent.indexing_status === 'reindexing' ? 'bg-yellow-500' :
                          agent.indexing_status === 'error' ? 'bg-red-500' :
                          'bg-green-500'
                        }`}
                        style={{ width: `${agent.index_progress}%` }}
                      />
                    </div>
                  )}
                </div>
              </div>
            )}

          </div>
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-slack-border bg-slack-bgHover">
          <div className="flex justify-between items-center gap-3">
            <div className="flex gap-2">
              {/* Export Button - only for repo and helper agents */}
              {(agent.type === 'repo' || agent.type === 'helper') && onExport && (
                <button
                  onClick={() => {
                    onExport(agent.name);
                  }}
                  className="px-4 py-2 text-sm text-blue-400 hover:text-blue-300 hover:bg-blue-500/10 rounded transition-colors border border-blue-500/30"
                  title={`Export ${agent.name} to MCP format`}
                >
                  📦 Export
                </button>
              )}
              
              {/* Remove Button - not available for moderator */}
              {onRemove && agent.type !== 'moderator' && (
                <button
                  onClick={() => {
                    onRemove(agent.id, agent.name);
                    onClose();
                  }}
                  className="px-4 py-2 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors border border-red-500/30"
                  title={`Remove ${agent.name} from conversation`}
                >
                  Remove
                </button>
              )}
            </div>
            <button
              onClick={onClose}
              className="px-4 py-2 bg-slack-accent hover:bg-slack-accentHover text-white rounded transition-colors"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
