import { useTerminalStore } from '../stores/terminalStore';
import {
  PendingChangesIcon,
  MyAgentsIcon,
  FilesIcon,
  EditorIcon,
  TerminalIcon,
  GitIcon,
  ModelLibraryIcon,
  DomainPacksIcon,
  SettingsIcon,
  LogoutIcon,
  TaskManagementIcon,
  ChatPanelIcon,
} from './Icons';
import { conversationModeSettingLabel } from '../utils/conversationMode';
import { workspaceContextModeLabel } from '../utils/outboundChatMetadata';
import type { ConversationModeSetting } from '../utils/conversationMode';
import type { WorkspaceContextMode } from '../constants/promptMetadata';
import type { SettingsTab } from './SettingsModal';
import { OllamaRuntimeChip } from './OllamaRuntimeChip';
import { MemoryMonitorChip } from './MemoryMonitorChip';
import { useApprovalStore } from '../stores/approvalStore';
import { useSettingsStore } from '../stores/settingsStore';
import { formatChord } from '../shortcuts/format';

export type ChatToolbarActionsLayout = 'horizontal' | 'vertical';

export interface ChatToolbarActionsProps {
  layout: ChatToolbarActionsLayout;
  onOpenCommandPalette: () => void;
  chatPanelVisible: boolean;
  onToggleChatPanel: () => void;
  conversationModeSetting: ConversationModeSetting;
  onCycleConversationMode: () => void;
  workspaceContextMode: WorkspaceContextMode;
  onCycleWorkspaceContext: () => void;
  workspaceContextButtonTitle: string;
  onOpenPendingChanges: () => void;
  onOpenFileExplorer: () => void;
  onOpenCodeEditor: () => void;
  phoenixPackInstalled?: boolean;
  onOpenPhoenix?: () => void;
  taskManagementOpen: boolean;
  onToggleTaskManagement: () => void;
  onNewRunbook: () => void;
  onOpenMyAgents: () => void;
  totalAgentsCount: number;
  devPackEnabled: boolean;
  /** When false, IDE layout toggle is shown disabled with install hint. */
  ideLayoutAvailable: boolean;
  onOpenProblems: () => void;
  gitModalOpen: boolean;
  onToggleGitModal: () => void;
  ideLayout: boolean;
  onToggleIdeLayout: () => void;
  ideLayoutButtonTitle: string;
  onOpenModelLibrary: () => void;
  onOpenDomainPacks: () => void;
  enabledPackCount?: number;
  onOpenSettings?: (tab?: SettingsTab) => void;
  onLogout?: () => void;
  username: string;
  serverAddr: string;
}

function ToolbarDivider({ layout }: { layout: ChatToolbarActionsLayout }) {
  if (layout === 'horizontal') {
    return <div className="w-px h-5 bg-slack-border mx-0.5 shrink-0" />;
  }
  return <div className="h-px w-full max-w-[2.25rem] bg-slack-border my-1 shrink-0" aria-hidden />;
}

const iconBtn =
  'w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2';

