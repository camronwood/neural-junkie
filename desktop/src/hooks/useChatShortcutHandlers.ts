import { useEffect, useRef, type MutableRefObject, type RefObject } from 'react';
import { shallow } from 'zustand/shallow';
import { useShortcutHandlersStore } from '../stores/shortcutHandlersStore';
import { useShortcutContextStore } from '../stores/shortcutContextStore';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useTerminalStore } from '../stores/terminalStore';
import { useApprovalStore } from '../stores/approvalStore';
import { useChatStore } from '../stores/chatStore';
import { useSettingsStore } from '../stores/settingsStore';
import { useIdeOverlayStore } from '../stores/ideOverlayStore';
import { adjacentChannel, buildNavigableChannelList } from '../utils/sidebarChannelNav';
import { panelsForPreset, type LayoutPreset } from '../utils/layoutPresets';
import type { Settings } from '../stores/settingsStore';

export interface ChatShortcutHandlerDeps {
  onOpenSettings?: () => void;
  channelSearchRef: RefObject<HTMLInputElement | null>;
  inputRef: RefObject<HTMLTextAreaElement | null>;
  ideEnabled: boolean;
  ideLayout: boolean;
  codeEditorOpen: boolean;
  showAgentStop: boolean;
  useSidebarChips: boolean;
  channelSidebarOpen: boolean;
  setChannelSidebarOpen: (fn: (prev: boolean) => boolean) => void;
  setFileExplorerOpen: (fn: (prev: boolean) => boolean) => void;
  setCodeEditorOpen: (fn: (prev: boolean) => boolean) => void;
  setTaskManagementOpen: (fn: (prev: boolean) => boolean) => void;
  onOpenPendingChanges: () => void;
  setToolbarSidebarOpen: (fn: (prev: boolean) => boolean) => void;
  setCommandPaletteOpen: (open: boolean) => void;
  setModelLibraryOpen: (open: boolean) => void;
  setDomainPacksOpen: (open: boolean) => void;
  setChatFindOpen: (open: boolean) => void;
  setCreateChannelOpen: (open: boolean) => void;
  setCreateNewDmOpen: (open: boolean) => void;
  chatPanelVisible: boolean;
  openCommandPalette: () => void | Promise<void>;
  handleChannelInterject: () => void | Promise<void>;
  handleNewRunbook: () => void | Promise<void>;
  handleSwitchChannel: (name: string) => void | Promise<void>;
  handleCreateDM: (agentId: string) => void | Promise<void>;
  updateLayoutSettings: (patch: Record<string, unknown>) => void | Promise<void>;
  approveFirstPendingRef: MutableRefObject<(() => void | Promise<void>) | null>;
  rejectFirstPendingRef: MutableRefObject<(() => void | Promise<void>) | null>;
}

