import { useEditorStore } from '../stores/editorStore';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useSettingsStore } from '../stores/settingsStore';
import {
  getEditApplyStreamMeta,
  isEditApplyStreamDelta,
  type Message,
} from '../types/protocol';
import { getLanguageFromPath } from './editorLanguage';
import { syncPendingChangeToEditor } from './syncPendingChangeToEditor';

type ActiveEditStream = {
  changeId: string;
  path: string;
  accumulated: string;
  cancelled: boolean;
  tabId: string | null;
};

const activeByChangeId = new Map<string, ActiveEditStream>();
const streamedChangeIds = new Set<string>();
const cancelledChangeIds = new Set<string>();

/** True if this change already received an edit-apply stream (new clients). */
export function wasEditApplyStreamed(changeId: string): boolean {
  return streamedChangeIds.has(changeId);
}

/** Cancel in-flight typewriter for a rejected change; revert buffer when possible. */
export function cancelEditApplyStream(changeId: string): void {
  const id = changeId.trim();
  if (!id) return;
  cancelledChangeIds.add(id);
  const active = activeByChangeId.get(id);
  if (!active) {
    return;
  }
  active.cancelled = true;
  activeByChangeId.delete(id);

  const pending = useFileChangeStore.getState().pendingChanges.find((c) => c.id === id);
  const revert =
    pending?.operation === 'edit'
      ? pending.old_content ?? ''
      : pending?.operation === 'create'
        ? ''
        : null;
  if (revert !== null && active.tabId) {
    useEditorStore.getState().applyExternalTabContent(active.tabId, revert);
  }
}

/** Test helper: clear stream tracking state. */
export function resetEditApplyStreamStateForTests(): void {
  activeByChangeId.clear();
  streamedChangeIds.clear();
  cancelledChangeIds.clear();
}

function resolveWorkspaceForPath(absoluteOrRel: string): {
  workspaceId: string;
  relativePath: string;
} | null {
  const absolutePath = absoluteOrRel.replace(/\\/g, '/');
  const explorer = useFileExplorerStore.getState();
  const workspace =
    explorer.workspaces.find(
      (item) => absolutePath === item.path || absolutePath.startsWith(`${item.path}/`),
    ) ??
    explorer.workspaces.find((item) => item.id === explorer.activeWorkspaceId) ??
    explorer.workspaces[0];
  if (!workspace) return null;

  // Guardrail: refuse paths outside any known workspace root.
  const underWorkspace =
    absolutePath === workspace.path ||
    absolutePath.startsWith(`${workspace.path}/`) ||
    (!absolutePath.startsWith('/') && !absolutePath.match(/^[A-Za-z]:\//));
  if (!underWorkspace && absolutePath.startsWith('/')) {
    const anyWs = explorer.workspaces.some(
      (w) => absolutePath === w.path || absolutePath.startsWith(`${w.path}/`),
    );
    if (!anyWs) return null;
  }

  const matched =
    explorer.workspaces.find(
      (item) => absolutePath === item.path || absolutePath.startsWith(`${item.path}/`),
    ) ?? workspace;
  const relativePath = absolutePath.startsWith(`${matched.path}/`)
    ? absolutePath.slice(matched.path.length + 1)
    : absolutePath.startsWith('/')
      ? absolutePath.replace(/^\/+/, '')
      : absolutePath;
  return { workspaceId: matched.id, relativePath };
}

function ensureTab(path: string, initialContent: string): string | null {
  const resolved = resolveWorkspaceForPath(path);
  if (!resolved) return null;
  const { workspaceId, relativePath } = resolved;
  const editor = useEditorStore.getState();
  const existing = editor.tabs.find(
    (tab) => tab.workspaceId === workspaceId && tab.path === relativePath,
  );
  if (existing) {
    editor.setActiveTab(existing.id);
    return existing.id;
  }
  editor.openFile(
    workspaceId,
    relativePath,
    initialContent,
    getLanguageFromPath(relativePath),
  );
  const opened = useEditorStore
    .getState()
    .tabs.find((tab) => tab.workspaceId === workspaceId && tab.path === relativePath);
  void useSettingsStore.getState().updateLayoutSettings({
    editorPanelVisible: true,
    filesPanelVisible: true,
  });
  return opened?.id ?? null;
}

/**
 * Apply a progressive resolved-edit stream_delta / stream_end to Monaco.
 * Auto-opens a tab on the first delta. Returns true when handled.
 */
export function applyEditStreamMessage(message: Message): boolean {
  if (message.type !== 'stream_delta' && message.type !== 'stream_end') {
    return false;
  }
  if (!isEditApplyStreamDelta(message.metadata)) {
    return false;
  }
  const meta = getEditApplyStreamMeta(message.metadata);
  if (!meta) return false;

  if (cancelledChangeIds.has(meta.changeId)) {
    return true;
  }

  streamedChangeIds.add(meta.changeId);

  let active = activeByChangeId.get(meta.changeId);
  if (!active) {
    active = {
      changeId: meta.changeId,
      path: meta.path,
      accumulated: '',
      cancelled: false,
      tabId: null,
    };
    activeByChangeId.set(meta.changeId, active);
  }
  if (active.cancelled) {
    return true;
  }

  if (message.type === 'stream_delta') {
    const chunk = message.content ?? '';
    // Prefer offset-based rebuild when server sends absolute offset.
    if (typeof meta.offset === 'number' && meta.offset >= 0) {
      if (meta.offset === 0) {
        active.accumulated = chunk;
      } else if (meta.offset === active.accumulated.length) {
        active.accumulated += chunk;
      } else if (meta.offset < active.accumulated.length) {
        active.accumulated = active.accumulated.slice(0, meta.offset) + chunk;
      } else {
        active.accumulated += chunk;
      }
    } else {
      active.accumulated += chunk;
    }

    if (!active.tabId) {
      active.tabId = ensureTab(meta.path, active.accumulated);
    }
    if (active.tabId) {
      useEditorStore.getState().applyExternalTabContent(active.tabId, active.accumulated);
    }
    return true;
  }

  // stream_end: hand off to pending hunk review when still interactive.
  activeByChangeId.delete(meta.changeId);
  const pending = useFileChangeStore
    .getState()
    .pendingChanges.find((c) => c.id === meta.changeId);
  if (pending && pending.status === 'pending') {
    void syncPendingChangeToEditor(pending).catch((err) => {
      console.error('Failed to hand off edit-apply stream to pending hunks:', err);
    });
  } else if (!pending) {
    // Proposal card / fetch may still be in flight — try after a short refresh.
    void useFileChangeStore
      .getState()
      .fetchPendingChanges('default')
      .then(async () => {
        const late = useFileChangeStore
          .getState()
          .pendingChanges.find((c) => c.id === meta.changeId);
        if (late && late.status === 'pending') {
          await syncPendingChangeToEditor(late);
        } else if (active.tabId) {
          // Auto-applied: pull finalized disk content into the open tab.
          const tab = useEditorStore.getState().getTabById(active.tabId);
          if (tab) {
            await useEditorStore.getState().refreshTabFromDisk(tab.workspaceId, tab.path);
          }
        }
      })
      .catch(() => {
        /* old path still has surfaceChangeProposal one-shot sync */
      });
  } else if (active.tabId) {
    const tab = useEditorStore.getState().getTabById(active.tabId);
    if (tab) {
      void useEditorStore.getState().refreshTabFromDisk(tab.workspaceId, tab.path);
    }
  }
  return true;
}
