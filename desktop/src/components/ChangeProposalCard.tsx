import { useEffect, useMemo, useState } from 'react';
import type {
  ChangeProposalCard as ChangeProposal,
  ChangeProposalStatus,
  Message,
} from '../types/protocol';
import { getChangeProposalCard } from '../types/protocol';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { useGitChangeStore } from '../stores/gitChangeStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useEditorStore } from '../stores/editorStore';
import { useSettingsStore } from '../stores/settingsStore';
import { useToastStore } from '../stores/toastStore';
import { getLanguageFromPath } from '../utils/editorLanguage';

const statusLabel: Record<ChangeProposalStatus, string> = {
  pending: 'Needs review',
  applying: 'Applying…',
  approved: 'Accepted',
  rejected: 'Rejected',
  stale: 'Stale',
  expired: 'Expired',
  failed: 'Failed',
};

function effectiveStatus(
  card: ChangeProposal,
  storeStatus: ChangeProposalStatus | undefined,
  busy: boolean,
  now: number,
): ChangeProposalStatus {
  if (busy) return 'applying';
  if (card.status !== 'pending') return card.status;
  if (storeStatus && storeStatus !== 'pending') return storeStatus;
  if (card.expires_at && Date.parse(card.expires_at) <= now) return 'expired';
  return 'pending';
}

function displayPath(card: ChangeProposal): string {
  if (card.operation === 'move') {
    return `${card.old_path || card.file_path || 'file'} → ${card.new_path || 'file'}`;
  }
  return card.file_path || card.paths?.join(', ') || 'Workspace';
}

function StatusBadge({ status }: { status: ChangeProposalStatus }) {
  const tone =
    status === 'approved'
      ? 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'
      : status === 'rejected' || status === 'failed'
        ? 'bg-red-500/15 text-red-300 border-red-500/30'
        : status === 'stale' || status === 'expired'
          ? 'bg-amber-500/15 text-amber-300 border-amber-500/30'
          : 'bg-blue-500/15 text-blue-300 border-blue-500/30';
  return (
    <span className={`rounded-full border px-2 py-0.5 text-[11px] font-medium ${tone}`}>
      {statusLabel[status]}
    </span>
  );
}

