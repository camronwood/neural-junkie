import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ChatAPI } from '../api/chatAPI';
import { useChatStore } from '../stores/chatStore';
import { collaborationsByIDSnapshot, useCollaborationsStore } from '../stores/collaborationsStore';
import type { Toast } from '../stores/toastStore';
import type { Collaboration } from '../types/protocol';
import { isTerminalCollaborationPhase } from '../utils/chatInboundCollab';
import { isNonTerminalCollaborationPhase, resolvePanelCollaboration } from '../utils/collaborationPanelState';
import { syncCollabTurnThinking } from '../utils/collabThinking';

export interface UseChatWindowCollaborationsArgs {
  api: ChatAPI;
  channel: string;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
}

function pruneTerminalCollaborations(
  next: Record<string, Collaboration>,
  channelName: string
): Record<string, Collaboration> {
  const terminals = Object.values(next)
    .filter(
      (c) =>
        isTerminalCollaborationPhase(c.phase) &&
        (c.channel || '') === channelName
    )
    .sort(
      (a, b) => Date.parse(b.updated_at || '') - Date.parse(a.updated_at || '')
    );
  if (terminals.length <= 3) return next;
  const drop = new Set(terminals.slice(3).map((c) => c.id));
  const pruned = { ...next };
  for (const id of drop) {
    delete pruned[id];
  }
  return pruned;
}

/**
 * Collaboration snapshot state, hub sync, polling, and per-channel derived views for ChatWindow.
 * Extraction loop 4 from ChatWindowRoot.
 */
