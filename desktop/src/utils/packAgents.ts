/**
 * Pack / system specialists vs user-created agents.
 * Keep in sync with internal/packs.KnownSpecialistAgentTypes (+ always-on core types).
 */

const PACK_SPECIALIST_TYPES = new Set([
  'backend',
  'frontend',
  'devops',
  'security',
  'architecture',
  'database',
  'rust',
  'sre',
  'mobile',
  'data-ml',
  'biology',
  'genomics',
  'structural-biology',
  'cheminformatics',
  'cad',
  'manufacturing',
  'aws',
  'incident',
  'arena',
  // retired pack specialists still treated as pack-managed for sidebar soft-delete
  'code-review',
  'maps',
  'music',
  'browser',
]);

const CORE_SOFT_DELETE_TYPES = new Set(['assistant', 'moderator', 'cli', 'confluence']);

/** User-created agents that may be permanently removed from the hub / sidebar. */
const HARD_DELETE_TYPES = new Set(['expert', 'repo']);

export function normalizeAgentType(type: string | undefined | null): string {
  return (type ?? '').trim().toLowerCase();
}

/** In-process specialists owned by domain packs (Room specialists toggles). */
export function isPackSpecialistAgentType(type: string | undefined | null): boolean {
  return PACK_SPECIALIST_TYPES.has(normalizeAgentType(type));
}

/**
 * Agents that must not be hard-deleted from the sidebar — only soft-hidden.
 * Pack specialists + always-on / auto-detected system agents.
 */
export function isSoftDeleteOnlyAgentType(type: string | undefined | null): boolean {
  const t = normalizeAgentType(type);
  return PACK_SPECIALIST_TYPES.has(t) || CORE_SOFT_DELETE_TYPES.has(t);
}

/** Custom experts and user repo agents — hard delete is allowed. */
export function isHardDeletableAgentType(type: string | undefined | null): boolean {
  return HARD_DELETE_TYPES.has(normalizeAgentType(type));
}

/** Parse `type:name` sidebar keys written by agentSidebarHideKey. */
export function agentTypeFromSidebarKey(key: string): string {
  const idx = key.indexOf(':');
  return idx >= 0 ? key.slice(0, idx) : '';
}
