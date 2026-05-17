/** Hub-local paths agents cannot see unless the user grants access for this message. */

import { GRANTED_HUB_DATA_ACCESS_KEY } from '../constants/promptMetadata';

export type HubDataAccessKind = 'file' | 'directory';

export interface HubDataAccessOption {
  id: string;
  kind: HubDataAccessKind;
  relativePath: string;
  label: string;
  description: string;
  defaultSelected: boolean;
}

export const HUB_DATA_ROOT_LABEL = '~/.neural-junkie';

const CATALOG: HubDataAccessOption[] = [
  {
    id: 'session-file',
    kind: 'file',
    relativePath: 'last-session.json',
    label: 'last-session.json',
    description: 'Local session resume cache (channels, recent messages, collabs).',
    defaultSelected: true,
  },
  {
    id: 'hub-dir',
    kind: 'directory',
    relativePath: '',
    label: HUB_DATA_ROOT_LABEL,
    description: 'Config, workspaces, collaborations folder, and other hub data (bounded scan).',
    defaultSelected: false,
  },
  {
    id: 'config-file',
    kind: 'file',
    relativePath: 'config.json',
    label: 'config.json',
    description: 'Hub agent and server configuration.',
    defaultSelected: false,
  },
  {
    id: 'workspaces-file',
    kind: 'file',
    relativePath: 'workspaces.json',
    label: 'workspaces.json',
    description: 'Registered file explorer workspaces.',
    defaultSelected: false,
  },
];

const SESSION_FILE_RE =
  /last-session(?:\.json)?|session\s+(?:file|log|snapshot)|review\s+(?:the\s+)?(?:last\s+)?session/i;
const HUB_DIR_RE =
  /(?:~\/|\/\.)?\.?neural-junkie\b|neural\s+junkie\s+(?:data|directory|folder|config)/i;
const HUB_FILE_RE = /\.neural-junkie\/([^\s"'`,;)]+)/gi;

function optionForRelativePath(rel: string, kind: HubDataAccessKind): HubDataAccessOption | null {
  const normalized = rel.replace(/^\.?\/?\.neural-junkie\/?/, '').trim();
  const match = CATALOG.find(
    (o) => o.kind === kind && o.relativePath === normalized
  );
  if (match) return { ...match, defaultSelected: true };
  if (kind === 'file' && normalized) {
    return {
      id: `file:${normalized}`,
      kind: 'file',
      relativePath: normalized,
      label: normalized,
      description: 'File under ~/.neural-junkie',
      defaultSelected: true,
    };
  }
  return null;
}

/**
 * Returns hub data paths the user may want to grant for this message (deduped).
 */
export function detectHubDataAccessNeeds(content: string): HubDataAccessOption[] {
  const text = content.trim();
  if (!text) return [];

  const found = new Map<string, HubDataAccessOption>();

  const add = (opt: HubDataAccessOption | null) => {
    if (!opt) return;
    found.set(`${opt.kind}:${opt.relativePath}`, { ...opt, defaultSelected: true });
  };

  if (SESSION_FILE_RE.test(text)) {
    add(CATALOG.find((c) => c.id === 'session-file') ?? null);
  }
  if (HUB_DIR_RE.test(text)) {
    add(CATALOG.find((c) => c.id === 'hub-dir') ?? null);
  }

  let m: RegExpExecArray | null;
  const re = new RegExp(HUB_FILE_RE.source, HUB_FILE_RE.flags);
  while ((m = re.exec(text)) !== null) {
    const rel = (m[1] ?? '').trim();
    if (!rel || rel.includes('..')) continue;
    add(optionForRelativePath(rel, 'file'));
  }

  // Full path: /Users/you/.neural-junkie/last-session.json
  const absPath = /\/[^\s"'`,;)]*\.neural-junkie\/([^\s"'`,;)]+)/gi;
  while ((m = absPath.exec(text)) !== null) {
    const rel = (m[1] ?? '').trim();
    if (!rel || rel.includes('..')) continue;
    add(optionForRelativePath(rel, 'file'));
  }

  // Explicit ~/.neural-junkie/... absolute-ish mentions
  const absish = /(?:~\/\.neural-junkie|\.neural-junkie)\/([^\s"'`,;)]+)/gi;
  while ((m = absish.exec(text)) !== null) {
    const rel = (m[1] ?? '').trim();
    if (!rel) {
      add(CATALOG.find((c) => c.id === 'hub-dir') ?? null);
      continue;
    }
    add(optionForRelativePath(rel, 'file'));
  }

  return [...found.values()];
}

export function hasGrantedHubDataAccess(metadata?: Record<string, unknown>): boolean {
  if (!metadata) return false;
  const raw = metadata[GRANTED_HUB_DATA_ACCESS_KEY] as { entries?: unknown[] } | undefined;
  return Array.isArray(raw?.entries) && raw.entries.length > 0;
}
