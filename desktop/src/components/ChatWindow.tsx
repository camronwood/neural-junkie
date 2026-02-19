import { useState, useRef, useCallback, useEffect } from 'react';
import { useChatStore } from '../stores/chatStore';
import { useTerminalStore } from '../stores/terminalStore';
import { useSettingsStore } from '../stores/settingsStore';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { ChatAPI } from '../api/chatAPI';
import { clearCredentials } from '../utils/secureStorage';
import { buildFileTreeString } from '../utils/workspaceContext';
import type { WorkspaceContext } from '../utils/workspaceContext';
import { useWebSocket } from '../hooks/useWebSocket';
import { MessageList } from './MessageList';
import { AgentList } from './AgentList';
import { RichTextInput } from './RichTextInput';
import { TypingIndicator } from './TypingIndicator';
import { ThreadPanel } from './ThreadPanel';
import { MyAgentsPanel } from './MyAgentsPanel';
import { PendingChangesPanel } from './PendingChangesPanel';
import { TerminalPanel } from './TerminalPanel';
import { FileExplorerPanel } from './FileExplorerPanel';
import { CodeEditorPanel } from './CodeEditorPanel';
import { ToastContainer } from './Toast';
import { ErrorBoundary } from './ErrorBoundary';
import { CommandPalette } from './CommandPalette';
import { ChannelSidebar } from './ChannelSidebar';
import { CreateChannelModal } from './CreateChannelModal';
import { PendingChangesIcon, MyAgentsIcon, FilesIcon, EditorIcon, TerminalIcon, SettingsIcon, LogoutIcon } from './Icons';
import type { Message, ThinkingStatusMetadata, CommandDefinition } from '../types/protocol';

interface ChatWindowProps {
  onOpenSettings?: () => void;
  onLogout?: () => void;
  testMode?: boolean;
  setTestMode?: (value: boolean) => void;
}

