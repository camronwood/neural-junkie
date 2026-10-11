import type { Settings } from '../stores/settingsStore';
import type { AgentInfo, Channel } from '../types/protocol';
import {
  agentSidebarHideKey,
  dmChannelNamesForAgent,
  parseDMDisplayName,
} from './dmChannelDisplay';
import {
  agentTypeFromSidebarKey,
  isSoftDeleteOnlyAgentType,
} from './packAgents';

export { agentSidebarHideKey } from './dmChannelDisplay';

export function isAgentShortcutHidden(settings: Settings, agent: AgentInfo): boolean {
  const keys = settings.hiddenAgentSidebarKeys ?? [];
  const legacyIds = settings.hiddenAgentIdsForSidebar ?? [];
  const key = agentSidebarHideKey(agent);
  if (keys.includes(key) || legacyIds.includes(agent.id)) return true;
  // Pack/system agents previously hard-deleted are treated as soft-hidden until migrated.
  if (isSoftDeleteOnlyAgentType(agent.type)) {
    const deletedKeys = settings.deletedAgentSidebarKeys ?? [];
    if (deletedKeys.includes(key)) return true;
  }
  return false;
}

export function isAgentShortcutDeleted(settings: Settings, agent: AgentInfo): boolean {
  // Pack/system specialists cannot be permanently removed from the sidebar.
  if (isSoftDeleteOnlyAgentType(agent.type)) return false;
  const keys = settings.deletedAgentSidebarKeys ?? [];
  const legacyIds = settings.deletedAgentIdsForSidebar ?? [];
  return keys.includes(agentSidebarHideKey(agent)) || legacyIds.includes(agent.id);
}

/**
 * Move pack/system keys out of deletedAgentSidebarKeys.
 * - Currently running specialists: cleared (shortcut returns under Direct Messages).
 * - Not running: soft-hidden so Unhide / Room specialists can restore them.
 * Custom expert/repo deletions are left alone.
 */
export function patchMigrateDeletedPackAgentsToHidden(
  settings: Settings,
  agents: AgentInfo[] = []
): Partial<Settings> | null {
  const deletedKeys = settings.deletedAgentSidebarKeys ?? [];
  if (deletedKeys.length === 0) return null;

  const liveKeys = new Set(
    agents.filter((a) => isAgentShownInSidebar(a)).map((a) => agentSidebarHideKey(a))
  );

  const keepDeleted: string[] = [];
  const moveToHidden: string[] = [];
  let clearedLive = false;
  for (const key of deletedKeys) {
    if (!isSoftDeleteOnlyAgentType(agentTypeFromSidebarKey(key))) {
      keepDeleted.push(key);
      continue;
    }
    if (liveKeys.has(key)) {
      clearedLive = true;
      continue;
    }
    moveToHidden.push(key);
  }
  if (moveToHidden.length === 0 && !clearedLive) return null;

  const hidden = new Set(settings.hiddenAgentSidebarKeys ?? []);
  for (const key of moveToHidden) hidden.add(key);

  return {
    deletedAgentSidebarKeys: keepDeleted,
    hiddenAgentSidebarKeys: [...hidden],
  };
}

export function isSidebarChannelDeleted(settings: Settings, ch: Channel): boolean {
  if (ch.type === 'dm') {
    return (settings.deletedDmChannelNames ?? []).includes(ch.name);
  }
  if (ch.type === 'collaboration') {
    return (settings.deletedCollaborationChannelNames ?? []).includes(ch.name);
  }
  return false;
}

/** Agents eligible for DM shortcut rows (matches hub runbook pool: active or idle, not paused/removed). */
export function isAgentShownInSidebar(agent: AgentInfo): boolean {
  if (agent.is_paused || agent.status === 'removed') return false;
  const status = agent.status?.trim() ?? '';
  return status === '' || status === 'active' || status === 'idle';
}

/** DM rows stay hidden when the linked specialist is not in the live agent list (pack off, stopped). */
export function isDmChannelVisibleInSidebar(ch: Channel, agents: AgentInfo[]): boolean {
  if (ch.type !== 'dm') return true;
  const ids = ch.agents?.map((a) => a.id) ?? ch.members ?? [];
  if (ids.some((id) => agents.some((a) => a.id === id && isAgentShownInSidebar(a)))) {
    return true;
  }
  const dmName = parseDMDisplayName(ch).toLowerCase();
  return agents.some(
    (a) => isAgentShownInSidebar(a) && a.name.toLowerCase() === dmName
  );
}

