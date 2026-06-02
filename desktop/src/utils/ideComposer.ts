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
import { hasCodeTaskSignals } from './conversationMode';

export type EditorAgentMode = 'ask' | 'agent';

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
  const agentType = pickAgentTypeFromTab(options.activeTab);
  const hasExplicitMention = /@\w/.test(options.content);
  const content = applyIdeAskPrefix(options.content, options.editorAgentMode);
  const metadata: Record<string, unknown> = {
    ...(options.composerMetadata ?? {}),
    ...(hasExplicitMention ? {} : { [IDE_ROUTE_AGENT_TYPE_KEY]: agentType }),
    [EDITOR_MODE_KEY]: options.editorAgentMode,
    [EDITOR_AGENT_TRUST_KEY]: options.editorAgentTrust,
  };
  if (options.editorAgentMode === 'agent' && hasCodeTaskSignals(options.content)) {
    metadata[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
  }
  return { content, metadata };
}

export async function mergeCodebaseAttachments(
  api: ChatAPI,
  content: string,
  repoPath: string | undefined,
  metadata: Record<string, unknown>
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
  if (!messageRequestsCodebase(content) || !repoPath?.trim()) {
    return next;
  }
  try {
    const { chunks } = await api.repoSemanticSearch({
      repoPath,
      query: content.replace(/@codebase\b/gi, '').trim(),
      limit: 6,
    });
    const existing = Array.isArray(next[PROMPT_ATTACHMENTS_METADATA_KEY])
      ? [...(next[PROMPT_ATTACHMENTS_METADATA_KEY] as unknown[])]
      : [];
    for (const ch of chunks) {
      existing.push({ type: 'codebase_chunk', path: ch.path, content: ch.content });
    }
    if (existing.length > 0) {
      return { ...next, [PROMPT_ATTACHMENTS_METADATA_KEY]: existing };
    }
  } catch (e) {
    console.error('[ide] @codebase search failed:', e);
  }
  return next;
}
