export type WorkspaceAddMode = 'create' | 'link' | 'remote' | 'devcontainer';

export interface FileExplorerAddWorkspaceModalProps {
  open: boolean;
  workspaceAddMode: WorkspaceAddMode;
  setWorkspaceAddMode: (mode: WorkspaceAddMode) => void;
  newWorkspaceName: string;
  setNewWorkspaceName: (v: string) => void;
  newWorkspaceParentPath: string;
  setNewWorkspaceParentPath: (v: string) => void;
  newWorkspacePath: string;
  setNewWorkspacePath: (v: string) => void;
  remoteHost: string;
  setRemoteHost: (v: string) => void;
  remoteUser: string;
  setRemoteUser: (v: string) => void;
  remotePath: string;
  setRemotePath: (v: string) => void;
  sidecarUrl: string;
  setSidecarUrl: (v: string) => void;
  sidecarToken: string;
  setSidecarToken: (v: string) => void;
  devcontainerRepoPath: string;
  setDevcontainerRepoPath: (v: string) => void;
  forDevcontainerPlanClear: () => void;
  devcontainerPlan: {
    container_name?: string;
    image?: string;
    workspace_folder?: string;
    sidecar_port?: number;
  } | null;
  devcontainerPlanLoading: boolean;
  onBrowseDirectory: (target: 'link' | 'parent' | 'devcontainer') => void;
  onLoadDevcontainerPlan: () => void;
  onAddWorkspace: () => void;
  onCancel: () => void;
}