/** Move legacy UUID hides to stable type:name keys when agents are known. */
export function patchMigrateHiddenAgentSidebarKeys(
  settings: Settings,
  agents: AgentInfo[]
): Partial<Settings> | null {
  const legacyIds = settings.hiddenAgentIdsForSidebar ?? [];
  if (legacyIds.length === 0) return null;

  const keys = new Set(settings.hiddenAgentSidebarKeys ?? []);
  const remainingIds: string[] = [];

  for (const id of legacyIds) {
    const agent = agents.find((a) => a.id === id);
    if (agent) {
      keys.add(agentSidebarHideKey(agent));
    } else {
      remainingIds.push(id);
    }
  }

  const nextKeys = [...keys];
  const curKeys = settings.hiddenAgentSidebarKeys ?? [];
  const changed =
    nextKeys.length !== curKeys.length ||
    nextKeys.some((k, i) => k !== curKeys[i]) ||
    remainingIds.length !== legacyIds.length;

  if (!changed) return null;

  return {
    hiddenAgentSidebarKeys: nextKeys,
    hiddenAgentIdsForSidebar: remainingIds.length > 0 ? remainingIds : undefined,
  };
}

/** Remove hidden (and deleted) sidebar entries when the user navigates to or creates them. */
export function patchRevealSidebarItems(
  settings: Settings,
  opts: {
    agentIds?: string[];
    agentSidebarKeys?: string[];
    dmChannelNames?: string[];
    collabChannelNames?: string[];
  }
): Partial<Settings> | null {
  const patch: Partial<Settings> = {};
  let changed = false;

  const sidebarKeys = opts.agentSidebarKeys ?? [];
  if (sidebarKeys.length > 0) {
    const cur = settings.hiddenAgentSidebarKeys ?? [];
    const next = cur.filter((k) => !sidebarKeys.includes(k));
    if (next.length !== cur.length) {
      patch.hiddenAgentSidebarKeys = next;
      changed = true;
    }
    const delKeys = settings.deletedAgentSidebarKeys ?? [];
    const nextDelKeys = delKeys.filter((k) => !sidebarKeys.includes(k));
    if (nextDelKeys.length !== delKeys.length) {
      patch.deletedAgentSidebarKeys = nextDelKeys;
      changed = true;
    }
  }

  const agentIds = opts.agentIds ?? [];
  if (agentIds.length > 0) {
    const cur = settings.hiddenAgentIdsForSidebar ?? [];
    const next = cur.filter((id) => !agentIds.includes(id));
    if (next.length !== cur.length) {
      patch.hiddenAgentIdsForSidebar = next;
      changed = true;
    }
    const delIds = settings.deletedAgentIdsForSidebar ?? [];
    const nextDelIds = delIds.filter((id) => !agentIds.includes(id));
    if (nextDelIds.length !== delIds.length) {
      patch.deletedAgentIdsForSidebar = nextDelIds;
      changed = true;
    }
  }

  const dmNames = opts.dmChannelNames ?? [];
  if (dmNames.length > 0) {
    const cur = settings.hiddenDmChannelNames ?? [];
    const next = cur.filter((n) => !dmNames.includes(n));
    if (next.length !== cur.length) {
      patch.hiddenDmChannelNames = next;
      changed = true;
    }
    const delDm = settings.deletedDmChannelNames ?? [];
    const nextDelDm = delDm.filter((n) => !dmNames.includes(n));
    if (nextDelDm.length !== delDm.length) {
      patch.deletedDmChannelNames = nextDelDm;
      changed = true;
    }
  }

  const collabNames = opts.collabChannelNames ?? [];
  if (collabNames.length > 0) {
    const cur = settings.hiddenCollaborationChannelNames ?? [];
    const next = cur.filter((n) => !collabNames.includes(n));
    if (next.length !== cur.length) {
      patch.hiddenCollaborationChannelNames = next;
      changed = true;
    }
  }

  return changed ? patch : null;
}