export function ChatWindow({ onOpenSettings, onLogout, testMode: propTestMode, setTestMode: propSetTestMode }: ChatWindowProps = {}) {
  const {
    serverAddr,
    channel,
    username,
    messages,
    agents,
    channels,
    thinkingAgents,
    openThreadId,
    threadMetadata,
    addMessage,
    setMessages,
    setAgents,
    setChannels,
    switchChannel,
    setIsTyping,
    setConnectionStatus,
    addThinkingAgent,
    removeThinkingAgent,
    updateAgentStatus,
    openThread,
    closeThread,
    updateThreadMetadata,
    myAgentsPanelOpen,
    setMyAgentsPanelOpen,
    logout,
  } = useChatStore();

  const { isPanelOpen, panelHeight, addSuggestedCommand, setPanelOpen } = useTerminalStore();
  const { layoutSettings, loadLayoutSettings } = useSettingsStore();

  // State for tracking counts
  const [totalAgentsCount, setTotalAgentsCount] = useState(0);

  // State for file explorer and code editor panels
  const [fileExplorerOpen, setFileExplorerOpen] = useState(false);
  const [codeEditorOpen, setCodeEditorOpen] = useState(false);
  
  // State for pending changes panel
  const [pendingChangesOpen, setPendingChangesOpen] = useState(false);

  // State for create channel modal
  const [createChannelOpen, setCreateChannelOpen] = useState(false);

  // State for command palette
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [commandPaletteFilter, setCommandPaletteFilter] = useState('');
  const [commandDefs, setCommandDefs] = useState<CommandDefinition[]>([]);

  // State for sharing workspace context with agents
  const [shareWorkspace, setShareWorkspace] = useState<boolean>(() => {
    return localStorage.getItem('share-workspace') === 'true';
  });

  // Clear stale chat width from localStorage (no longer used - chat area always flex-grows)
  useEffect(() => {
    localStorage.removeItem('main-chat-area-width');
  }, []);
  
  // State for test mode - use prop if provided, otherwise local state
  const [localTestMode, setLocalTestMode] = useState(false);
  const testMode = propTestMode !== undefined ? propTestMode : localTestMode;
  const setTestMode = propSetTestMode || setLocalTestMode;

  const [api] = useState(() => new ChatAPI(serverAddr));
  const wsURL = api.getWebSocketURL(channel);
  
  // Debounce timeout ref for agent list refresh
  const agentRefreshTimeoutRef = useRef<number | null>(null);
  
  // Ref to access RichTextInput methods
  const inputRef = useRef<HTMLTextAreaElement | null>(null);

  // Load layout settings on mount
  useEffect(() => {
    loadLayoutSettings();
  }, [loadLayoutSettings]);

  // Apply layout settings when they change (including initial mount and real-time updates)
  useEffect(() => {
    if (layoutSettings) {
      // Apply panel visibility from settings
      setFileExplorerOpen(layoutSettings.filesPanelVisible);
      setCodeEditorOpen(layoutSettings.editorPanelVisible);
      setPanelOpen(layoutSettings.terminalPanelVisible);
      setMyAgentsPanelOpen(layoutSettings.myAgentsPanelVisible);
      setPendingChangesOpen(layoutSettings.pendingChangesPanelVisible);
    }
  }, [layoutSettings, setPanelOpen, setMyAgentsPanelOpen, setPendingChangesOpen]);

  // Load agents function
  const loadAgents = useCallback(async () => {
    try {
      const agentList = await api.fetchAgents();
      setAgents(agentList);
      
      // Remove agents from loading state if they're now active
      const { loadingAgents, removeLoadingAgent } = useChatStore.getState();
      const activeAgentNames = new Set(agentList.map(agent => agent.name));
      
      loadingAgents.forEach(agentName => {
        if (activeAgentNames.has(agentName)) {
          removeLoadingAgent(agentName);
        }
      });
    } catch (error) {
      console.error('Failed to load agents:', error);
    }
  }, [api, setAgents]);

  // Load counts for badges
  const loadCounts = useCallback(async () => {
    try {
      const [myAgents, removedAgents] = await Promise.all([
        api.fetchMyAgents(),
        api.fetchRemovedAgents()
      ]);
      setTotalAgentsCount(myAgents.length + removedAgents.length);
    } catch (error) {
      console.error('Failed to load counts:', error);
    }
  }, [api]);

  // Load channels
  const loadChannels = useCallback(async () => {
    try {
      const channelList = await api.fetchChannels();
      setChannels(channelList);
    } catch (error) {
      console.error('Failed to load channels:', error);
    }
  }, [api, setChannels]);

  // Handle switching channel: switch store state, then load fresh messages
  const handleSwitchChannel = useCallback(async (channelName: string) => {
    if (channelName === channel) return;
    switchChannel(channelName);
    localStorage.setItem('last-channel', channelName);
    try {
      const msgs = await api.fetchMessages(channelName, 50);
      setMessages(msgs);
    } catch (error) {
      console.error('Failed to load messages for channel:', error);
    }
  }, [api, channel, switchChannel, setMessages]);

  // Create a custom channel
  const handleCreateChannel = useCallback(async (name: string, description: string, agentIds: string[]) => {
    try {
      await api.createChannel(name, description, 'custom', agentIds, username);
      await loadChannels();
      await handleSwitchChannel(name);
    } catch (error) {
      console.error('Failed to create channel:', error);
    }
  }, [api, username, loadChannels, handleSwitchChannel]);

  // Create a DM channel with an agent
  const handleCreateDM = useCallback(async (agentId: string) => {
    try {
      const ch = await api.createChannel('', '', 'dm', [agentId], username);
      await loadChannels();
      await handleSwitchChannel(ch.name);
    } catch (error) {
      console.error('Failed to create DM channel:', error);
    }
  }, [api, username, loadChannels, handleSwitchChannel]);

  // Debounced agent refresh (prevents excessive API calls)
  const debouncedRefreshAgents = useCallback(() => {
    if (agentRefreshTimeoutRef.current) {
      clearTimeout(agentRefreshTimeoutRef.current);
    }
    agentRefreshTimeoutRef.current = window.setTimeout(() => {
      loadAgents();
      loadCounts();
      loadChannels();
    }, 300);
  }, [loadAgents, loadCounts, loadChannels]);

  // WebSocket connection
  const { status } = useWebSocket({
    url: wsURL,
    onMessage: async (message: Message) => {
      // Handle all agent_status messages - never add them to chat
      if (message.type === 'agent_status') {
        // Handle thinking status -> typing indicator
        if (message.metadata?.thinking_status) {
          const thinkingStatus = message.metadata.thinking_status as ThinkingStatusMetadata['thinking_status'];
          
          if (thinkingStatus === 'started') {
            addThinkingAgent(message.from.id, message.from.name, message.from.type);
          } else if (thinkingStatus === 'completed' || thinkingStatus === 'error') {
            removeThinkingAgent(message.from.id);
          }
        }
        
        // Handle status updates - update agent info immediately
        if (message.metadata?.indexing_status !== undefined || 
            message.metadata?.index_progress !== undefined ||
            message.metadata?.status !== undefined ||
            message.from.is_paused !== undefined) {
          const statusUpdates: Partial<typeof message.from> = {};
          
          if (message.metadata?.indexing_status !== undefined) {
            statusUpdates.indexing_status = message.metadata.indexing_status as string;
          }
          if (message.metadata?.index_progress !== undefined) {
            statusUpdates.index_progress = message.metadata.index_progress as number;
          }
          if (message.metadata?.status !== undefined) {
            statusUpdates.status = message.metadata.status as string;
          }
          if (message.from.is_paused !== undefined) {
            statusUpdates.is_paused = message.from.is_paused;
          }
          
          // Update agent status immediately in the store
          updateAgentStatus(message.from.id, statusUpdates);
        }
        
        // For all agent_status types: also debounce a full refresh for safety
        debouncedRefreshAgents();
        return; // Never add agent_status to message list
      }
      
      // Handle thread messages - only update metadata, ThreadPanel's WebSocket will add the actual message
      if (message.is_thread_reply && message.thread_id) {
        // Just update thread metadata (don't add message - ThreadPanel's WS will do that)
        try {
          const metadata = await api.fetchThreadMetadata(message.thread_id);
          updateThreadMetadata(message.thread_id, metadata);
        } catch (error) {
          console.error('Failed to fetch thread metadata:', error);
        }
      } else {
        // Add channel messages to main message list
        addMessage(message);
        
        // Check for suggested commands in the message
        if (message.metadata?.suggested_commands) {
          const suggestions = message.metadata.suggested_commands as any[];
          suggestions.forEach((suggestion) => {
            addSuggestedCommand(suggestion);
          });
        }
      }
      
      // Clear thinking indicator when agent sends actual message
      if (message.type === 'chat' || message.type === 'answer') {
        removeThinkingAgent(message.from.id);
      }
      
      // Auto-refresh agents for join/leave events
      if (message.type === 'agent_join' || message.type === 'agent_leave') {
        debouncedRefreshAgents();
      }
    },
    onConnect: () => {
      console.log('Connected to chat');
      setConnectionStatus('connected');
      loadInitialData();
    },
    onDisconnect: () => {
      console.log('Disconnected from chat');
      setConnectionStatus('disconnected');
    },
    onError: (error) => {
      console.error('WebSocket error:', error);
      setConnectionStatus('error');
    },
  });

  // Load initial data when connected
  const loadInitialData = async () => {
    try {
      // Load existing messages
      const existingMessages = await api.fetchMessages(channel, 50);
      setMessages(existingMessages);

      // Load agents, counts, channels, and command definitions
      await Promise.all([loadAgents(), loadCounts(), loadChannels()]);

      try {
        const defs = await api.fetchCommands();
        setCommandDefs(defs);
      } catch (err) {
        console.error('Failed to load command definitions:', err);
      }

      // Send join message
      setTimeout(async () => {
        await api.sendMessage(
          channel,
          `${username} has joined the chat`,
          { name: username, type: 'human' },
          'system_info'
        );
      }, 500);
    } catch (error) {
      console.error('Failed to load initial data:', error);
    }
  };

  const handleSendMessage = async (content: string, metadata?: Record<string, any>) => {
    setIsTyping(true);

    // Assemble workspace context if sharing is enabled
    let mergedMetadata = metadata;
    if (shareWorkspace) {
      const editorTabs = useEditorStore.getState().tabs;
      const activeTabId = useEditorStore.getState().activeTabId;
      const { workspaces, activeWorkspaceId, fileTree } = useFileExplorerStore.getState();
      const activeWorkspace = workspaces.find(w => w.id === activeWorkspaceId) ?? workspaces[0];

      // Build file tree from the active workspace's loaded nodes
      const nodes = activeWorkspace ? (fileTree[activeWorkspace.id] ?? []) : [];

      const workspaceContext: WorkspaceContext = {
        workspace_name: activeWorkspace?.name ?? '',
        workspace_path: activeWorkspace?.path ?? '',
        file_tree: buildFileTreeString(nodes, 3),
        open_files: editorTabs.map(tab => ({
          path: tab.path,
          language: tab.language ?? 'text',
          content: tab.content.substring(0, 10000),
          is_active: tab.id === activeTabId,
        })),
      };

      mergedMetadata = { ...metadata, workspace_context: workspaceContext };
    }

    try {
      await api.sendMessage(
        channel,
        content,
        { name: username, type: 'human' },
        'question',
        mergedMetadata
      );
    } catch (error) {
      console.error('Failed to send message:', error);
      // TODO: Show error to user
    }
  };

  // Ensure command definitions are loaded, fetching them if needed
  const ensureCommandDefs = async () => {
    if (commandDefs.length > 0) return;
    try {
      const defs = await api.fetchCommands();
      setCommandDefs(defs);
    } catch (err) {
      console.error('Failed to load command definitions:', err);
    }
  };

  // Handle command executed from command palette
  const handleCommandExecute = async (commandString: string) => {
    if (inputRef.current && (inputRef.current as any).clearInput) {
      (inputRef.current as any).clearInput();
    }
    await handleSendMessage(commandString);
  };

  // Handle slash trigger from input
  const handleSlashTrigger = async (query: string) => {
    await ensureCommandDefs();
    setCommandPaletteFilter(query);
    setCommandPaletteOpen(true);
  };

  // Open command palette from toolbar button
  const openCommandPalette = async () => {
    await ensureCommandDefs();
    setCommandPaletteFilter('');
    setCommandPaletteOpen(true);
  };

  // Handle clicking on agent in sidebar to insert mention
  const handleAgentClick = (agentName: string) => {
    if (inputRef.current && (inputRef.current as any).insertMentionText) {
      (inputRef.current as any).insertMentionText(agentName);
    }
  };

  const handleRemoveAgent = async (_agentId: string, agentName: string) => {
    if (window.confirm(`Remove ${agentName} from conversation? (You can recall later)`)) {
      try {
        await api.removeAgent(channel, agentName, { name: username, type: 'human' });
        // Agent list will be refreshed automatically via WebSocket agent_leave message
      } catch (error) {
        console.error('Failed to remove agent:', error);
      }
    }
  };

  const handleExportAgent = async (agentName: string) => {
    console.log('handleExportAgent called with:', agentName, 'channel:', channel);
    try {
      await api.exportAgent(channel, agentName);
      console.log('Export command sent successfully');
    } catch (error) {
      console.error('Failed to export agent:', error);
    }
  };

  const handleLogout = async () => {
    try {
      // Clear saved credentials
      await clearCredentials();
      
      // Reset chat store state
      logout();
      
      // Notify parent to switch to login view
      if (onLogout) {
        onLogout();
      }
    } catch (error) {
      console.error('[ChatWindow] Failed to logout:', error);
    }
  };

  // Connection status indicator
  const getStatusColor = () => {
    switch (status) {
      case 'connected':
        return 'bg-green-500';
      case 'connecting':
        return 'bg-yellow-500';
      case 'error':
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  const getStatusText = () => {
    switch (status) {
      case 'connected':
        return 'Connected';
      case 'connecting':
        return 'Connecting...';
      case 'error':
        return 'Connection Error';
      default:
        return 'Disconnected';
    }
  };

  // Find parent message for open thread
  const parentMessage = openThreadId ? messages.find(m => m.id === openThreadId) : null;

  return (
    <ErrorBoundary>
      <div className="flex flex-col h-screen bg-slack-bg">
      {/* Top Toolbar - always visible, spans full width */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-slack-border bg-slack-bgHover flex-shrink-0">
        <div className="flex items-center gap-2">
          {(() => {
            const ch = channels.find(c => c.name === channel);
            const isDM = ch?.type === 'dm';
            const agentCount = ch?.agents?.length ?? 0;
            return (
              <>
                <h1 className="text-sm font-bold text-slack-text">
                  {isDM ? '@ ' : '# '}{isDM && ch?.agents?.[0] ? ch.agents[0].name : channel}
                </h1>
                {ch?.description && (
                  <span className="text-xs text-slack-textMuted hidden sm:inline truncate max-w-[200px]" title={ch.description}>
                    {ch.description}
                  </span>
                )}
                {agentCount > 0 && !isDM && (
                  <span className="text-[10px] text-slack-textMuted bg-slack-bgHover px-1.5 py-0.5 rounded">
                    {agentCount} agent{agentCount !== 1 ? 's' : ''}
                  </span>
                )}
              </>
            );
          })()}
          <div className="flex items-center gap-1.5 text-xs">
            <div className={`w-1.5 h-1.5 rounded-full ${getStatusColor()}`} />
            <span className="text-slack-textMuted">{getStatusText()}</span>
          </div>
        </div>
        
        <div className="flex items-center gap-1">
          <button
            onClick={openCommandPalette}
            className="w-7 h-7 bg-indigo-600 hover:bg-indigo-700 text-white rounded transition-colors flex items-center justify-center font-mono text-xs font-bold"
            title="Command palette"
          >
            /
          </button>

          <button
            onClick={() => {
              const next = !shareWorkspace;
              setShareWorkspace(next);
              localStorage.setItem('share-workspace', String(next));
            }}
            className={`w-7 h-7 rounded transition-colors flex items-center justify-center relative ${
              shareWorkspace
                ? 'bg-purple-600 hover:bg-purple-700 text-white ring-1 ring-purple-400 ring-offset-1 ring-offset-slack-bg'
                : 'bg-slack-bgHover hover:bg-slack-border text-slack-textMuted'
            }`}
            title={shareWorkspace ? 'Workspace sharing ON — agents can see your files' : 'Share workspace context with agents'}
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
            </svg>
            {shareWorkspace && (
              <span className="absolute -bottom-0.5 -right-0.5 bg-green-500 rounded-full h-2 w-2 border border-slack-bg" />
            )}
          </button>

          <button
            onClick={() => setPendingChangesOpen(true)}
            className="w-7 h-7 bg-orange-600 hover:bg-orange-700 text-white rounded transition-colors flex items-center justify-center relative"
            title="Pending changes"
          >
            <PendingChangesIcon className="w-3.5 h-3.5" />
          </button>
          
          <button
            onClick={() => setMyAgentsPanelOpen(true)}
            className="w-7 h-7 bg-slack-accent hover:bg-slack-accentHover text-white rounded transition-colors flex items-center justify-center relative"
            title="My agents"
          >
            <MyAgentsIcon className="w-3.5 h-3.5" />
            {totalAgentsCount > 0 && (
              <span className="absolute -bottom-0.5 -right-0.5 bg-white text-slack-accent text-[9px] font-bold rounded-full h-3.5 w-3.5 flex items-center justify-center">
                {totalAgentsCount}
              </span>
            )}
          </button>
          
          <button
            onClick={() => setFileExplorerOpen(true)}
            className="w-7 h-7 bg-green-600 hover:bg-green-700 text-white rounded transition-colors flex items-center justify-center"
            title="File explorer"
          >
            <FilesIcon className="w-3.5 h-3.5" />
          </button>
          
          <button
            onClick={() => setCodeEditorOpen(true)}
            className="w-7 h-7 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors flex items-center justify-center"
            title="Code editor"
          >
            <EditorIcon className="w-3.5 h-3.5" />
          </button>
          
          <button
            onClick={() => useTerminalStore.getState().togglePanel()}
            className="w-7 h-7 bg-gray-600 hover:bg-gray-700 text-white rounded transition-colors flex items-center justify-center relative"
            title="Terminal (⌘J)"
          >
            <TerminalIcon className="w-3.5 h-3.5" />
            {useTerminalStore.getState().suggestedCommands.length > 0 && (
              <span className="absolute -bottom-0.5 -right-0.5 bg-yellow-500 text-black text-[9px] font-bold rounded-full h-3.5 w-3.5 flex items-center justify-center">
                {useTerminalStore.getState().suggestedCommands.length}
              </span>
            )}
          </button>
          
          <div className="w-px h-5 bg-slack-border mx-0.5" />
          
          {onOpenSettings && (
            <button
              onClick={onOpenSettings}
              className="w-7 h-7 text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover rounded transition-colors flex items-center justify-center"
              title="Settings (⌘,)"
            >
              <SettingsIcon className="w-3.5 h-3.5" />
            </button>
          )}
          
          {onLogout && (
            <button
              onClick={handleLogout}
              className="w-7 h-7 text-slack-textMuted hover:text-red-500 hover:bg-red-500/10 rounded transition-colors flex items-center justify-center"
              title="Logout"
            >
              <LogoutIcon className="w-3.5 h-3.5" />
            </button>
          )}
          
          <div className="w-px h-5 bg-slack-border mx-0.5" />
          
          <div className="text-xs text-slack-textMuted">
            <span className="font-medium text-slack-text">{username}</span>
            <span className="mx-1">•</span>
            <span>{serverAddr}</span>
          </div>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Channel Sidebar */}
        <ChannelSidebar
          channels={channels}
          agents={agents}
          onSwitchChannel={handleSwitchChannel}
          onCreateChannel={() => setCreateChannelOpen(true)}
          onCreateDM={handleCreateDM}
        />

        {/* File Explorer Panel - slides in from left */}
        {fileExplorerOpen && (
          <FileExplorerPanel
            onClose={() => setFileExplorerOpen(false)}
            onFileOpen={() => setCodeEditorOpen(true)}
          />
        )}

        {/* Code Editor Panel - slides in from left */}
        {codeEditorOpen && (
          <CodeEditorPanel
            onClose={() => setCodeEditorOpen(false)}
          />
        )}

        {/* Main Chat Area - always flex-grow to fill remaining space */}
        <div 
          className="flex flex-col transition-all duration-300 ease-in-out relative overflow-hidden"
          style={{ 
            flex: '1 1 auto',
            minWidth: '300px',
          }}
        >

        {/* Messages */}
        <MessageList
          messages={messages}
          threadMetadata={threadMetadata}
          onOpenThread={openThread}
        />

        {/* Typing Indicator */}
        {thinkingAgents.size > 0 && (
          <TypingIndicator agents={Array.from(thinkingAgents.values())} />
        )}

        {/* Input */}
        <RichTextInput
          onSend={handleSendMessage}
          disabled={status !== 'connected'}
          placeholder={
            status === 'connected'
              ? 'Type your message here...'
              : 'Connecting...'
          }
          agents={agents}
          ref={inputRef}
          onSlashTrigger={handleSlashTrigger}
        />
        </div>

        {/* Thread Panel - slides in when thread is open */}
        {openThreadId && parentMessage && (
          <ThreadPanel
            threadId={openThreadId}
            parentMessage={parentMessage}
            onClose={closeThread}
          />
        )}

        {/* My Agents Panel - slides in from right */}
        {myAgentsPanelOpen && (
          <MyAgentsPanel
            onClose={() => setMyAgentsPanelOpen(false)}
          />
        )}

        {/* Pending Changes Panel */}
        {pendingChangesOpen && (
          <PendingChangesPanel
            onClose={() => setPendingChangesOpen(false)}
          />
        )}


        {/* Sidebar - Agent List */}
        <AgentList 
          agents={agents} 
          onRefresh={loadAgents}
          onAgentClick={handleAgentClick}
          onRemoveAgent={handleRemoveAgent}
          onExportAgent={handleExportAgent}
        />
      </div>

      {/* Terminal Panel - slides up from bottom */}
      <div 
        className="transition-all duration-300 ease-in-out overflow-hidden"
        style={{ height: isPanelOpen ? `${panelHeight}px` : '0px' }}
      >
        <TerminalPanel height={panelHeight} />
      </div>
      
      {/* Command Palette */}
      <CommandPalette
        commands={commandDefs}
        agents={agents}
        isOpen={commandPaletteOpen}
        initialFilter={commandPaletteFilter}
        onClose={() => {
          setCommandPaletteOpen(false);
          if (inputRef.current && (inputRef.current as any).clearInput) {
            (inputRef.current as any).clearInput();
          }
        }}
        onExecute={handleCommandExecute}
      />

      {/* Create Channel Modal */}
      <CreateChannelModal
        agents={agents}
        isOpen={createChannelOpen}
        onClose={() => setCreateChannelOpen(false)}
        onCreate={handleCreateChannel}
      />

      {/* Toast Notifications */}
      <ToastContainer />
      </div>
    </ErrorBoundary>
  );
}

