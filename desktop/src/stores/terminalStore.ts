import { create } from 'zustand';

export interface CommandSuggestion {
  id: string;
  command: string;
  plugin: string;
  description: string;
  is_safe: boolean;
  agent_name: string;
  message_id: string;
  created_at: string;
  /** Optional absolute cwd for execute_command (e.g. collaboration sandbox). */
  cwd?: string;
}

export interface TerminalTab {
  id: string;
  label: string;
  type: 'user' | 'agent';
  agentName?: string;
  cwd: string;
}

interface TerminalStore {
  // Panel state
  isPanelOpen: boolean;
  togglePanel: () => void;
  setPanelOpen: (open: boolean) => void;

  // Panel height (for resizing)
  panelHeight: number;
  setPanelHeight: (height: number) => void;

  // Tabs
  tabs: TerminalTab[];
  activeTabId: string;
  addTab: (tab: TerminalTab) => void;
  removeTab: (id: string) => void;
  setActiveTab: (id: string) => void;
  renameTab: (id: string, label: string) => void;

  // Command suggestions (shown as inline banner)
  suggestedCommands: CommandSuggestion[];
  addSuggestedCommand: (suggestion: CommandSuggestion) => void;
  removeSuggestedCommand: (id: string) => void;
  clearSuggestedCommands: () => void;

  // Legacy fields kept for backward compat during transition
  commandHistory: never[];
  executingCommand: null;
}

let tabCounter = 1;

export const useTerminalStore = create<TerminalStore>((set) => ({
  // Panel state
  isPanelOpen: false,
  togglePanel: () => set((state) => ({ isPanelOpen: !state.isPanelOpen })),
  setPanelOpen: (open) => set({ isPanelOpen: open }),

  // Panel height
  panelHeight: 300,
  setPanelHeight: (height) => set({ panelHeight: height }),

  // Tabs
  tabs: [{ id: 'tab-0', label: 'Terminal', type: 'user' as const, cwd: '~' }],
  activeTabId: 'tab-0',

  addTab: (tab) =>
    set((state) => ({
      tabs: [...state.tabs, tab],
      activeTabId: tab.id,
    })),

  removeTab: (id) =>
    set((state) => {
      const remaining = state.tabs.filter((t) => t.id !== id);
      if (remaining.length === 0) {
        const newTab: TerminalTab = {
          id: `tab-${++tabCounter}`,
          label: 'Terminal',
          type: 'user',
          cwd: '~',
        };
        return { tabs: [newTab], activeTabId: newTab.id };
      }
      const activeGone = state.activeTabId === id;
      return {
        tabs: remaining,
        activeTabId: activeGone ? remaining[remaining.length - 1].id : state.activeTabId,
      };
    }),

  setActiveTab: (id) => set({ activeTabId: id }),

  renameTab: (id, label) =>
    set((state) => ({
      tabs: state.tabs.map((t) => (t.id === id ? { ...t, label } : t)),
    })),

  // Command suggestions
  suggestedCommands: [],
  addSuggestedCommand: (suggestion) =>
    set((state) => ({
      suggestedCommands: [...state.suggestedCommands, suggestion],
    })),
  removeSuggestedCommand: (id) =>
    set((state) => ({
      suggestedCommands: state.suggestedCommands.filter((cmd) => cmd.id !== id),
    })),
  clearSuggestedCommands: () => set({ suggestedCommands: [] }),

  // Legacy stubs
  commandHistory: [],
  executingCommand: null,
}));

export function createNewTab(type: 'user' | 'agent' = 'user', agentName?: string, cwd?: string): TerminalTab {
  const id = `tab-${++tabCounter}`;
  const label = type === 'agent' && agentName ? agentName : 'Terminal';
  return { id, label, type, agentName, cwd: cwd ?? '~' };
}