export function useChatWindowCollaborations({ api, channel, addToast }: UseChatWindowCollaborationsArgs) {
  const [activeCollab, setActiveCollab] = useState<Collaboration | null>(null);
  const [runbookBuilderDirty, setRunbookBuilderDirty] = useState(false);
  const activeCollabRef = useRef<Collaboration | null>(null);
  const collaborationsByIDRef = useRef<Record<string, Collaboration>>({});
  const collaborationsByID = useCollaborationsStore((s) => s.byID);
  const setCollaborationsByID = useCollaborationsStore((s) => s.setByID);
  const mergeCollabSnapshot = useCollaborationsStore((s) => s.mergeSnapshot);

  useEffect(() => {
    activeCollabRef.current = activeCollab;
  }, [activeCollab]);

  useEffect(() => {
    collaborationsByIDRef.current = collaborationsByID;
  }, [collaborationsByID]);

  const mergeCollaborationSnapshot = useCallback((snapshot: Collaboration) => {
    if (!snapshot?.id) return;
    const prevSnap = collaborationsByIDSnapshot()[snapshot.id];
    const isTerminal = isTerminalCollaborationPhase(snapshot.phase);
    const wasTerminal = prevSnap && isTerminalCollaborationPhase(prevSnap.phase);

    mergeCollabSnapshot(snapshot, pruneTerminalCollaborations);

    if (isTerminal && !wasTerminal) {
      const completed =
        snapshot.tasks?.filter(t => t.status === 'completed').length ?? 0;
      const total = snapshot.tasks?.length ?? 0;
      addToast({
        type: 'success',
        title: 'Collaboration completed',
        message:
          total > 0
            ? `${snapshot.title || 'Collaboration'} — ${completed}/${total} tasks done.`
            : `${snapshot.title || 'Collaboration'} is closed.`,
      });
    }

    setActiveCollab(current => (current?.id === snapshot.id ? snapshot : current));

    const ch = snapshot.channel || useChatStore.getState().channel;
    if (ch && !isTerminal) {
      if (snapshot.phase === 'reviewing') {
        useChatStore.getState().clearThinkingAgents(ch);
      } else {
        syncCollabTurnThinking(snapshot, ch);
      }
    }
  }, [addToast, mergeCollabSnapshot]);

  const clearActiveCollabIf = useCallback((collaborationID: string) => {
    setActiveCollab(current => (current?.id === collaborationID ? null : current));
  }, []);

  const loadCollaborations = useCallback(async (targetChannel: string) => {
    try {
      const includeTerminal = targetChannel.startsWith('collab-');
      const snapshots = await api.fetchCollaborations(undefined, includeTerminal);
      setCollaborationsByID((prev) => {
        const next: Record<string, Collaboration> = {};
        for (const snapshot of snapshots) {
          if (!snapshot?.id) continue;
          if (isTerminalCollaborationPhase(snapshot.phase)) {
            if (includeTerminal && snapshot.channel === targetChannel) {
              next[snapshot.id] = snapshot;
            }
            continue;
          }
          next[snapshot.id] = snapshot;
        }
        // Keep channel snapshots when hub list is briefly stale (e.g. right after /runbook redirect).
        for (const existing of Object.values(prev)) {
          if (!existing?.id || next[existing.id]) continue;
          if (existing.channel !== targetChannel) continue;
          if (!isNonTerminalCollaborationPhase(existing.phase)) continue;
          next[existing.id] = existing;
        }
        if (includeTerminal) {
          return pruneTerminalCollaborations(next, targetChannel);
        }
        return next;
      });
      // Prefer the store map just updated above — collaborationsByIDRef lags one effect tick.
      // Keep the last good snapshot when the hub list briefly omits the active collab.
      setActiveCollab(current => {
        if (!current || current.channel !== targetChannel) return current;
        const refreshed = snapshots.find(snapshot => snapshot.id === current.id);
        if (refreshed) return refreshed;
        const cached =
          collaborationsByIDSnapshot()[current.id] ??
          collaborationsByIDRef.current[current.id];
        if (cached && cached.channel === targetChannel) return cached;
        return current;
      });
    } catch (error) {
      console.error('Failed to load collaborations:', error);
      addToast({
        type: 'error',
        title: 'Could not load collaborations',
        message:
          error instanceof Error
            ? error.message
            : 'Failed to refresh collaboration snapshots. Try again or check hub connectivity.',
      });
    }
  }, [addToast, api]);

  // Hub phase changes are not always re-broadcast on every message; poll while panel is open.
  useEffect(() => {
    if (!activeCollab?.id) return;
    const targetChannel = activeCollab.channel?.trim() || channel;
    const tick = () => {
      if (
        runbookBuilderDirty &&
        activeCollab.source === 'runbook' &&
        (activeCollab.phase === 'draft' || activeCollab.phase === 'reviewing')
      ) {
        return;
      }
      const latest = collaborationsByIDRef.current[activeCollab.id];
      if (!latest || !isNonTerminalCollaborationPhase(latest.phase)) {
        return;
      }
      void loadCollaborations(targetChannel);
    };
    tick();
    const id = window.setInterval(tick, 10_000);
    return () => window.clearInterval(id);
  }, [activeCollab?.id, activeCollab?.channel, activeCollab?.phase, activeCollab?.source, channel, loadCollaborations, runbookBuilderDirty]);

  const activeCollabForChannel = useMemo(
    () => Object.values(collaborationsByID).find((c) => c.channel === channel),
    [collaborationsByID, channel],
  );

  const trackedCollaborations = useMemo(
    () =>
      Object.values(collaborationsByID).sort(
        (a, b) => Date.parse(b.updated_at || '') - Date.parse(a.updated_at || '')
      ),
    [collaborationsByID]
  );

  const executingCollaborationForChannel = useMemo(
    () =>
      trackedCollaborations.find(c => c.channel === channel && c.phase === 'executing') ?? null,
    [trackedCollaborations, channel]
  );

  const collaborationForChannel = useMemo(
    () => trackedCollaborations.find(c => c.channel === channel) ?? null,
    [trackedCollaborations, channel]
  );

  const panelCollaboration = useMemo(
    () => resolvePanelCollaboration(activeCollab, collaborationsByID),
    [activeCollab, collaborationsByID]
  );

  const isClosedCollaborationChannel = Boolean(
    collaborationForChannel && isTerminalCollaborationPhase(collaborationForChannel.phase)
  );

  const extendableCollaborations = useMemo(
    () =>
      trackedCollaborations.filter(
        (c) =>
          (c.phase === 'planning' || c.phase === 'reviewing') &&
          (c.discussion?.status === 'budget_exhausted' ||
            c.discussion?.status === 'timed_out' ||
            c.discussion?.status === 'active')
      ),
    [trackedCollaborations]
  );

  return {
    activeCollab,
    setActiveCollab,
    runbookBuilderDirty,
    setRunbookBuilderDirty,
    activeCollabRef,
    collaborationsByIDRef,
    collaborationsByID,
    mergeCollaborationSnapshot,
    clearActiveCollabIf,
    loadCollaborations,
    activeCollabForChannel,
    trackedCollaborations,
    executingCollaborationForChannel,
    collaborationForChannel,
    panelCollaboration,
    isClosedCollaborationChannel,
    extendableCollaborations,
  };
}
