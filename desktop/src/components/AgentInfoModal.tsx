import { useEffect, useState } from 'react';
import type { AgentInfo, AgentToolCapabilities } from '../types/protocol';
import { getAgentColor } from '../types/protocol';
import { useChatStore } from '../stores/chatStore';
import { usePacksStore } from '../stores/packsStore';
import { PACK_CAP } from '../stores/packCapabilities';
import { ChatAPI, type UserLearning } from '../api/chatAPI';
import { LearningProposalModal } from './LearningProposalModal';

const TOOL_EXAMPLE_PROMPTS: Record<string, string> = {
  analyze_sequence: 'Analyze this peptide: MKTAYIAKQRQISFVK',
  fold_protein: 'Fold MKTAY and tell me where the PDB was written.',
  generate_image: 'Generate an image of a cell diagram for a lab notebook.',
  analyze_go_code: 'Run static analysis on ./cmd/server',
  run_go_tests: 'Run tests for ./internal/agent/...',
};

interface AgentInfoModalProps {
  agent: AgentInfo | undefined;
  isOpen: boolean;
  onClose: () => void;
  onProviderSwitch?: (agentId: string, provider: string, model: string) => void;
  onExport?: (agentName: string) => void;
  onRemove?: (agentId: string, agentName: string) => void;
  onDelete?: (agentId: string, agentName: string) => void;
  deletingAgent?: boolean;
  onApprovalModeChange?: (agentId: string, mode: 'interactive' | 'auto_edit' | 'yolo') => void;
  /** Called after agent custom rules are saved successfully (refresh agent list). */
  onAfterRulesSaved?: () => void;
  onTrainLoRA?: (agentId: string) => void;
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
  onDelete,
  deletingAgent = false,
  onApprovalModeChange,
  onAfterRulesSaved,
  onTrainLoRA,
  switchingProvider,
  availableOllamaModels = [],
  availableLMStudioModels = []
}: AgentInfoModalProps) {
  const serverAddr = useChatStore(s => s.serverAddr);
  const hasLoRATraining = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_TRAINING));
  const hasPersonalLearning = usePacksStore((s) => s.hasCapability(PACK_CAP.PERSONAL_LEARNING));
  const [rulesDraft, setRulesDraft] = useState('');
  const [savingRules, setSavingRules] = useState(false);
  const [rulesError, setRulesError] = useState<string | null>(null);
  const [toolCaps, setToolCaps] = useState<AgentToolCapabilities | null>(null);
  const [toolsLoading, setToolsLoading] = useState(false);
  const [toolsError, setToolsError] = useState<string | null>(null);
  const [fetchedOllamaModels, setFetchedOllamaModels] = useState<string[]>([]);
  const [fetchedLMStudioModels, setFetchedLMStudioModels] = useState<string[]>([]);
  const [learnings, setLearnings] = useState<UserLearning[]>([]);
  const [learningsLoading, setLearningsLoading] = useState(false);
  const [learningsError, setLearningsError] = useState<string | null>(null);
  const [addLearningOpen, setAddLearningOpen] = useState(false);

  const isExpertAgent =
    !!agent &&
    agent.type !== 'loading' &&
    agent.type !== 'cli' &&
    agent.type !== 'moderator' &&
    agent.type !== 'human';

  useEffect(() => {
    if (agent && agent.type !== 'loading') {
      setRulesDraft(agent.custom_rules_markdown ?? '');
      setRulesError(null);
    }
  }, [agent]);

  const isCLIAgent =
    agent?.type === 'cli' ||
    (typeof agent?.ai_provider === 'string' && agent.ai_provider.endsWith('-cli'));

  useEffect(() => {
    if (!isOpen || !agent || agent.type === 'loading' || isCLIAgent) {
      setToolCaps(null);
      setToolsError(null);
      return;
    }
    let cancelled = false;
    setToolsLoading(true);
    setToolsError(null);
    const api = new ChatAPI(serverAddr);
    api
      .fetchAgentTools(agent.id)
      .then((cap) => {
        if (!cancelled) setToolCaps(cap);
      })
      .catch((e) => {
        if (!cancelled) {
          setToolCaps(null);
          setToolsError(e instanceof Error ? e.message : 'Failed to load tools');
        }
      })
      .finally(() => {
        if (!cancelled) setToolsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [isOpen, agent?.id, agent?.type, isCLIAgent, serverAddr]);

  useEffect(() => {
    if (!isOpen || !agent || !hasPersonalLearning || !isExpertAgent) {
      setLearnings([]);
      return;
    }
    let cancelled = false;
    setLearningsLoading(true);
    setLearningsError(null);
    const api = new ChatAPI(serverAddr);
    void api
      .fetchLearnings(agent.id)
      .then((rows) => {
        if (!cancelled) setLearnings(rows);
      })
      .catch((e) => {
        if (!cancelled) {
          setLearnings([]);
          setLearningsError(e instanceof Error ? e.message : 'Failed to load learnings');
        }
      })
      .finally(() => {
        if (!cancelled) setLearningsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [isOpen, agent?.id, hasPersonalLearning, isExpertAgent, serverAddr]);

  useEffect(() => {
    if (!isOpen || !agent || agent.type === 'loading' || isCLIAgent) {
      return;
    }
    let cancelled = false;
    const api = new ChatAPI(serverAddr);
    void api.fetchOllamaModels().then((models) => {
      if (!cancelled) {
        setFetchedOllamaModels(models);
      }
    }).catch(() => {});
    void api.fetchLMStudioModels().then((models) => {
      if (!cancelled) {
        setFetchedLMStudioModels(models);
      }
    }).catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [isOpen, agent?.id, agent?.type, isCLIAgent, serverAddr]);

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

  const effectiveProvider =
    agent.ai_provider ||
    toolCaps?.chat_provider ||
    (agent.model && !agent.model.startsWith('claude') ? 'ollama' : '') ||
    'unknown';
  const effectiveModel = agent.ai_model || agent.model || toolCaps?.chat_model || '';
  const selectValue =
    effectiveProvider && effectiveModel && effectiveProvider !== 'unknown'
      ? `${effectiveProvider}::${effectiveModel}`
      : '';

  const mergedOllamaModels = [
    ...new Set([...availableOllamaModels, ...fetchedOllamaModels]),
  ];
  const mergedLMStudioModels = [
    ...new Set([...availableLMStudioModels, ...fetchedLMStudioModels]),
  ];

  const ollamaOptions = [...mergedOllamaModels];
  if (
    effectiveProvider === 'ollama' &&
    effectiveModel &&
    !ollamaOptions.includes(effectiveModel)
  ) {
    ollamaOptions.unshift(effectiveModel);
  }
  const lmStudioOptions = [...mergedLMStudioModels];
  if (
    effectiveProvider === 'lmstudio' &&
    effectiveModel &&
    !lmStudioOptions.includes(effectiveModel)
  ) {
    lmStudioOptions.unshift(effectiveModel);
  }

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
      <div className="relative flex flex-col bg-slack-bg border border-slack-border rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex shrink-0 items-center justify-between p-6 border-b border-slack-border">
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
        <div className="min-h-0 flex-1 overflow-y-auto p-6 space-y-6">
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
                    <span className={`text-sm font-medium ${getProviderColor(effectiveProvider)}`}>
                      {getProviderIcon(effectiveProvider)} {effectiveProvider}
                    </span>
                    <span className="text-sm text-slack-textMuted">•</span>
                    <span className="text-sm text-slack-text">{effectiveModel || 'unknown'}</span>
                  </div>
                  {onProviderSwitch && (
                    <div className="relative">
                      <select
                        value={selectValue}
                        onChange={(e) => {
                          const [provider, ...modelParts] = e.target.value.split('::');
                          const model = modelParts.join('::');
                          onProviderSwitch(agent.id, provider, model);
                        }}
                        disabled={switchingProvider === agent.id}
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-1 focus:ring-slack-accent"
                        title="Switch AI provider"
                      >
                        {selectValue &&
                          !selectValue.startsWith('claude::') &&
                          !ollamaOptions.some((m) => selectValue === `ollama::${m}`) &&
                          !lmStudioOptions.some((m) => selectValue === `lmstudio::${m}`) && (
                            <optgroup label="Current">
                              <option value={selectValue}>
                                {effectiveProvider === 'ollama' ? '🤖' : effectiveProvider === 'lmstudio' ? '🎨' : '🧠'}{' '}
                                {effectiveModel} (current)
                              </option>
                            </optgroup>
                          )}
                        <optgroup label="Claude">
                          <option value="claude::claude-sonnet">🧠 Claude Sonnet</option>
                          <option value="claude::claude-haiku">🧠 Claude Haiku</option>
                        </optgroup>
                        <optgroup label="Ollama">
                          {ollamaOptions.length > 0 ? (
                            ollamaOptions.map((m) => (
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
                          {lmStudioOptions.length > 0 ? (
                            lmStudioOptions.map((m) => (
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

            {/* Tools & models */}
            <div>
              <h3 className="text-sm font-medium text-slack-textMuted mb-2">Tools &amp; models</h3>
              {isCLIAgent ? (
                <p className="text-sm text-slack-textMuted">
                  CLI agents use their host toolset (files, shell, etc.), not hub MCP tools. Use Tool
                  Approval Mode above to control confirmations.
                </p>
              ) : toolsLoading ? (
                <p className="text-sm text-slack-textMuted">Loading tools…</p>
              ) : toolsError ? (
                <p className="text-sm text-red-400">{toolsError}</p>
              ) : toolCaps ? (
                <div className="space-y-3 text-sm">
                  <div className="rounded border border-slack-border bg-slack-bgHover p-3 space-y-1">
                    <div>
                      <span className="text-slack-textMuted">Chat model: </span>
                      <span className="text-slack-text font-mono text-xs">
                        {toolCaps.chat_provider} / {toolCaps.chat_model}
                      </span>
                      {toolCaps.chat_native_tools ? (
                        <span className="ml-2 text-xs text-green-500">native tools</span>
                      ) : (
                        <span className="ml-2 text-xs text-slack-textMuted">no native tools</span>
                      )}
                    </div>
                    {toolCaps.tool_loop_model && (
                      <div>
                        <span className="text-slack-textMuted">Tool loop: </span>
                        <span className="text-slack-text font-mono text-xs">{toolCaps.tool_loop_model}</span>
                        {toolCaps.tool_loop_uses_fallback && (
                          <span className="ml-2 text-xs text-amber-500">fallback</span>
                        )}
                      </div>
                    )}
                    {toolCaps.mcp_enabled && toolCaps.mcp_port ? (
                      <div className="text-xs text-slack-textMuted">
                        MCP server: localhost:{toolCaps.mcp_port}
                      </div>
                    ) : null}
                  </div>
                  {(toolCaps.tools ?? []).length === 0 ? (
                    <p className="text-slack-textMuted">
                      No hub tools registered. Enable a domain pack and MCP in Settings, or run{' '}
                      <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">/tools-list</code>{' '}
                      in a channel with this agent.
                    </p>
                  ) : (
                    <ul className="space-y-2">
                      {(toolCaps.tools ?? []).map((tool) => (
                        <li
                          key={tool.name}
                          className="rounded border border-slack-border bg-slack-bgHover p-2"
                        >
                          <div className="font-mono text-xs text-slack-accent">
                            {tool.name}
                            {tool.parameters && tool.parameters.length > 0 && (
                              <span className="text-slack-textMuted">
                                ({tool.parameters.map((p) => p.name).join(', ')})
                              </span>
                            )}
                          </div>
                          <p className="text-xs text-slack-textMuted mt-1">{tool.description}</p>
                          {TOOL_EXAMPLE_PROMPTS[tool.name] && (
                            <p className="text-xs text-slack-text mt-1 italic">
                              Try: &ldquo;{TOOL_EXAMPLE_PROMPTS[tool.name]}&rdquo;
                            </p>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                  {toolCaps.notes && toolCaps.notes.length > 0 && (
                    <ul className="text-xs text-slack-textMuted list-disc pl-4 space-y-1">
                      {toolCaps.notes.map((note) => (
                        <li key={note}>{note}</li>
                      ))}
                    </ul>
                  )}
                  <p className="text-xs text-slack-textMuted">
                    In chat, use <code className="font-mono bg-slack-bgHover px-1 rounded">/tools-list</code>{' '}
                    to see tools for all agents in the current channel.
                  </p>
                </div>
              ) : (
                <p className="text-sm text-slack-textMuted">No tool information available.</p>
              )}
            </div>

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

            {!isLoading && (
              <div>
                <h3 className="text-sm font-medium text-slack-textMuted mb-2">Agent rules (markdown)</h3>
                <p className="text-xs text-slack-textMuted mb-2">
                  Instructions scoped to this agent only. Stored on the hub server.
                </p>
                <textarea
                  value={rulesDraft}
                  onChange={(e) => setRulesDraft(e.target.value)}
                  rows={6}
                  className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text font-mono focus:outline-none focus:ring-1 focus:ring-slack-accent"
                  placeholder="Example: Always cite file paths from this workspace."
                />
                {rulesError && <p className="text-xs text-red-400 mt-1">{rulesError}</p>}
                <button
                  type="button"
                  onClick={async () => {
                    setRulesError(null);
                    setSavingRules(true);
                    try {
                      const api = new ChatAPI(serverAddr);
                      await api.setAgentCustomRulesMarkdown(agent.id, rulesDraft);
                      onAfterRulesSaved?.();
                    } catch (e) {
                      setRulesError(e instanceof Error ? e.message : 'Save failed');
                    } finally {
                      setSavingRules(false);
                    }
                  }}
                  disabled={savingRules}
                  className="mt-2 px-3 py-1.5 text-sm bg-slack-accent text-white rounded hover:opacity-90 disabled:opacity-50"
                >
                  {savingRules ? 'Saving…' : 'Save agent rules'}
                </button>
              </div>
            )}

            {hasPersonalLearning && isExpertAgent && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-sm font-medium text-slack-textMuted">
                    Learnings ({learnings.length})
                  </h3>
                  <button
                    type="button"
                    onClick={() => setAddLearningOpen(true)}
                    className="text-xs px-2 py-1 rounded border border-slack-border text-slack-text hover:bg-slack-bgHover"
                  >
                    Add learning
                  </button>
                </div>
                {learningsLoading && (
                  <p className="text-xs text-slack-textMuted">Loading…</p>
                )}
                {learningsError && (
                  <p className="text-xs text-red-400">{learningsError}</p>
                )}
                {!learningsLoading && learnings.length === 0 && (
                  <p className="text-xs text-slack-textMuted">No saved learnings for this expert yet.</p>
                )}
                {learnings.length > 0 && (
                  <ul className="space-y-2 max-h-40 overflow-y-auto">
                    {learnings.slice(0, 8).map((e) => (
                      <li
                        key={e.id}
                        className="text-xs p-2 rounded bg-slack-bgHover border border-slack-border flex justify-between gap-2"
                      >
                        <span>
                          <span className="text-slack-textMuted">[{e.category}]</span> {e.content}
                        </span>
                        <button
                          type="button"
                          className="text-red-400 hover:text-red-300 shrink-0"
                          onClick={async () => {
                            try {
                              const api = new ChatAPI(serverAddr);
                              await api.deleteLearning(e.id);
                              setLearnings((prev) => prev.filter((x) => x.id !== e.id));
                            } catch (err) {
                              setLearningsError(err instanceof Error ? err.message : 'Forget failed');
                            }
                          }}
                        >
                          Forget
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}

          </div>
        </div>

        {/* Footer */}
        <div className="shrink-0 p-6 border-t border-slack-border bg-slack-bgHover">
          <div className="flex justify-between items-center gap-3">
            <div className="flex gap-2">
              {hasLoRATraining && onTrainLoRA && isExpertAgent && (
                <button
                  type="button"
                  onClick={() => {
                    onTrainLoRA(agent.id);
                    onClose();
                  }}
                  className="px-4 py-2 text-sm text-purple-300 hover:text-purple-200 hover:bg-purple-500/10 rounded transition-colors border border-purple-500/30"
                  title={`Train a LoRA adapter from ${agent.name} sessions`}
                >
                  🎯 Train LoRA
                </button>
              )}
              {/* Export Button - repo agents only */}
              {agent.type === 'repo' && onExport && (
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
              
              {/* Remove — hide from channel only (can recall later) */}
              {onRemove && agent.type !== 'moderator' && agent.type !== 'human' && (
                <button
                  type="button"
                  disabled={deletingAgent}
                  onClick={() => {
                    onRemove(agent.id, agent.name);
                    onClose();
                  }}
                  className="px-4 py-2 text-sm text-amber-400 hover:text-amber-300 hover:bg-amber-500/10 rounded transition-colors border border-amber-500/30 disabled:opacity-50"
                  title={`Remove ${agent.name} from this channel (can recall later)`}
                >
                  Remove from channel
                </button>
              )}

              {/* Delete — permanent */}
              {onDelete && agent.type !== 'moderator' && agent.type !== 'human' && (
                <button
                  type="button"
                  disabled={deletingAgent}
                  onClick={() => {
                    onDelete(agent.id, agent.name);
                  }}
                  className="px-4 py-2 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors border border-red-500/30 disabled:opacity-50"
                  title={`Permanently delete ${agent.name}`}
                >
                  {deletingAgent ? 'Deleting…' : 'Delete permanently'}
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

      {hasPersonalLearning && agent && (
        <LearningProposalModal
          isOpen={addLearningOpen}
          proposal={
            addLearningOpen
              ? {
                  type: 'learning_proposal',
                  agent_id: agent.id,
                  agent_name: agent.name,
                  agent_type: agent.type,
                  draft: '',
                  category: 'preference',
                }
              : null
          }
          serverAddr={serverAddr}
          onClose={() => setAddLearningOpen(false)}
          onSaved={async () => {
            const api = new ChatAPI(serverAddr);
            const rows = await api.fetchLearnings(agent.id);
            setLearnings(rows);
          }}
        />
      )}
    </div>
  );
}
