import { useFileExplorerStore } from '../stores/fileExplorerStore';

export const REPO_AGENT_WORKSPACE_ACTION = 'select_repo_workspace';

export interface RepoAgentWorkspaceAction {
  type: typeof REPO_AGENT_WORKSPACE_ACTION;
  path: string;
  name?: string;
}

/** Normalize paths for comparison (trailing slashes, backslashes). */
export function normalizeWorkspacePath(path: string): string {
  const trimmed = path.trim().replace(/\\/g, '/');
  if (trimmed === '/') return trimmed;
  return trimmed.replace(/\/+$/, '') || trimmed;
}

/** Parse `/create-repo-agent <path> [name] [provider] [model]`. */
export function parseCreateRepoAgentCommand(command: string): { repoPath: string; agentName?: string } | null {
  const trimmed = command.trim();
  if (!trimmed.toLowerCase().startsWith('/create-repo-agent')) {
    return null;
  }
  const rest = trimmed.slice('/create-repo-agent'.length).trim();
  if (!rest) return null;

  const parts = rest.split(/\s+/);
  const repoPath = parts[0];
  if (!repoPath) return null;

  const providers = new Set(['ollama', 'claude', 'lmstudio', 'huggingface', 'hf']);
  let agentName: string | undefined;
  if (parts.length >= 2 && !providers.has(parts[1].toLowerCase())) {
    agentName = parts[1];
  }

  return { repoPath, agentName };
}

export function isRepoAgentWorkspaceAction(meta: unknown): meta is RepoAgentWorkspaceAction {
  if (!meta || typeof meta !== 'object') return false;
  const m = meta as Record<string, unknown>;
  return m.type === REPO_AGENT_WORKSPACE_ACTION && typeof m.path === 'string' && m.path.trim() !== '';
}

const linkedRepoPaths = new Set<string>();

/**
 * Ensures the repo path exists as a desktop workspace and selects it (idempotent per normalized path per session).
 */
export async function ensureRepoAgentWorkspace(
  repoPath: string,
  options?: { preferredName?: string }
): Promise<string | null> {
  const normalized = normalizeWorkspacePath(repoPath);
  if (!normalized) return null;

  if (linkedRepoPaths.has(normalized)) {
    const existing = useFileExplorerStore
      .getState()
      .workspaces.find((w) => normalizeWorkspacePath(w.path) === normalized);
    if (existing) {
      useFileExplorerStore.getState().setActiveWorkspace(existing.id);
      return existing.id;
    }
    linkedRepoPaths.delete(normalized);
  }

  try {
    const store = useFileExplorerStore.getState();
    await store.loadWorkspaces();

    let workspace = useFileExplorerStore
      .getState()
      .workspaces.find((w) => normalizeWorkspacePath(w.path) === normalized);

    if (!workspace) {
      const name =
        options?.preferredName?.trim() ||
        normalized.split('/').filter(Boolean).pop() ||
        'Repository';
      workspace = await store.addWorkspace(name, repoPath);
    } else {
      store.setActiveWorkspace(workspace.id);
    }

    await store.loadFiles(workspace.id, '/');
    linkedRepoPaths.add(normalized);
    return workspace.id;
  } catch (e) {
    console.error('[ensureRepoAgentWorkspace]', e);
    return null;
  }
}