export function ChangeProposalMessageCard({ message }: { message: Message }) {
  const card = getChangeProposalCard(message);
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState('');
  const [now, setNow] = useState(() => Date.now());
  const fileChange = useFileChangeStore((state) =>
    card?.kind === 'file_change' ? state.changesById[card.id] : undefined,
  );
  const fileBusy = useFileChangeStore((state) =>
    card?.kind === 'file_change' ? state.busyById[card.id] === true : false,
  );
  const fileError = useFileChangeStore((state) =>
    card?.kind === 'file_change' ? state.errorsById[card.id] : '',
  );
  const gitChange = useGitChangeStore((state) =>
    card?.kind === 'git_change' ? state.changesById[card.id] : undefined,
  );
  const gitBusy = useGitChangeStore((state) =>
    card?.kind === 'git_change' ? state.busyById[card.id] === true : false,
  );
  const gitError = useGitChangeStore((state) =>
    card?.kind === 'git_change' ? state.errorsById[card.id] : '',
  );
  const addToast = useToastStore((state) => state.addToast);

  useEffect(() => {
    if (!card?.expires_at || card.status !== 'pending') return;
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, [card?.expires_at, card?.status]);

  const status = useMemo(() => {
    if (!card) return 'failed' as const;
    const storeStatus =
      card.kind === 'file_change' ? fileChange?.status : gitChange?.status;
    return effectiveStatus(card, storeStatus, fileBusy || gitBusy, now);
  }, [card, fileBusy, fileChange?.status, gitBusy, gitChange?.status, now]);

  if (!card) return null;

  const destructive =
    card.kind === 'file_change' && (card.operation === 'delete' || card.operation === 'move');
  const canAct = status === 'pending';
  const errorText = card.error || fileError || gitError;

  const approve = async () => {
    try {
      if (card.kind === 'file_change') {
        await useFileChangeStore.getState().approveChange(card.id);
      } else {
        await useGitChangeStore.getState().approveGitChange(card.id);
      }
    } catch (error) {
      addToast({
        type: 'error',
        title: 'Change not applied',
        message: error instanceof Error ? error.message : 'The proposal could not be applied.',
      });
    }
  };

  const reject = async () => {
    try {
      if (card.kind === 'file_change') {
        await useFileChangeStore.getState().rejectChange(card.id, reason || undefined);
      } else {
        await useGitChangeStore.getState().rejectGitChange(card.id, reason || undefined);
      }
      setRejecting(false);
    } catch (error) {
      addToast({
        type: 'error',
        title: 'Change not rejected',
        message: error instanceof Error ? error.message : 'The proposal could not be rejected.',
      });
    }
  };

  const reviewFile = async () => {
    try {
      await useFileChangeStore.getState().getFileDiff(card.id);
      useFileChangeStore.getState().selectChange(card.id);
      const change = useFileChangeStore.getState().changesById[card.id];
      if (!change) throw new Error('Change details are unavailable.');
      const absolutePath = change.file_path || change.old_path || change.new_path || '';
      const explorer = useFileExplorerStore.getState();
      const workspace =
        explorer.workspaces.find(
          (item) => absolutePath === item.path || absolutePath.startsWith(`${item.path}/`),
        ) ??
        explorer.workspaces.find((item) => item.id === explorer.activeWorkspaceId) ??
        explorer.workspaces[0];
      if (!workspace) throw new Error('Open the proposal workspace before reviewing this file.');
      const relativePath = absolutePath.startsWith(`${workspace.path}/`)
        ? absolutePath.slice(workspace.path.length + 1)
        : absolutePath;
      const content =
        change.operation === 'create'
          ? change.new_content || ''
          : change.old_content || change.new_content || '';
      useEditorStore
        .getState()
        .openFile(workspace.id, relativePath, content, getLanguageFromPath(relativePath));
      await useSettingsStore.getState().updateLayoutSettings({
        editorPanelVisible: true,
        filesPanelVisible: true,
      });
    } catch (error) {
      addToast({
        type: 'error',
        title: 'Review unavailable',
        message: error instanceof Error ? error.message : 'Could not open the proposed change.',
      });
    }
  };

  return (
    <section
      data-change-proposal-id={card.id}
      className={`rounded-lg border p-3 ${
        destructive
          ? 'border-amber-500/40 bg-amber-950/15'
          : 'border-slack-border bg-slack-bgHover/45'
      }`}
      aria-label={`${card.kind === 'file_change' ? 'File' : 'Git'} change proposal`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-semibold text-slack-text">
              {card.kind === 'file_change' ? 'Proposed file change' : 'Proposed Git operation'}
            </span>
            <span className="rounded bg-slack-bg px-1.5 py-0.5 text-[11px] uppercase text-slack-textMuted">
              {card.operation}
            </span>
            <StatusBadge status={status} />
          </div>
          <div className="mt-1 truncate font-mono text-xs text-slack-textMuted" title={displayPath(card)}>
            {displayPath(card)}
          </div>
          {card.message && <p className="mt-2 text-sm text-slack-text">{card.message}</p>}
          {card.paths && card.paths.length > 0 && (
            <div className="mt-2 text-xs text-slack-textMuted">
              {card.paths.length} path{card.paths.length === 1 ? '' : 's'}: {card.paths.join(', ')}
            </div>
          )}
          {(card.reason || errorText) && (
            <div className="mt-2 text-xs text-amber-300">{card.reason || errorText}</div>
          )}
        </div>
      </div>

      {canAct && (
        <div className="mt-3">
          {rejecting ? (
            <div className="flex flex-col gap-2 sm:flex-row">
              <input
                autoFocus
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') void reject();
                  if (event.key === 'Escape') setRejecting(false);
                }}
                placeholder="Optional rejection reason"
                className="min-w-0 flex-1 rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-xs text-slack-text"
                aria-label="Rejection reason"
              />
              <button
                type="button"
                onClick={() => void reject()}
                className="rounded bg-red-700 px-3 py-1.5 text-xs text-white hover:bg-red-600"
              >
                Confirm reject
              </button>
              <button
                type="button"
                onClick={() => setRejecting(false)}
                className="rounded border border-slack-border px-3 py-1.5 text-xs text-slack-textMuted"
              >
                Cancel
              </button>
            </div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {card.kind === 'file_change' && (
                <button
                  type="button"
                  onClick={() => void reviewFile()}
                  className="rounded border border-slack-border bg-slack-bg px-3 py-1.5 text-xs text-slack-text hover:bg-slack-border"
                >
                  Review in editor
                </button>
              )}
              <button
                type="button"
                onClick={() => void approve()}
                className={`rounded px-3 py-1.5 text-xs text-white ${
                  destructive ? 'bg-amber-700 hover:bg-amber-600' : 'bg-emerald-700 hover:bg-emerald-600'
                }`}
              >
                {destructive ? 'Accept destructive change' : 'Accept'}
              </button>
              <button
                type="button"
                onClick={() => setRejecting(true)}
                className="rounded border border-red-500/40 px-3 py-1.5 text-xs text-red-300 hover:bg-red-500/10"
              >
                Reject
              </button>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
