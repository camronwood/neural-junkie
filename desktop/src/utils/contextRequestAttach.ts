import type { ChatAPI } from '../api/chatAPI';
import type { ContextScope } from '../constants/promptMetadata';
import type { FileNode } from '../stores/fileExplorerStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import type { OpenFileContext, WorkspaceContext } from './workspaceContext';
import { scopeFromContextRequest } from './inferContextScope';
import { trimWorkspaceContext } from './outboundChatMetadata';

export interface ContextRequestPayload {
  context_tier?: string;
  subject?: string;
  review_mode?: string;
  requested_categories?: string[];
  requested_capabilities?: string[];
  include_workspace_identity?: boolean;
  include_file_tree?: boolean;
  include_active_tab?: boolean;
  include_selection?: boolean;
  include_open_files?: boolean;
  include_document_bodies?: boolean;
  include_git_status?: boolean;
  include_diagnostics?: boolean;
}

const DOC_EXT_RE = /\.(md|markdown|txt|rst|adoc)$/i;
const DOC_NAME_RE = /^(readme|changelog|license|contributing|security)(\.|$)/i;
const PRIORITY_DIR_RE = /(^|\/)(internal|gilead-security|docs?|documentation)(\/|$)/i;
const MAX_DOC_BODIES = 12;
const MAX_DOC_BYTES = 12000;

function collectDocPaths(nodes: FileNode[], out: string[], depth = 0): void {
  if (depth > 8 || out.length >= MAX_DOC_BODIES * 3) return;
  for (const node of nodes) {
    if (node.is_dir) {
      if (node.children?.length) collectDocPaths(node.children, out, depth + 1);
      continue;
    }
    const path = (node.path ?? '').replace(/\\/g, '/');
    const base = path.split('/').pop() ?? '';
    if (DOC_EXT_RE.test(base) || DOC_NAME_RE.test(base)) {
      out.push(path);
    }
  }
}

function prioritizeDocPaths(paths: string[]): string[] {
  const scored = paths.map((p) => {
    let score = 0;
    if (PRIORITY_DIR_RE.test(p)) score += 10;
    if (/readme/i.test(p)) score += 5;
    if (/\.md$/i.test(p)) score += 2;
    return { p, score };
  });
  scored.sort((a, b) => b.score - a.score || a.p.localeCompare(b.p));
  return scored.slice(0, MAX_DOC_BODIES).map((s) => s.p);
}

/**
 * Loads bounded Markdown/docs bodies when the stamp asks for document review.
 */
export async function loadRequestedDocumentBodies(
  api: ChatAPI,
  workspaceId: string,
  nodes: FileNode[],
): Promise<OpenFileContext[]> {
  const candidates = prioritizeDocPaths((() => {
    const all: string[] = [];
    collectDocPaths(nodes, all);
    return all;
  })());
  const out: OpenFileContext[] = [];
  for (const path of candidates) {
    try {
      const content = await api.fetchFileContent(workspaceId, path);
      out.push({
        path,
        language: 'markdown',
        content: (content ?? '').substring(0, MAX_DOC_BYTES),
        is_active: false,
      });
    } catch {
      /* skip unreadable paths */
    }
  }
  return out;
}

export async function applyContextRequestToMetadata(options: {
  api: ChatAPI;
  metadata: Record<string, unknown>;
  message: string;
  contextRequest: ContextRequestPayload;
  prepareToken: string;
  fullWorkspace: WorkspaceContext;
  activeTabPath?: string;
}): Promise<Record<string, unknown>> {
  const {
    api,
    metadata,
    message,
    contextRequest,
    prepareToken,
    fullWorkspace,
    activeTabPath,
  } = options;
  const meta: Record<string, unknown> = { ...metadata, prepare_token: prepareToken };
  const scope: ContextScope = scopeFromContextRequest(contextRequest);
  meta.context_scope = scope;
  meta.context_scope_reason = `hub context_request tier=${contextRequest.context_tier ?? scope}`;

  if (scope === 'none' || !contextRequest.include_workspace_identity) {
    if (scope === 'none') {
      delete meta.workspace_context;
    }
    return meta;
  }

  let trimmed = trimWorkspaceContext(scope, fullWorkspace, message, activeTabPath);
  if (!trimmed) {
    return meta;
  }

  // Outline+document bodies: attach prioritized Markdown even when no editor tabs.
  if (contextRequest.include_document_bodies) {
    const { workspaces, activeWorkspaceId, fileTree } = useFileExplorerStore.getState();
    const ws =
      workspaces.find((w) => w.id === (fullWorkspace.workspace_id || activeWorkspaceId)) ??
      workspaces.find((w) => w.id === activeWorkspaceId) ??
      workspaces[0];
    if (ws) {
      const nodes = fileTree[ws.id] ?? [];
      const docs = await loadRequestedDocumentBodies(api, ws.id, nodes);
      const existing = trimmed.open_files ?? [];
      const seen = new Set(existing.map((f) => f.path));
      const merged = [...existing];
      for (const doc of docs) {
        if (!seen.has(doc.path)) {
          merged.push(doc);
          seen.add(doc.path);
        }
      }
      // When subject is workspace documents, prefer document bodies over empty focus trim.
      if (merged.length > 0) {
        trimmed = {
          ...trimmed,
          file_tree: contextRequest.include_file_tree ? fullWorkspace.file_tree : trimmed.file_tree,
          open_files: merged,
        };
      }
    }
  }

  if (!contextRequest.include_file_tree) {
    trimmed = { ...trimmed, file_tree: '' };
  }
  if (!contextRequest.include_active_tab && !contextRequest.include_open_files && !contextRequest.include_document_bodies) {
    trimmed = { ...trimmed, open_files: [] };
  }

  meta.workspace_context = trimmed;
  meta.context_plan_subject = contextRequest.subject;
  meta.context_plan_review_mode = contextRequest.review_mode;
  return meta;
}