/**
 * When an agent becomes active again (pack enable, join, restart), restore sidebar shortcut
 * and any matching hidden DM rows. Skips the first snapshot after load so hides survive restart.
 */
export function patchUnhideAgentShortcutsOnActivation(
  settings: Settings,
  agents: AgentInfo[],
  channels: Channel[],
  previousStatusById: Record<string, string | undefined>,
  options: { seeded: boolean }
): { patch: Partial<Settings> | null; nextStatusById: Record<string, string> } {
  const nextStatusById = Object.fromEntries(agents.map((a) => [a.id, a.status]));
  if (!options.seeded) {
    return { patch: null, nextStatusById };
  }

  const hiddenKeys = settings.hiddenAgentSidebarKeys ?? [];
  const hiddenIds = settings.hiddenAgentIdsForSidebar ?? [];
  if (hiddenKeys.length === 0 && hiddenIds.length === 0) {
    return { patch: null, nextStatusById };
  }

  const hiddenKeySet = new Set(hiddenKeys);
  const hiddenIdSet = new Set(hiddenIds);
  const keysToRemove = new Set<string>();
  const idsToRemove = new Set<string>();
  const dmToReveal = new Set<string>();

  for (const a of agents) {
    if (!isAgentShownInSidebar(a)) continue;
    const key = agentSidebarHideKey(a);
    const isHidden = hiddenKeySet.has(key) || hiddenIdSet.has(a.id);
    if (!isHidden) continue;

    const prev = previousStatusById[a.id];
    if (prev === 'active') continue;

    keysToRemove.add(key);
    idsToRemove.add(a.id);
    for (const dmName of dmChannelNamesForAgent(channels, a)) {
      dmToReveal.add(dmName);
    }
  }

  if (keysToRemove.size === 0 && idsToRemove.size === 0 && dmToReveal.size === 0) {
    return { patch: null, nextStatusById };
  }

  const patch = patchRevealSidebarItems(settings, {
    agentSidebarKeys: [...keysToRemove],
    agentIds: [...idsToRemove],
    dmChannelNames: [...dmToReveal],
  });

  return { patch, nextStatusById };
}

/** Reveal hidden sidebar entries for all currently active agents (e.g. after enabling a pack). */
export function patchRevealActiveAgentsInSidebar(
  settings: Settings,
  agents: AgentInfo[],
  channels: Channel[]
): Partial<Settings> | null {
  const active = agents.filter((a) => isAgentShownInSidebar(a));
  if (active.length === 0) return null;

  const keys: string[] = [];
  const ids: string[] = [];
  const dms = new Set<string>();

  const hiddenDmSet = new Set(settings.hiddenDmChannelNames ?? []);

  for (const a of active) {
    for (const name of dmChannelNamesForAgent(channels, a)) {
      if (hiddenDmSet.has(name)) {
        dms.add(name);
      }
    }
    if (!isAgentShortcutHidden(settings, a)) continue;
    keys.push(agentSidebarHideKey(a));
    ids.push(a.id);
  }

  return patchRevealSidebarItems(settings, {
    agentSidebarKeys: keys,
    agentIds: ids,
    dmChannelNames: [...dms],
  });
}

/** Build reveal patch from a channel row (DM or collaboration). */
export function patchRevealForChannel(
  settings: Settings,
  channelName: string,
  channels: Channel[],
  agents: AgentInfo[]
): Partial<Settings> | null {
  const ch = channels.find((c) => c.name === channelName);
  if (!ch) return null;

  if (ch.type === 'dm') {
    const agentId = ch.agents?.[0]?.id ?? ch.members?.[0];
    const agent =
      (agentId ? agents.find((a) => a.id === agentId) : undefined) ??
      agents.find((a) => a.name.toLowerCase() === parseDMDisplayName(ch).toLowerCase());
    return patchRevealSidebarItems(settings, {
      dmChannelNames: [channelName],
      agentIds: agent ? [agent.id] : agentId ? [agentId] : undefined,
      agentSidebarKeys: agent ? [agentSidebarHideKey(agent)] : undefined,
    });
  }

  if (ch.type === 'collaboration') {
    return patchRevealSidebarItems(settings, {
      collabChannelNames: [channelName],
    });
  }

  return null;
}
