import type { ChatAPI } from '../api/chatAPI';
import type { AgentInfo } from '../types/protocol';
import type { EditorTab } from '../stores/editorStore';
import type { EffectiveComposerMode } from '../constants/composerMode';
import {
  EDITOR_AGENT_TRUST_KEY,
  EDITOR_MODE_KEY,
  IMPLEMENTATION_SESSION_METADATA_KEY,
  IDE_ROUTE_AGENT_TYPE_KEY,
} from '../constants/promptMetadata';
import {
  applyIdeAskPrefix,
  applyIdePlanPrefix,
  mergeCodebaseAttachments,
} from './ideComposer';
import { resolveDmPartnerAgent } from './dmChannelDisplay';
import type { Channel } from '../types/protocol';

export function isSlashCommandContent(content: string): boolean {
  return content.trimStart().startsWith('/');
}

export type PrepareOutboundPayloadOptions = {
  content: string;
  composerMode: EffectiveComposerMode;
  agents: AgentInfo[];
  activeTab: EditorTab | null;
  editorAgentTrust: string;
  composerMetadata?: Record<string, unknown>;
  api?: ChatAPI;
  repoPath?: string;
  repoPaths?: string[];
  ideEnabled?: boolean;
  channel?: string;
  channelMeta?: Pick<Channel, 'type' | 'agents' | 'description' | 'name'>;
};

/**
 * Unified send preparation for main channel and thread replies.
 * Maps Cursor-style Ask / Agent / Export to editor_mode + implementation_session metadata.
 */
export async function prepareOutboundPayload(
  options: PrepareOutboundPayloadOptions
): Promise<{ content: string; metadata: Record<string, unknown> }> {
  const {
    content,
    composerMode,
    agents,
    editorAgentTrust,
    composerMetadata,
    api,
    repoPath,
    repoPaths,
  } = options;

  const slashCommand = isSlashCommandContent(content);

  let sendContent = content;
  const effectiveMode = composerMode;
  if (!slashCommand && effectiveMode === 'ask') {
    sendContent = applyIdeAskPrefix(content, 'ask');
  } else if (!slashCommand && effectiveMode === 'plan') {
    sendContent = applyIdePlanPrefix(content, 'plan');
  }

  const hasExplicitMention = /@\w/.test(content);
  const isDm = options.channelMeta?.type === 'dm';
  const dmPartner = resolveDmPartnerAgent(options.channel ?? '', options.channelMeta, agents);

  const metadata: Record<string, unknown> = slashCommand
    ? { ...(composerMetadata ?? {}) }
    : {
        ...(composerMetadata ?? {}),
        ...(isDm && !hasExplicitMention && dmPartner?.type
          ? { [IDE_ROUTE_AGENT_TYPE_KEY]: dmPartner.type }
          : {}),
        [EDITOR_MODE_KEY]: effectiveMode,
        [EDITOR_AGENT_TRUST_KEY]:
          effectiveMode === 'ask' || effectiveMode === 'plan'
            ? 'interactive'
            : editorAgentTrust,
      };

  const scopedPaths =
    repoPaths?.filter((p) => p?.trim()).map((p) => p.trim()) ??
    (repoPath?.trim() ? [repoPath.trim()] : []);

  if (slashCommand) {
    if (api && scopedPaths.length > 0) {
      return {
        content: sendContent,
        metadata: await mergeCodebaseAttachments(api, sendContent, repoPath, metadata, scopedPaths),
      };
    }
    return { content: sendContent, metadata };
  }

  if (effectiveMode === 'export') {
    metadata[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
  } else {
    delete metadata[IMPLEMENTATION_SESSION_METADATA_KEY];
  }

  if (api && scopedPaths.length > 0) {
    return {
      content: sendContent,
      metadata: await mergeCodebaseAttachments(api, sendContent, repoPath, metadata, scopedPaths),
    };
  }

  return { content: sendContent, metadata };
}
