import { useCallback, useEffect, useState } from 'react';
import { shallow } from 'zustand/shallow';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useToastStore } from '../stores/toastStore';
import { GitDiffView } from './GitDiffView';

export interface GitStatusResponse {
  branch: string;
  clean: boolean;
  staged: string[];
  unstaged: string[];
  untracked: string[];
}

export interface GitModalProps {
  isOpen: boolean;
  onClose: () => void;
}

function normalizeGitStatus(raw: unknown): GitStatusResponse | null {
  if (!raw || typeof raw !== 'object') return null;
  const o = raw as Record<string, unknown>;
  const arr = (v: unknown): string[] =>
    Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : [];
  return {
    branch: typeof o.branch === 'string' ? o.branch : '',
    clean: Boolean(o.clean),
    staged: arr(o.staged),
    unstaged: arr(o.unstaged),
    untracked: arr(o.untracked),
  };
}

/** Git SCM modal (Software development pack). */
export function GitModal({ isOpen, onClose }: GitModalProps) {
  const { activeWorkspaceId, workspaces } = useFileExplorerStore(
    (s) => ({
      activeWorkspaceId: s.activeWorkspaceId,
      workspaces: s.workspaces,
    }),
    shallow
  );
  const active = workspaces.find((w) => w.id === activeWorkspaceId) ?? workspaces[0];
  const { addToast } = useToastStore();
  const [status, setStatus] = useState<GitStatusResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [commitMsg, setCommitMsg] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [diffView, setDiffView] = useState<{
    path: string;
    staged: boolean;
    original: string;
    modified: string;
  } | null>(null);

  const refresh = useCallback(async () => {
    if (!active?.id) return;
    setLoading(true);
    setError(null);
    try {
      const api = new ChatAPI(getHubBaseURL());
      const raw = await api.getGitStatus(active.id);
      setStatus(normalizeGitStatus(raw));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setStatus(null);
    } finally {
      setLoading(false);
    }
  }, [active?.id]);

  useEffect(() => {
    if (!isOpen) return;
    void refresh();
  }, [isOpen, refresh]);

  const run = async (label: string, fn: () => Promise<void>) => {
    try {
      await fn();
      addToast({ type: 'success', title: label, message: 'Done' });
      await refresh();
    } catch (e) {
      addToast({
        type: 'error',
        title: label,
        message: e instanceof Error ? e.message : String(e),
      });
    }
  };

  const viewDiff = async (path: string, staged: boolean) => {
    if (!active?.id) return;
    setLoading(true);
    try {
      const api = new ChatAPI(getHubBaseURL());
      const sides = await api.getGitFileSides(active.id, path, staged);
      setDiffView({ path, staged, original: sides.original, modified: sides.modified });
    } catch (e) {
      addToast({
        type: 'error',
        title: 'Diff',
        message: e instanceof Error ? e.message : String(e),
      });
    } finally {
      setLoading(false);
    }
  };

  const FileRow = ({
    path,
    kind,
  }: {
    path: string;
    kind: 'staged' | 'unstaged' | 'untracked';
  }) => (
    <li className="flex items-center gap-1 py-0.5 group">
      <span className="font-mono text-xs text-slack-text truncate flex-1" title={path}>
        {path}
      </span>
      {kind === 'unstaged' && (
        <button
          type="button"
          className="text-[10px] px-1 rounded text-emerald-400 hover:bg-slack-bgHover"
          onClick={() => void run('Stage', () => new ChatAPI(getHubBaseURL()).gitAdd(active!.id, [path]))}
        >
          Stage
        </button>
      )}
      {kind === 'untracked' && (
        <button
          type="button"
          className="text-[10px] px-1 rounded text-emerald-400 hover:bg-slack-bgHover"
          onClick={() => void run('Stage', () => new ChatAPI(getHubBaseURL()).gitAdd(active!.id, [path]))}
        >
          Stage
        </button>
      )}
      {kind === 'staged' && (
        <button
          type="button"
          className="text-[10px] px-1 rounded text-amber-400 hover:bg-slack-bgHover"
          onClick={() => void run('Unstage', () => new ChatAPI(getHubBaseURL()).gitReset(active!.id, [path]))}
        >
          Unstage
        </button>
      )}
      <button
        type="button"
        className="text-[10px] px-1 rounded text-sky-400 hover:bg-slack-bgHover"
        onClick={() => void viewDiff(path, kind === 'staged')}
      >
        Diff
      </button>
    </li>
  );

  if (!isOpen) return null;

  const body = () => {
    if (!active) {
      return <p className="text-sm text-slack-textMuted">Add a workspace in the file explorer first.</p>;
    }
    if (!active.is_git_repo) {
      return (
        <p className="text-sm text-slack-textMuted">
          <span className="text-slack-text font-medium">{active.name}</span> is not a git repository.
        </p>
      );
    }
    return (
      <div className="space-y-4 text-sm text-slack-text">
        {error && (
          <p className="text-red-400 rounded-md border border-red-800/50 bg-red-950/40 px-3 py-2">{error}</p>
        )}
        {loading && !status && (
          <p className="text-slack-textMuted">Loading git status…</p>
        )}
        {diffView ? (
          <GitDiffView
            path={diffView.path}
            original={diffView.original}
            modified={diffView.modified}
            staged={diffView.staged}
            onBack={() => setDiffView(null)}
          />
        ) : (
          status && (
            <div className="space-y-3 rounded-lg border border-slack-border bg-slack-bgHover/40 p-3">
              <p>
                Branch:{' '}
                <code className="font-mono text-xs text-amber-200 bg-slack-bg px-1.5 py-0.5 rounded">
                  {status.branch || '—'}
                </code>
              </p>
              <p className={status.clean ? 'text-emerald-400' : 'text-amber-400'}>
                {status.clean ? 'Working tree clean' : 'Changes present'}
              </p>
              <div className="flex gap-2 flex-wrap">
                {(status.unstaged?.length ?? 0) + (status.untracked?.length ?? 0) > 0 && (
                  <button
                    type="button"
                    className="text-xs px-2 py-0.5 rounded border border-slack-border text-emerald-400 hover:bg-slack-bgHover"
                    onClick={() =>
                      void run('Stage all', () =>
                        new ChatAPI(getHubBaseURL()).gitAdd(active!.id, [
                          ...(status.unstaged ?? []),
                          ...(status.untracked ?? []),
                        ])
                      )
                    }
                  >
                    Stage all
                  </button>
                )}
                {(status.staged?.length ?? 0) > 0 && (
                  <button
                    type="button"
                    className="text-xs px-2 py-0.5 rounded border border-slack-border text-amber-400 hover:bg-slack-bgHover"
                    onClick={() =>
                      void run('Unstage all', () =>
                        new ChatAPI(getHubBaseURL()).gitReset(active!.id, status.staged ?? [])
                      )
                    }
                  >
                    Unstage all
                  </button>
                )}
              </div>
              {(status.staged?.length ?? 0) > 0 && (
                <div>
                  <p className="text-slack-textMuted text-xs uppercase tracking-wide mb-1">Staged</p>
                  <ul className="space-y-0.5 max-h-28 overflow-y-auto">
                    {status.staged.map((f) => (
                      <FileRow key={`s-${f}`} path={f} kind="staged" />
                    ))}
                  </ul>
                </div>
              )}
              {(status.unstaged?.length ?? 0) > 0 && (
                <div>
                  <p className="text-slack-textMuted text-xs uppercase tracking-wide mb-1">Unstaged</p>
                  <ul className="space-y-0.5 max-h-28 overflow-y-auto">
                    {status.unstaged.map((f) => (
                      <FileRow key={`u-${f}`} path={f} kind="unstaged" />
                    ))}
                  </ul>
                </div>
              )}
              {(status.untracked?.length ?? 0) > 0 && (
                <div>
                  <p className="text-slack-textMuted text-xs uppercase tracking-wide mb-1">Untracked</p>
                  <ul className="space-y-0.5 max-h-28 overflow-y-auto">
                    {status.untracked.map((f) => (
                      <FileRow key={`t-${f}`} path={f} kind="untracked" />
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )
        )}
        <div className="space-y-2">
          <label className="block text-xs text-slack-textMuted" htmlFor="git-commit-msg">
            Commit message
          </label>
          <input
            id="git-commit-msg"
            type="text"
            value={commitMsg}
            onChange={(e) => setCommitMsg(e.target.value)}
            placeholder="Describe your changes"
            className="w-full px-3 py-2 rounded-md border border-slack-border bg-slack-bgHover text-slack-text placeholder:text-slack-textMuted focus:outline-none focus:ring-2 focus:ring-slack-accent/50"
          />
          <button
            type="button"
            disabled={!commitMsg.trim() || loading}
            onClick={() =>
              void run('Commit', async () => {
                const api = new ChatAPI(getHubBaseURL());
                await api.commitChanges(active.id, commitMsg.trim());
                setCommitMsg('');
              })
            }
            className="w-full py-2 rounded-md bg-slack-accent hover:bg-slack-accentHover text-white text-sm font-medium disabled:opacity-50"
          >
            Commit
          </button>
          <div className="flex gap-2">
            <button
              type="button"
              disabled={loading}
              onClick={() =>
                void run('Pull', async () => {
                  const api = new ChatAPI(getHubBaseURL());
                  await api.pullChanges(active.id);
                })
              }
              className="flex-1 py-2 rounded-md border border-slack-border bg-slack-bgHover text-slack-text hover:bg-slack-border/80 text-sm disabled:opacity-50"
            >
              Pull
            </button>
            <button
              type="button"
              disabled={loading}
              onClick={() =>
                void run('Push', async () => {
                  const api = new ChatAPI(getHubBaseURL());
                  await api.pushChanges(active.id);
                })
              }
              className="flex-1 py-2 rounded-md border border-slack-border bg-slack-bgHover text-slack-text hover:bg-slack-border/80 text-sm disabled:opacity-50"
            >
              Push
            </button>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div
      className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto py-6 px-4"
      role="presentation"
    >
      <div className="fixed inset-0 bg-black/60" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="nj-git-modal-title"
        className={`relative z-10 flex w-full flex-col overflow-hidden rounded-xl border border-slack-border bg-slack-bg shadow-2xl max-h-[min(85vh,640px)] ${
          diffView ? 'max-w-4xl' : 'max-w-md'
        }`}
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slack-border px-4 py-3">
          <div className="min-w-0">
            <h2 id="nj-git-modal-title" className="text-lg font-semibold text-slack-text">
              Git
            </h2>
            {active && (
              <p className="text-xs text-slack-textMuted truncate mt-0.5" title={active.path}>
                {active.name}
              </p>
            )}
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {active?.is_git_repo && (
              <button
                type="button"
                onClick={() => void refresh()}
                disabled={loading}
                className="rounded px-2 py-1 text-sm text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover disabled:opacity-50"
                title="Refresh status"
              >
                Refresh
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              className="rounded px-2 py-1 text-sm text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover"
              aria-label="Close git panel"
            >
              Close
            </button>
          </div>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-4">{body()}</div>
      </div>
    </div>
  );
}

/** @deprecated Use GitModal */
export const GitPanel = GitModal;
