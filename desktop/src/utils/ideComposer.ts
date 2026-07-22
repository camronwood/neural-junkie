import type { ChatAPI } from '../api/chatAPI';
import type { AgentInfo } from '../types/protocol';
import type { EditorTab } from '../stores/editorStore';
import {
  EDITOR_AGENT_TRUST_KEY,
  EDITOR_MODE_KEY,
  IDE_ROUTE_AGENT_TYPE_KEY,
  IMPLEMENTATION_SESSION_METADATA_KEY,
  PROMPT_ATTACHMENTS_METADATA_KEY,
} from '../constants/promptMetadata';

export type EditorAgentMode = 'ask' | 'plan' | 'agent';

/** Specialist type for IDE metadata routing (not injected as @mentions). */
export function pickAgentTypeFromTab(activeTab: EditorTab | null): string {
  const path = activeTab?.path?.toLowerCase() ?? '';
  const lang = activeTab?.language?.toLowerCase() ?? '';

  if (path.endsWith('.go') || lang === 'go') return 'backend';

  if (
    /\.(tsx?|jsx?|vue|svelte|css|scss|less|html?)$/.test(path) ||
    /tailwind\.config\./.test(path) ||
    /postcss\.config\./.test(path) ||
    lang.includes('typescript') ||
    lang.includes('javascript') ||
    lang === 'css' ||
    lang === 'html' ||
    lang === 'vue' ||
    lang === 'svelte'
  ) {
    return 'frontend';
  }

  // IDE default: frontend for typical app/UI work; Go/Rust/etc. still hit backend above.
  return 'frontend';
}

export function resolveIdeAgentName(agents: AgentInfo[], agentType: string): string {
  const match = agents.find((a) => a.type === agentType);
  return match?.name ?? (agentType === 'frontend' ? 'FrontendEngineer' : 'BackendEngineer');
}

/** Composer chip label for implicit IDE specialist routing. */
export function ideRoutingChipLabel(activeTab: EditorTab | null, agents: AgentInfo[]): string {
  const agentType = pickAgentTypeFromTab(activeTab);
  const name = resolveIdeAgentName(agents, agentType);
  return `→ ${name}`;
}

export function parseFileFolderAttachments(content: string): Array<{ type: string; path: string }> {
  const out: Array<{ type: string; path: string }> = [];
  const fileRe = /@file:([^\s]+)/gi;
  let m: RegExpExecArray | null;
  while ((m = fileRe.exec(content)) !== null) {
    out.push({ type: 'file_ref', path: m[1] });
  }
  const folderRe = /@folder:([^\s]+)/gi;
  while ((m = folderRe.exec(content)) !== null) {
    out.push({ type: 'folder_outline', path: m[1] });
  }
  return out;
}

export function ensureIdeAgentMention(content: string, agentName: string): string {
  if (/@\w/.test(content)) return content;
  const trimmed = content.trimStart();
  return `@${agentName} ${trimmed}`;
}

export function messageRequestsCodebase(content: string): boolean {
  return /@codebase\b/i.test(content);
}

export function applyIdeAskPrefix(content: string, mode: EditorAgentMode): string {
  if (mode !== 'ask') return content;
  if (content.toLowerCase().includes('[ask mode')) return content;
  return `[ASK mode — read-only tools, no file edits]\n${content}`;
}

export function applyIdePlanPrefix(content: string, mode: EditorAgentMode): string {
  if (mode !== 'plan') return content;
  if (content.toLowerCase().includes('[plan mode')) return content;
  return `[PLAN mode — outline approach and numbered steps only; no file edits or shell commands that modify the repo]\n${content}`;
}

/** Resolve specialist slug from @mention or active editor tab. */
export function pickAgentTypeForImplementation(
  content: string,
  activeTab: EditorTab | null,
  agents: AgentInfo[]
): string {
  const mentionRe = /@([A-Za-z][\w-]*)/;
  const m = mentionRe.exec(content);
  if (m) {
    const needle = m[1].toLowerCase();
    const match = agents.find(
      (a) =>
        a.name.toLowerCase() === needle ||
        a.name.toLowerCase().startsWith(needle) ||
        a.type.toLowerCase() === needle
    );
    if (match?.type) return match.type;
  }
  return pickAgentTypeFromTab(activeTab);
}