export function ChatToolbarActions({
  layout,
  onOpenCommandPalette,
  chatPanelVisible,
  onToggleChatPanel,
  conversationModeSetting,
  onCycleConversationMode,
  workspaceContextMode,
  onCycleWorkspaceContext,
  workspaceContextButtonTitle,
  onOpenPendingChanges,
  onOpenFileExplorer,
  onOpenCodeEditor,
  phoenixPackInstalled,
  onOpenPhoenix,
  taskManagementOpen,
  onToggleTaskManagement,
  onNewRunbook,
  onOpenMyAgents,
  totalAgentsCount,
  devPackEnabled,
  ideLayoutAvailable,
  onOpenProblems,
  gitModalOpen,
  onToggleGitModal,
  ideLayout,
  onToggleIdeLayout,
  ideLayoutButtonTitle,
  onOpenModelLibrary,
  onOpenDomainPacks,
  enabledPackCount = 0,
  onOpenSettings,
  onLogout,
  username,
  serverAddr,
}: ChatToolbarActionsProps) {
  const suggestedCount = useTerminalStore((s) => s.suggestedCommands.length);
  const pendingToolCount = useApprovalStore((s) => s.pendingTools.length);
  const approvalCount = suggestedCount + pendingToolCount;
  const memoryMonitorEnabled = useSettingsStore(
    (s) => s.layoutSettings.memoryMonitorEnabled !== false,
  );
  const isVertical = layout === 'vertical';

  const rootClass = isVertical
    ? 'flex flex-col items-center gap-1 py-1'
    : 'flex items-center gap-1.5 flex-nowrap justify-end';

  const ideLayoutDisabledTitle = `Install and enable Software development pack in Domain packs (${formatChord('mod+shift+k')}) to switch IDE layout`;

  const groupClass = isVertical
    ? 'flex flex-col items-center gap-1 w-full'
    : 'flex items-center gap-1';

  return (
    <div className={rootClass}>
      <div className={groupClass} aria-label="Commands">
        <button
          type="button"
          onClick={onOpenCommandPalette}
          className={`${iconBtn} bg-indigo-600 hover:bg-indigo-700 text-white font-mono text-xs font-bold focus-visible:outline-indigo-400`}
          title={`Command palette (${formatChord('mod+shift+p')})`}
          aria-label="Open command palette with Cmd+Shift+P or Ctrl+Shift+P"
        >
          P
        </button>
      </div>

      <ToolbarDivider layout={layout} />

      <div className={groupClass} aria-label="File workspace tools">
        <button
          type="button"
          onClick={onToggleChatPanel}
          className={`${iconBtn} focus-visible:outline-slack-accent ${
            chatPanelVisible
              ? 'bg-slack-accent text-white'
              : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text hover:bg-slack-border'
          }`}
          title={`${chatPanelVisible ? 'Hide main chat' : 'Show main chat'} (${formatChord('mod+shift+c')})`}
          aria-label={chatPanelVisible ? 'Hide main chat panel' : 'Show main chat panel'}
          aria-pressed={chatPanelVisible}
        >
          <ChatPanelIcon className="w-3.5 h-3.5" />
        </button>

        <button
          type="button"
          onClick={onCycleConversationMode}
          className={`${isVertical ? 'w-7 px-0' : 'px-2'} h-7 rounded text-[11px] font-medium transition-colors ${
            conversationModeSetting !== 'auto'
              ? 'bg-sky-600 hover:bg-sky-700 text-white ring-1 ring-sky-400 ring-offset-1 ring-offset-slack-bg'
              : 'bg-slack-bgHover hover:bg-slack-border text-slack-textMuted'
          }`}
          title={`Conversation mode: ${conversationModeSettingLabel(conversationModeSetting)} (click to cycle Auto / Chat / Code)`}
          aria-label={`Conversation mode ${conversationModeSettingLabel(conversationModeSetting)}`}
        >
          {conversationModeSettingLabel(conversationModeSetting)}
        </button>

        <button
          type="button"
          onClick={onCycleWorkspaceContext}
          className={`${iconBtn} relative ${
            workspaceContextMode === 'always'
              ? 'bg-purple-600 hover:bg-purple-700 text-white ring-1 ring-purple-400 ring-offset-1 ring-offset-slack-bg'
              : workspaceContextMode === 'auto'
                ? 'bg-slack-bgHover hover:bg-slack-border text-slack-text ring-1 ring-purple-500/40'
                : 'bg-slack-bgHover hover:bg-slack-border text-slack-textMuted'
          }`}
          title={workspaceContextButtonTitle}
          aria-label={`Workspace context mode ${workspaceContextModeLabel(workspaceContextMode)}`}
        >
          <svg xmlns="http://www.w3.org/2000/svg" className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
            <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
          </svg>
          {workspaceContextMode === 'auto' && (
            <span className="absolute -bottom-0.5 -right-0.5 bg-green-500 rounded-full h-2 w-2 border border-slack-bg" />
          )}
        </button>

        <button
          type="button"
          onClick={onOpenPendingChanges}
          className={`${iconBtn} bg-orange-600 hover:bg-orange-700 text-white focus-visible:outline-orange-400`}
          title={`Pending changes (${formatChord('mod+shift+u')})`}
          aria-label="Open pending file changes"
        >
          <PendingChangesIcon className="w-3.5 h-3.5" />
        </button>

        <button
          type="button"
          onClick={onOpenFileExplorer}
          className={`${iconBtn} bg-green-600 hover:bg-green-700 text-white focus-visible:outline-green-400`}
          title={`File explorer (${formatChord('mod+shift+e')})`}
          aria-label="Open file explorer"
        >
          <FilesIcon className="w-3.5 h-3.5" />
        </button>

        <button
          type="button"
          onClick={onOpenCodeEditor}
          className={`${iconBtn} bg-blue-600 hover:bg-blue-700 text-white focus-visible:outline-blue-400`}
          title={`Code editor (${formatChord('mod+shift+f')})`}
          aria-label="Open code editor"
        >
          <EditorIcon className="w-3.5 h-3.5" />
        </button>

        <button
          type="button"
          onClick={ideLayoutAvailable ? onToggleIdeLayout : undefined}
          disabled={!ideLayoutAvailable}
          className={`${iconBtn} text-[10px] font-bold shrink-0 ${
            !ideLayoutAvailable
              ? 'bg-slack-bgHover text-slack-textMuted opacity-50 cursor-not-allowed'
              : ideLayout
                ? 'bg-violet-600 text-white ring-1 ring-violet-400 ring-offset-1 ring-offset-slack-bg'
                : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
          }`}
          title={ideLayoutAvailable ? ideLayoutButtonTitle : ideLayoutDisabledTitle}
          aria-label="Toggle IDE vs team layout"
          aria-pressed={ideLayoutAvailable ? ideLayout : undefined}
          aria-disabled={!ideLayoutAvailable}
        >
          IDE
        </button>

        {phoenixPackInstalled && onOpenPhoenix && (
          <button
            type="button"
            onClick={onOpenPhoenix}
            className={`${iconBtn} bg-indigo-600 hover:bg-indigo-500 text-white text-[10px] font-bold focus-visible:outline-indigo-400`}
            title="Phoenix TIM — sign in, browse, download"
            aria-label="Open Phoenix TIM browser"
          >
            PHX
          </button>
        )}
      </div>

      <ToolbarDivider layout={layout} />

      <div className={groupClass} aria-label="Collaboration and agents">
        <button
          type="button"
          onClick={onToggleTaskManagement}
          className={`${iconBtn} ${
            taskManagementOpen ? 'bg-violet-600 hover:bg-violet-700' : 'bg-violet-700/80 hover:bg-violet-700'
          } text-white`}
          title={`Task management (${formatChord('mod+shift+t')})`}
          aria-label="Open task management"
          aria-pressed={taskManagementOpen}
        >
          <TaskManagementIcon className="w-3.5 h-3.5" />
        </button>

        <button
          type="button"
          onClick={onNewRunbook}
          className={`${iconBtn} bg-slate-600 hover:bg-slate-500 text-white text-[10px] font-bold`}
          title={`New runbook (${formatChord('mod+shift+r')})`}
          aria-label="Create new runbook"
        >
          RB
        </button>

        <button
          type="button"
          onClick={onOpenMyAgents}
          className={`${iconBtn} bg-slack-accent hover:bg-slack-accentHover text-white relative focus-visible:outline-slack-accent`}
          title={`My agents (${formatChord('mod+shift+a')})`}
          aria-label="Open my agents"
        >
          <MyAgentsIcon className="w-3.5 h-3.5" />
          {totalAgentsCount > 0 && (
            <span className="absolute -bottom-0.5 -right-0.5 bg-white text-slack-accent text-[10px] font-bold rounded-full h-4 w-4 flex items-center justify-center leading-none">
              {totalAgentsCount}
            </span>
          )}
        </button>
      </div>

      <ToolbarDivider layout={layout} />

      <div className={groupClass} aria-label="Developer tools">
        {devPackEnabled && (
          <button
            type="button"
            onClick={onOpenProblems}
            className={`${iconBtn} bg-purple-600 hover:bg-purple-500 text-white focus-visible:outline-purple-400`}
            title={`Problems (${formatChord('mod+shift+d')})`}
            aria-label="Open problems panel"
          >
            <span className="text-xs font-bold">!</span>
          </button>
        )}

        {devPackEnabled && (
          <button
            type="button"
            onClick={onToggleGitModal}
            className={`${iconBtn} focus-visible:outline-orange-400 ${
              gitModalOpen
                ? 'bg-orange-500 ring-2 ring-orange-300/60 text-white'
                : 'bg-orange-600 hover:bg-orange-500 text-white'
            }`}
            title={`Git (${formatChord('mod+shift+g')})`}
            aria-label={gitModalOpen ? 'Close git panel' : 'Open git panel'}
            aria-pressed={gitModalOpen}
          >
            <GitIcon className="w-3.5 h-3.5" />
          </button>
        )}

        <button
          type="button"
          onClick={() => useTerminalStore.getState().togglePanel()}
          className={`${iconBtn} bg-gray-600 hover:bg-gray-700 text-white relative focus-visible:outline-gray-400 ${
            approvalCount > 0 ? 'ring-2 ring-amber-400/70' : ''
          }`}
          title={
            approvalCount > 0
              ? `${approvalCount} command/tool approval${approvalCount === 1 ? '' : 's'} waiting (${formatChord('mod+j')})`
              : `Terminal (${formatChord('mod+j')})`
          }
          aria-label="Toggle terminal panel"
        >
          <TerminalIcon className="w-3.5 h-3.5" />
          {approvalCount > 0 && (
            <span className="absolute -bottom-0.5 -right-0.5 bg-amber-500 text-black text-[10px] font-bold rounded-full h-4 w-4 flex items-center justify-center leading-none">
              {approvalCount}
            </span>
          )}
        </button>
      </div>

      <ToolbarDivider layout={layout} />

      <div className={groupClass} aria-label="Models and account">
        {memoryMonitorEnabled && (
          <MemoryMonitorChip
            layout={layout}
            serverAddr={serverAddr}
            onOpenModelLibrary={onOpenModelLibrary}
            onOpenSettings={onOpenSettings}
          />
        )}

        <OllamaRuntimeChip
          layout={layout}
          serverAddr={serverAddr}
          onOpenModelLibrary={onOpenModelLibrary}
          onOpenSettings={onOpenSettings}
        />

        <button
          type="button"
          onClick={onOpenDomainPacks}
          className={`${iconBtn} bg-teal-600 hover:bg-teal-500 text-white relative focus-visible:outline-teal-300`}
          title={`Domain packs (${formatChord('mod+shift+k')})`}
          aria-label="Open domain pack store"
        >
          <DomainPacksIcon className="w-3.5 h-3.5" />
          {enabledPackCount > 0 && (
            <span className="absolute -bottom-0.5 -right-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-white text-[10px] font-bold leading-none text-teal-700">
              {enabledPackCount}
            </span>
          )}
        </button>

        <button
          type="button"
          onClick={onOpenModelLibrary}
          className={`${iconBtn} bg-amber-600 hover:bg-amber-500 text-white focus-visible:outline-amber-300`}
          title={`Model library (${formatChord('mod+shift+m')})`}
          aria-label="Open model library"
        >
          <ModelLibraryIcon className="w-3.5 h-3.5" />
        </button>

        {onOpenSettings && (
          <button
            type="button"
            onClick={() => onOpenSettings()}
            className={`${iconBtn} text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover focus-visible:outline-slack-accent`}
            title={`Settings (${formatChord('mod+,')})`}
            aria-label="Open settings"
          >
            <SettingsIcon className="w-3.5 h-3.5" />
          </button>
        )}

        {onLogout && (
          <button
            type="button"
            onClick={onLogout}
            className={`${iconBtn} text-slack-textMuted hover:text-red-500 hover:bg-red-500/10 focus-visible:outline-red-400`}
            title="Logout"
            aria-label="Log out"
          >
            <LogoutIcon className="w-3.5 h-3.5" />
          </button>
        )}
      </div>

      {!isVertical && (
        <>
          <ToolbarDivider layout={layout} />
          <div className="text-xs text-slack-textMuted hidden xl:block">
            <span className="font-medium text-slack-text">{username}</span>
            <span className="mx-1">•</span>
            <span>{serverAddr}</span>
          </div>
        </>
      )}

      {isVertical && (
        <div className="mt-2 px-1 text-[10px] leading-tight text-center text-slack-textMuted max-w-[3.5rem] break-all">
          <span className="font-medium text-slack-text block truncate w-full" title={username}>
            {username}
          </span>
          <span className="block truncate w-full" title={serverAddr}>
            {serverAddr}
          </span>
        </div>
      )}
    </div>
  );
}