export function useChatShortcutHandlers(deps: ChatShortcutHandlerDeps) {
  const registerHandlers = useShortcutHandlersStore((s) => s.registerHandlers);
  const unregisterHandlers = useShortcutHandlersStore((s) => s.unregisterHandlers);
  const setGates = useShortcutHandlersStore((s) => s.setGates);
  const closeTopOverlay = useShortcutContextStore((s) => s.closeTopOverlay);

  const { channel, channels, agents, openThreadId } = useChatStore(
    (s) => ({
      channel: s.channel,
      channels: s.channels,
      agents: s.agents,
      openThreadId: s.openThreadId,
    }),
    shallow
  );
  const setMyAgentsPanelOpen = useChatStore((s) => s.setMyAgentsPanelOpen);
  const myAgentsPanelOpen = useChatStore((s) => s.myAgentsPanelOpen);
  const closeThread = useChatStore((s) => s.closeThread);
  const togglePanel = useTerminalStore((s) => s.togglePanel);
  const requestClearBuffer = useTerminalStore((s) => s.requestClearBuffer);
  const pendingTools = useApprovalStore((s) => s.pendingTools);
  const suggestedCommands = useTerminalStore((s) => s.suggestedCommands);
  const { settings, isLoaded: settingsLoaded, layoutSettings } = useSettingsStore(
    (s) => ({ settings: s.settings, isLoaded: s.isLoaded, layoutSettings: s.layoutSettings }),
    shallow
  );
  const requestWorkspaceSwitcher = useFileExplorerStore((s) => s.requestWorkspaceSwitcher);

  const depsRef = useRef(deps);
  depsRef.current = deps;

  useEffect(() => {
    setGates({
      ideEnabled: deps.ideEnabled,
      ideLayout: deps.ideLayout,
      codeEditorOpen: deps.codeEditorOpen,
      showAgentStop: deps.showAgentStop,
      hasPendingApprovals: pendingTools.length + suggestedCommands.length > 0,
      threadOpen: Boolean(openThreadId),
      chatConnected: true,
    });
  }, [
    deps.ideEnabled,
    deps.ideLayout,
    deps.codeEditorOpen,
    deps.showAgentStop,
    pendingTools.length,
    suggestedCommands.length,
    openThreadId,
    setGates,
  ]);

  useEffect(() => {
    const onFocusIn = (e: FocusEvent) => {
      const t = e.target as HTMLElement | null;
      setGates({
        terminalFocused: Boolean(t?.closest?.('.xterm')),
        monacoFocused: Boolean(t?.closest?.('.monaco-editor')),
      });
    };
    const onFocusOut = () => {
      const active = document.activeElement as HTMLElement | null;
      setGates({
        terminalFocused: Boolean(active?.closest?.('.xterm')),
        monacoFocused: Boolean(active?.closest?.('.monaco-editor')),
      });
    };
    document.addEventListener('focusin', onFocusIn);
    document.addEventListener('focusout', onFocusOut);
    return () => {
      document.removeEventListener('focusin', onFocusIn);
      document.removeEventListener('focusout', onFocusOut);
    };
  }, [setGates]);

  useEffect(() => {
    const d = depsRef;
    registerHandlers({
      openSettings: () => d.current.onOpenSettings?.(),
      toggleTerminal: () => togglePanel(),
      openCommandPalette: () => void d.current.openCommandPalette(),
      openModelLibrary: () => d.current.setModelLibraryOpen(true),
      openDomainPacks: () => d.current.setDomainPacksOpen(true),
      openChatFind: () => d.current.setChatFindOpen(true),
      handleEscape: async () => {
        if (closeTopOverlay()) return;
        if (d.current.showAgentStop) {
          await d.current.handleChannelInterject();
        }
      },
      toggleChannelSidebar: () => {
        d.current.setChannelSidebarOpen((prev) => {
          const next = !prev;
          localStorage.setItem('channel-sidebar-open', String(next));
          return next;
        });
      },
      toggleFileExplorer: () => {
        d.current.setFileExplorerOpen((prev) => {
          const next = !prev;
          if (next) {
            void d.current.updateLayoutSettings({ filesPanelVisible: true });
          }
          return next;
        });
      },
      toggleTaskManagement: () => d.current.setTaskManagementOpen((o) => !o),
      toggleGitPanel: () => {
        const s = useIdeOverlayStore.getState();
        s.setGitModalOpen(!s.gitModalOpen);
      },
      toggleProblemsPanel: () => {
        const s = useIdeOverlayStore.getState();
        s.setProblemsOpen(!s.problemsOpen);
      },
      togglePendingChanges: () => d.current.onOpenPendingChanges(),
      toggleMyAgents: () => setMyAgentsPanelOpen(!myAgentsPanelOpen),
      toggleChatPanel: () => {
        void d.current.updateLayoutSettings({
          chatPanelVisible: !d.current.chatPanelVisible,
        });
      },
      toggleToolbarSidebar: () => {
        if (!d.current.useSidebarChips) return;
        d.current.setToolbarSidebarOpen((prev) => {
          const next = !prev;
          localStorage.setItem('toolbar-sidebar-open', String(next));
          return next;
        });
      },
      toggleIdeLayout: () => {
        const next: LayoutPreset = d.current.ideLayout ? 'team' : 'ide';
        void d.current.updateLayoutSettings(panelsForPreset(next));
      },
      prevChannel: () => {
        const list = buildNavigableChannelList(channels, agents, {
          settingsLoaded,
          settings: settings as Settings,
          sidebarAgentsVisible: layoutSettings.sidebarAgentsVisible,
        });
        const next = adjacentChannel(list, channel, 'prev');
        if (!next) return;
        if (next.type === 'agent-shortcut') {
          const agent = agents.find((a) => a.name === next.name);
          if (agent) void d.current.handleCreateDM(agent.id);
        } else {
          void d.current.handleSwitchChannel(next.name);
        }
      },
      nextChannel: () => {
        const list = buildNavigableChannelList(channels, agents, {
          settingsLoaded,
          settings: settings as Settings,
          sidebarAgentsVisible: layoutSettings.sidebarAgentsVisible,
        });
        const next = adjacentChannel(list, channel, 'next');
        if (!next) return;
        if (next.type === 'agent-shortcut') {
          const agent = agents.find((a) => a.name === next.name);
          if (agent) void d.current.handleCreateDM(agent.id);
        } else {
          void d.current.handleSwitchChannel(next.name);
        }
      },
      focusChannelSearch: () => d.current.channelSearchRef.current?.focus(),
      createChannel: () => d.current.setCreateChannelOpen(true),
      createNewDm: () => d.current.setCreateNewDmOpen(true),
      openWorkspaceSwitcher: () => {
        d.current.setFileExplorerOpen(() => true);
        void d.current.updateLayoutSettings({ filesPanelVisible: true });
        requestWorkspaceSwitcher();
      },
      quickOpen: () => useIdeOverlayStore.getState().setQuickOpenOpen(true),
      goToSymbol: () => useIdeOverlayStore.getState().setSymbolModalOpen(true),
      fastEdit: () => useIdeOverlayStore.getState().setFastEditOpen(true),
      focusComposer: () => d.current.inputRef.current?.focus(),
      saveActiveTab: async () => {
        const { activeTabId, saveTab } = useEditorStore.getState();
        if (activeTabId) await saveTab(activeTabId);
      },
      saveAllTabs: async () => {
        await useEditorStore.getState().saveAllTabs();
      },
      closeActiveTab: () => {
        const { activeTabId, closeTab } = useEditorStore.getState();
        if (activeTabId) closeTab(activeTabId);
      },
      openCodeEditor: () => {
        d.current.setCodeEditorOpen(() => true);
        void d.current.updateLayoutSettings({ editorPanelVisible: true });
      },
      cycleEditorTabForward: () => useEditorStore.getState().cycleActiveTab(1),
      cycleEditorTabBackward: () => useEditorStore.getState().cycleActiveTab(-1),
      newRunbook: () => void d.current.handleNewRunbook(),
      approveFirstPending: () => void d.current.approveFirstPendingRef.current?.(),
      rejectFirstPending: () => void d.current.rejectFirstPendingRef.current?.(),
      closeThread: () => closeThread(),
      clearTerminal: () => requestClearBuffer(),
    });

    return () => {
      unregisterHandlers([
        'openSettings',
        'toggleTerminal',
        'openCommandPalette',
        'openModelLibrary',
        'openDomainPacks',
        'openChatFind',
        'handleEscape',
        'toggleChannelSidebar',
        'toggleFileExplorer',
        'toggleTaskManagement',
        'toggleGitPanel',
        'toggleProblemsPanel',
        'togglePendingChanges',
        'toggleMyAgents',
        'toggleChatPanel',
        'toggleToolbarSidebar',
        'toggleIdeLayout',
        'prevChannel',
        'nextChannel',
        'focusChannelSearch',
        'createChannel',
        'createNewDm',
        'openWorkspaceSwitcher',
        'quickOpen',
        'goToSymbol',
        'fastEdit',
        'focusComposer',
        'saveActiveTab',
        'saveAllTabs',
        'closeActiveTab',
        'openCodeEditor',
        'cycleEditorTabForward',
        'cycleEditorTabBackward',
        'newRunbook',
        'approveFirstPending',
        'rejectFirstPending',
        'closeThread',
        'clearTerminal',
      ]);
    };
  }, [
    registerHandlers,
    unregisterHandlers,
    closeTopOverlay,
    togglePanel,
    requestClearBuffer,
    channels,
    agents,
    channel,
    settings,
    settingsLoaded,
    layoutSettings.sidebarAgentsVisible,
    myAgentsPanelOpen,
    setMyAgentsPanelOpen,
    closeThread,
    requestWorkspaceSwitcher,
  ]);
}