/**
 * Explicit IDE/dispatch metadata only. Semantic action selection and
 * implementation_session inference belong to the server turn router.
 */
export function buildImplementationSessionMetadata(options: {
  content: string;
  agents: AgentInfo[];
  activeTab: EditorTab | null;
  editorAgentMode: EditorAgentMode;
  editorAgentTrust: string;
  composerMetadata?: Record<string, unknown>;
  /** When set, DM sends use the channel partner type — not tab routing. */
  channelType?: string;
  dmPartnerAgentType?: string;
}): Record<string, unknown> {
  const hasExplicitMention = /@\w/.test(options.content);
  let agentType = pickAgentTypeForImplementation(
    options.content,
    options.activeTab,
    options.agents
  );
  const isDm = options.channelType === 'dm';
  if (isDm && options.dmPartnerAgentType) {
    agentType = options.dmPartnerAgentType;
  }
  const metadata: Record<string, unknown> = {
    ...(options.composerMetadata ?? {}),
    ...(hasExplicitMention || (!isDm && !options.activeTab)
      ? {}
      : { [IDE_ROUTE_AGENT_TYPE_KEY]: agentType }),
    [EDITOR_MODE_KEY]: options.editorAgentMode,
    [EDITOR_AGENT_TRUST_KEY]: options.editorAgentTrust,
  };
  // Only an explicit upstream flag may request an implementation session from the
  // desktop. Natural-language task detection is server-authoritative.
  if (options.composerMetadata?.[IMPLEMENTATION_SESSION_METADATA_KEY] === true) {
    metadata[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
  } else {
    delete metadata[IMPLEMENTATION_SESSION_METADATA_KEY];
  }
  return metadata;
}

/**
 * Prepare content + routing metadata for IDE layout sends on the main channel.
 * Does not inject @mentions — routing uses `ide_route_agent_type` only so an
 * earlier @FrontendEngineer thread is not overridden by auto-@BackendEngineer.
 */
export function buildIdeDispatchPayload(options: {
  content: string;
  agents: AgentInfo[];
  activeTab: EditorTab | null;
  editorAgentMode: EditorAgentMode;
  editorAgentTrust: string;
  composerMetadata?: Record<string, unknown>;
}): { content: string; metadata: Record<string, unknown> } {
  const content = applyIdePlanPrefix(
    applyIdeAskPrefix(options.content, options.editorAgentMode),
    options.editorAgentMode
  );
  const metadata = buildImplementationSessionMetadata(options);
  return { content, metadata };
}

export async function mergeCodebaseAttachments(
  api: ChatAPI,
  content: string,
  repoPath: string | undefined,
  metadata: Record<string, unknown>,
  repoPaths?: string[]
): Promise<Record<string, unknown>> {
  let next = { ...metadata };
  const fileRefs = parseFileFolderAttachments(content);
  if (fileRefs.length > 0) {
    const existing = Array.isArray(next[PROMPT_ATTACHMENTS_METADATA_KEY])
      ? [...(next[PROMPT_ATTACHMENTS_METADATA_KEY] as unknown[])]
      : [];
    for (const ref of fileRefs) {
      existing.push(ref);
    }
    next = { ...next, [PROMPT_ATTACHMENTS_METADATA_KEY]: existing };
  }
  const paths = (repoPaths ?? []).filter((p) => p?.trim()).map((p) => p.trim());
  const searchPaths = paths.length > 0 ? paths : repoPath?.trim() ? [repoPath.trim()] : [];
  if (!messageRequestsCodebase(content) || searchPaths.length === 0) {
    return next;
  }
  try {
    const { chunks } = await api.repoSemanticSearch({
      repoPaths: searchPaths,
      query: content.replace(/@codebase\b/gi, '').trim(),
      limit: 6,
    });
    const existing = Array.isArray(next[PROMPT_ATTACHMENTS_METADATA_KEY])
      ? [...(next[PROMPT_ATTACHMENTS_METADATA_KEY] as unknown[])]
      : [];
    for (const ch of chunks) {
      existing.push({
        type: 'codebase_chunk',
        path: ch.path,
        start_line: ch.start_line,
        end_line: ch.end_line,
        content: ch.content,
        score: ch.score,
      });
    }
    next = { ...next, [PROMPT_ATTACHMENTS_METADATA_KEY]: existing };
  } catch {
    // Soft-fail: send without codebase chunks.
  }
  return next;
}
