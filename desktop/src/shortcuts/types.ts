export type ShortcutScope =
  | 'global'
  | 'chat'
  | 'ide'
  | 'modal'
  | 'terminalFocused'
  | 'monacoFocused';

export type ShortcutHandler = () => void | Promise<void>;

export interface ParsedChord {
  key: string;
  mod: boolean;
  shift: boolean;
  alt: boolean;
  ctrl: boolean;
}

export interface ShortcutDefinition {
  id: string;
  chord: string;
  label: string;
  scope: ShortcutScope;
  priority: number;
  preventDefault?: boolean;
  allowInInput?: boolean;
  skipInMonaco?: boolean;
  skipInTerminal?: boolean;
  when?: () => boolean;
  handlerId: keyof ShortcutHandlerMap;
}

export interface ShortcutHandlerMap {
  openSettings: ShortcutHandler;
  toggleTerminal: ShortcutHandler;
  openCommandPalette: ShortcutHandler;
  openModelLibrary: ShortcutHandler;
  openChatFind: ShortcutHandler;
  handleEscape: ShortcutHandler;
  toggleChannelSidebar: ShortcutHandler;
  toggleFileExplorer: ShortcutHandler;
  toggleTaskManagement: ShortcutHandler;
  toggleGitPanel: ShortcutHandler;
  toggleProblemsPanel: ShortcutHandler;
  togglePendingChanges: ShortcutHandler;
  toggleMyAgents: ShortcutHandler;
  toggleChatPanel: ShortcutHandler;
  toggleToolbarSidebar: ShortcutHandler;
  toggleIdeLayout: ShortcutHandler;
  prevChannel: ShortcutHandler;
  nextChannel: ShortcutHandler;
  focusChannelSearch: ShortcutHandler;
  createChannel: ShortcutHandler;
  createNewDm: ShortcutHandler;
  openWorkspaceSwitcher: ShortcutHandler;
  quickOpen: ShortcutHandler;
  goToSymbol: ShortcutHandler;
  fastEdit: ShortcutHandler;
  focusComposer: ShortcutHandler;
  saveActiveTab: ShortcutHandler;
  saveAllTabs: ShortcutHandler;
  closeActiveTab: ShortcutHandler;
  openCodeEditor: ShortcutHandler;
  cycleEditorTabForward: ShortcutHandler;
  cycleEditorTabBackward: ShortcutHandler;
  newRunbook: ShortcutHandler;
  approveFirstPending: ShortcutHandler;
  rejectFirstPending: ShortcutHandler;
  closeThread: ShortcutHandler;
  clearTerminal: ShortcutHandler;
}

export type ShortcutHandlerId = keyof ShortcutHandlerMap;
