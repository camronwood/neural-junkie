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
}

export interface TerminalCommand {
  id: string;
  command: string;
  status: 'pending' | 'executing' | 'completed' | 'failed';
  exit_code: number;
  stdout: string;
  stderr: string;
  duration_ms: number;
  started_at: string;
  ended_at?: string;
}

export interface ConsoleOutput {
  type: 'command' | 'stdout' | 'stderr' | 'system';
  content: string;
  timestamp: string;
}

interface TerminalStore {
  // Panel state
  isPanelOpen: boolean;
  togglePanel: () => void;
  setPanelOpen: (open: boolean) => void;

  // Command suggestions
  suggestedCommands: CommandSuggestion[];
  addSuggestedCommand: (suggestion: CommandSuggestion) => void;
  removeSuggestedCommand: (id: string) => void;
  clearSuggestedCommands: () => void;

  // Command history
  commandHistory: TerminalCommand[];
  addCommand: (command: TerminalCommand) => void;
  updateCommand: (id: string, updates: Partial<TerminalCommand>) => void;
  clearHistory: () => void;

  // Currently executing command
  executingCommand: TerminalCommand | null;
  setExecutingCommand: (command: TerminalCommand | null) => void;

  // Panel height (for resizing)
  panelHeight: number;
  setPanelHeight: (height: number) => void;

  // Console state
  consoleHistory: string[];
  consoleHistoryIndex: number;
  consoleOutput: ConsoleOutput[];
  currentWorkingDir: string;
  addConsoleCommand: (command: string) => void;
  addConsoleOutput: (type: ConsoleOutput['type'], content: string) => void;
  clearConsoleOutput: () => void;
  setCurrentWorkingDir: (dir: string) => void;
  navigateHistory: (direction: 'up' | 'down') => string | null;
}

export const useTerminalStore = create<TerminalStore>((set) => ({
  // Panel state
  isPanelOpen: false,
  togglePanel: () => set((state) => ({ isPanelOpen: !state.isPanelOpen })),
  setPanelOpen: (open) => set({ isPanelOpen: open }),

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

  // Command history
  commandHistory: [],
  addCommand: (command) =>
    set((state) => ({
      commandHistory: [...state.commandHistory, command],
    })),
  updateCommand: (id, updates) =>
    set((state) => ({
      commandHistory: state.commandHistory.map((cmd) =>
        cmd.id === id ? { ...cmd, ...updates } : cmd
      ),
    })),
  clearHistory: () => set({ commandHistory: [] }),

  // Currently executing command
  executingCommand: null,
  setExecutingCommand: (command) => set({ executingCommand: command }),

  // Panel height
  panelHeight: 300, // Default height in pixels
  setPanelHeight: (height) => set({ panelHeight: height }),

  // Console state
  consoleHistory: [],
  consoleHistoryIndex: -1,
  consoleOutput: [],
  currentWorkingDir: '~',
  addConsoleCommand: (command) =>
    set((state) => ({
      consoleHistory: [...state.consoleHistory, command],
      consoleHistoryIndex: state.consoleHistory.length,
    })),
  addConsoleOutput: (type, content) =>
    set((state) => ({
      consoleOutput: [
        ...state.consoleOutput,
        {
          type,
          content,
          timestamp: new Date().toISOString(),
        },
      ],
    })),
  clearConsoleOutput: () => set({ consoleOutput: [] }),
  setCurrentWorkingDir: (dir) => set({ currentWorkingDir: dir }),
  navigateHistory: (direction: 'up' | 'down'): string | null => {
    const state = useTerminalStore.getState();
    let newIndex = state.consoleHistoryIndex;
    
    if (direction === 'up') {
      newIndex = Math.max(0, newIndex - 1);
    } else {
      newIndex = Math.min(state.consoleHistory.length, newIndex + 1);
    }
    
    set({ consoleHistoryIndex: newIndex });
    
    if (newIndex >= 0 && newIndex < state.consoleHistory.length) {
      return state.consoleHistory[newIndex];
    }
    return null;
  },
}));