export function FileExplorerAddWorkspaceModal(props: FileExplorerAddWorkspaceModalProps) {
  if (!props.open) return null;
  const {
    workspaceAddMode,
    setWorkspaceAddMode,
    newWorkspaceName,
    setNewWorkspaceName,
    newWorkspaceParentPath,
    setNewWorkspaceParentPath,
    newWorkspacePath,
    setNewWorkspacePath,
    remoteHost,
    setRemoteHost,
    remoteUser,
    setRemoteUser,
    remotePath,
    setRemotePath,
    sidecarUrl,
    setSidecarUrl,
    sidecarToken,
    setSidecarToken,
    devcontainerRepoPath,
    setDevcontainerRepoPath,
    forDevcontainerPlanClear,
    devcontainerPlan,
    devcontainerPlanLoading,
    onBrowseDirectory,
    onLoadDevcontainerPlan,
    onAddWorkspace,
    onCancel,
  } = props;

  return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-slack-bg border border-slack-border rounded p-6 w-[28rem] max-w-[95vw]">
            <h3 className="text-lg font-bold text-slack-text mb-4">Add Workspace</h3>
            <div className="flex rounded-md border border-slack-border overflow-hidden text-xs mb-4" role="tablist">
              <button
                type="button"
                role="tab"
                aria-selected={workspaceAddMode === 'create'}
                onClick={() => setWorkspaceAddMode('create')}
                className={`flex-1 px-3 py-1.5 font-medium ${
                  workspaceAddMode === 'create'
                    ? 'bg-slack-accent text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Create new
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={workspaceAddMode === 'link'}
                onClick={() => setWorkspaceAddMode('link')}
                className={`flex-1 px-3 py-1.5 font-medium ${
                  workspaceAddMode === 'link'
                    ? 'bg-slack-accent text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Link existing
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={workspaceAddMode === 'remote'}
                onClick={() => setWorkspaceAddMode('remote')}
                className={`flex-1 px-3 py-1.5 font-medium ${
                  workspaceAddMode === 'remote'
                    ? 'bg-slack-accent text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Remote SSH
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={workspaceAddMode === 'devcontainer'}
                onClick={() => setWorkspaceAddMode('devcontainer')}
                className={`flex-1 px-3 py-1.5 font-medium ${
                  workspaceAddMode === 'devcontainer'
                    ? 'bg-slack-accent text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Dev container
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slack-text mb-1">
                  Name
                </label>
                <input
                  type="text"
                  value={newWorkspaceName}
                  onChange={(e) => setNewWorkspaceName(e.target.value)}
                  className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text focus:outline-none focus:border-slack-accent"
                  placeholder="Phoenix run 2026-06-04"
                />
              </div>
              {workspaceAddMode === 'create' ? (
                <div>
                  <label className="block text-sm font-medium text-slack-text mb-1">
                    Location (optional)
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={newWorkspaceParentPath}
                      onChange={(e) => setNewWorkspaceParentPath(e.target.value)}
                      className="flex-1 px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text focus:outline-none focus:border-slack-accent font-mono text-xs"
                      placeholder="~/.neural-junkie/workspaces (default)"
                    />
                    <button
                      type="button"
                      onClick={() => void onBrowseDirectory('parent')}
                      className="px-3 py-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
                      title="Browse for parent folder"
                    >
                      📁
                    </button>
                  </div>
                  <p className="mt-1 text-[11px] text-slack-textMuted">
                    Creates a new folder from the name. Default root:{' '}
                    <code className="font-mono">~/.neural-junkie/workspaces/</code>
                  </p>
                </div>
              ) : workspaceAddMode === 'remote' ? (
                <>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="block text-sm font-medium text-slack-text mb-1">Host</label>
                      <input
                        type="text"
                        value={remoteHost}
                        onChange={(e) => setRemoteHost(e.target.value)}
                        className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text font-mono text-xs"
                        placeholder="ec2-xxx.compute.amazonaws.com"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slack-text mb-1">User</label>
                      <input
                        type="text"
                        value={remoteUser}
                        onChange={(e) => setRemoteUser(e.target.value)}
                        className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text font-mono text-xs"
                        placeholder="ec2-user"
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-1">Remote path</label>
                    <input
                      type="text"
                      value={remotePath}
                      onChange={(e) => setRemotePath(e.target.value)}
                      className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text font-mono text-xs"
                      placeholder="/home/ec2-user/myproject"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-1">Sidecar URL</label>
                    <input
                      type="text"
                      value={sidecarUrl}
                      onChange={(e) => setSidecarUrl(e.target.value)}
                      className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text font-mono text-xs"
                      placeholder="http://127.0.0.1:19876"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-1">Sidecar token</label>
                    <input
                      type="password"
                      value={sidecarToken}
                      onChange={(e) => setSidecarToken(e.target.value)}
                      className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text font-mono text-xs"
                      placeholder="Bearer token from nj-remote"
                    />
                  </div>
                  <p className="text-[11px] text-slack-textMuted">
                    Tunnel:{' '}
                    <code className="font-mono">
                      ssh -L 19876:127.0.0.1:19876 {remoteUser || 'user'}@{remoteHost || 'host'}
                    </code>
                  </p>
                </>
              ) : workspaceAddMode === 'devcontainer' ? (
                <>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-1">Local repo path</label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={devcontainerRepoPath}
                        onChange={(e) => {
                          setDevcontainerRepoPath(e.target.value);
                          forDevcontainerPlanClear();
                        }}
                        className="flex-1 px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text font-mono text-xs"
                        placeholder="/Users/you/myproject"
                      />
                      <button
                        type="button"
                        onClick={() => void onBrowseDirectory('devcontainer')}
                        className="px-3 py-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
                        title="Browse for repo with .devcontainer"
                      >
                        📁
                      </button>
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => void onLoadDevcontainerPlan()}
                    disabled={!devcontainerRepoPath.trim() || devcontainerPlanLoading}
                    className="w-full px-3 py-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white text-xs rounded transition-colors disabled:opacity-50"
                  >
                    {devcontainerPlanLoading ? 'Loading plan…' : 'Load devcontainer plan'}
                  </button>
                  {devcontainerPlan && (
                    <div className="text-[11px] text-slack-textMuted space-y-1 border border-slack-border rounded p-2">
                      {devcontainerPlan.container_name && (
                        <p>
                          Container: <code className="font-mono">{devcontainerPlan.container_name}</code>
                        </p>
                      )}
                      {devcontainerPlan.image && (
                        <p>
                          Image: <code className="font-mono">{devcontainerPlan.image}</code>
                        </p>
                      )}
                      {devcontainerPlan.workspace_folder && (
                        <p>
                          Workspace: <code className="font-mono">{devcontainerPlan.workspace_folder}</code>
                        </p>
                      )}
                      {devcontainerPlan.sidecar_port != null && (
                        <p>
                          Sidecar port: <code className="font-mono">{devcontainerPlan.sidecar_port}</code>
                        </p>
                      )}
                    </div>
                  )}
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-1">Sidecar URL</label>
                    <input
                      type="text"
                      value={sidecarUrl}
                      onChange={(e) => setSidecarUrl(e.target.value)}
                      className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text font-mono text-xs"
                      placeholder="http://127.0.0.1:19876"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-1">Sidecar token</label>
                    <input
                      type="password"
                      value={sidecarToken}
                      onChange={(e) => setSidecarToken(e.target.value)}
                      className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text font-mono text-xs"
                      placeholder="Bearer token from nj-remote in container"
                    />
                  </div>
                  <p className="text-[11px] text-slack-textMuted">
                    Run <code className="font-mono">devcontainer up</code>, then{' '}
                    <code className="font-mono">nj-remote -root $WORKSPACE -addr :19876</code> inside the container.
                  </p>
                </>
              ) : (
                <div>
                  <label className="block text-sm font-medium text-slack-text mb-1">
                    Path
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={newWorkspacePath}
                      onChange={(e) => setNewWorkspacePath(e.target.value)}
                      className="flex-1 px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text focus:outline-none focus:border-slack-accent font-mono text-xs"
                      placeholder="/path/to/existing/folder"
                    />
                    <button
                      type="button"
                      onClick={() => void onBrowseDirectory('link')}
                      className="px-3 py-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
                      title="Browse for directory"
                    >
                      📁
                    </button>
                  </div>
                </div>
              )}
            </div>
            <div className="flex gap-2 mt-6">
              <button
                type="button"
                onClick={() => void onAddWorkspace()}
                disabled={
                  !newWorkspaceName.trim() ||
                  (workspaceAddMode === 'link' && !newWorkspacePath.trim()) ||
                  (workspaceAddMode === 'remote' &&
                    (!remoteHost.trim() || !remotePath.trim() || !sidecarUrl.trim())) ||
                  (workspaceAddMode === 'devcontainer' &&
                    (!devcontainerRepoPath.trim() || !sidecarUrl.trim()))
                }
                className="px-4 py-2 bg-slack-accent hover:bg-slack-accentHover text-white text-sm rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {workspaceAddMode === 'create'
                  ? 'Create'
                  : workspaceAddMode === 'remote' || workspaceAddMode === 'devcontainer'
                    ? 'Connect'
                    : 'Add'}
              </button>
              <button
                type="button"
                onClick={onCancel}
                className="px-4 py-2 bg-slack-bgHover text-slack-text text-sm rounded transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>

  );
}
