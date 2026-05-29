import type { ChatAPI } from '../api/chatAPI';
import type { AgentInfo } from '../types/protocol';
import type { EditorTab } from '../stores/editorStore';
import {
  EDITOR_AGENT_TRUST_KEY,
  EDITOR_MODE_KEY,
  IDE_ROUTE_AGENT_TYPE_KEY,
  PROMPT_ATTACHMENTS_METADATA_KEY,
} from '../constants/promptMetadata';

export type EditorAgentMode = 'ask' | 'agent';

export function pickAgentTypeFromTab(activeTab: EditorTab | null): string {
  if (activeTab?.path?.endsWith('.go') || activeTab?.language === 'go') return 'backend';
  if (
    activeTab?.path?.match(/\.(tsx?|jsx?)$/) ||
    activeTab?.language?.includes('typescript') ||
    activeTab?.language?.includes('javascript')
  ) {
    return 'frontend';
  }
  return 'backend';
}

export function resolveIdeAgentName(agents: AgentInfo[], agentType: string): string {
  const match = agents.find((a) => a.type === agentType);
  return match?.name ?? (agentType === 'frontend' ? 'FrontendEngineer' : 'BackendEngineer');
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

/** Prepare content + routing metadata for IDE layout sends on the main channel. */
export function buildIdeDispatchPayload(options: {
  content: string;
  agents: AgentInfo[];
  activeTab: EditorTab | null;
  editorAgentMode: EditorAgentMode;
  editorAgentTrust: string;
  composerMetadata?: Record<string, unknown>;
}): { content: string; metadata: Record<string, unknown> } {
  const agentType = pickAgentTypeFromTab(options.activeTab);
  const agentName = resolveIdeAgentName(options.agents, agentType);
  const hasExplicitMention = /@\w/.test(options.content);
  const content = applyIdeAskPrefix(
    ensureIdeAgentMention(options.content, agentName),
    options.editorAgentMode
  );
  return {
    content,
    metadata: {
      ...(options.composerMetadata ?? {}),
      ...(hasExplicitMention ? {} : { [IDE_ROUTE_AGENT_TYPE_KEY]: agentType }),
      [EDITOR_MODE_KEY]: options.editorAgentMode,
      [EDITOR_AGENT_TRUST_KEY]: options.editorAgentTrust,
    },
  };
}

export async function mergeCodebaseAttachments(
  api: ChatAPI,
  content: string,
  repoPath: string | undefined,
  metadata: Record<string, unknown>
): Promise<Record<string, unknown>> {
  if (!messageRequestsCodebase(content) || !repoPath?.trim()) {
    return metadata;
  }
  try {
    const { chunks } = await api.repoSemanticSearch({
      repoPath,
      query: content.replace(/@codebase\b/gi, '').trim(),
      limit: 6,
    });
    const existing = Array.isArray(metadata[PROMPT_ATTACHMENTS_METADATA_KEY])
      ? [...(metadata[PROMPT_ATTACHMENTS_METADATA_KEY] as unknown[])]
      : [];
    for (const ch of chunks) {
      existing.push({ type: 'codebase_chunk', path: ch.path, content: ch.content });
    }
    if (existing.length > 0) {
      return { ...metadata, [PROMPT_ATTACHMENTS_METADATA_KEY]: existing };
    }
  } catch (e) {
    console.error('[ide] @codebase search failed:', e);
  }
  return metadata;
}
