import { CommandForm } from './CommandForm';
import type { ChatAPI } from '../api/chatAPI';
import type {
  AgentInfo,
  Channel,
  Collaboration,
  CommandDefinition,
  FileChange,
} from '../types/protocol';

export interface StartCollaborationModalProps {
  isOpen: boolean;
  command: CommandDefinition | null;
  agents: AgentInfo[];
  channels?: Channel[];
  activeChannel?: string;
  collaborations?: Collaboration[];
  pendingChanges?: FileChange[];
  api?: ChatAPI;
  onClose: () => void;
  onSubmit: (commandString: string, metadata?: Record<string, unknown>) => void;
}

/**
 * Guided start surface for multi-agent collaborations.
 * Reuses CommandForm so slash and UI stay in sync.
 */
export function StartCollaborationModal({
  isOpen,
  command,
  agents,
  channels = [],
  activeChannel = '',
  collaborations = [],
  pendingChanges = [],
  api,
  onClose,
  onSubmit,
}: StartCollaborationModalProps) {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4"
      role="presentation"
      data-testid="start-collaboration-modal"
    >
      <div className="absolute inset-0 bg-black/60" onClick={onClose} role="presentation" aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="start-collaboration-title"
        className="relative z-10 flex w-full max-w-lg min-h-0 flex-col overflow-hidden rounded-xl border border-slack-border bg-slack-bg shadow-2xl"
        style={{ maxHeight: 'min(90dvh, calc(100dvh - 1.5rem))' }}
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slack-border px-4 py-3">
          <div>
            <h2 id="start-collaboration-title" className="text-lg font-semibold text-slack-text">
              Start collaboration
            </h2>
            <p className="text-xs text-slack-textMuted mt-0.5">
              Pick 1–3 agents and a goal. Agents plan together; you approve before execution.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close start collaboration"
            className="text-slack-textMuted hover:text-slack-text px-2 py-1 rounded hover:bg-slack-bgHover"
          >
            ✕
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto">
          {!command ? (
            <p className="px-4 py-8 text-sm text-slack-textMuted">
              Collaboration command is unavailable. Open the command palette and run{' '}
              <code className="font-mono">/collaborate</code>.
            </p>
          ) : (
            <CommandForm
              command={command}
              agents={agents}
              channels={channels}
              activeChannel={activeChannel}
              collaborations={collaborations}
              pendingChanges={pendingChanges}
              api={api}
              onSubmit={(cmd, metadata) => {
                onSubmit(cmd, metadata);
                onClose();
              }}
              onBack={onClose}
            />
          )}
        </div>
      </div>
    </div>
  );
}

/** Resolve the /collaborate command definition from the hub command list. */
export function findCollaborateCommand(
  commands: CommandDefinition[]
): CommandDefinition | null {
  return (
    commands.find((c) => c.name === '/collaborate') ??
    commands.find((c) => c.name.replace(/^\//, '') === 'collaborate') ??
    null
  );
}
