import type { ChatAPI } from '../api/chatAPI';
import type { AgentInfo } from '../types/protocol';
import type { EditorTab } from '../stores/editorStore';
import type { ComposerMode } from '../constants/composerMode';
import { resolveEffectiveComposerMode } from '../constants/composerMode';
import {
  CONVERSATION_MODE_METADATA_KEY,
  EDITOR_AGENT_TRUST_KEY,
  EDITOR_MODE_KEY,
  IMPLEMENTATION_SESSION_METADATA_KEY,
  IDE_ROUTE_AGENT_TYPE_KEY,
} from '../constants/promptMetadata';
import {
  applyIdeAskPrefix,
  applyIdePlanPrefix,
  buildImplementationSessionMetadata,
  mergeCodebaseAttachments,
  pickAgentTypeForImplementation,
} from './ideComposer';
import { hasBootFixRoutingSignals } from './bootFixRouting';
import { hasCodeReviewSignals } from './codeReviewSignals';
import { resolveDmPartnerAgent } from './dmChannelDisplay';
import type { Channel } from '../types/protocol';

export function isSlashCommandContent(content: string): boolean {
  return content.trimStart().startsWith('/');
}

export type PrepareOutboundPayloadOptions = {
  content: string;
  composerMode: ComposerMode;
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
    activeTab,
    editorAgentTrust,
    composerMetadata,
    api,
    repoPath,
    repoPaths,
    ideEnabled = false,
  } = options;

  const slashCommand = isSlashCommandContent(content);

  let sendContent = content;
  const effectiveMode = slashCommand ? 'agent' : resolveEffectiveComposerMode(content, composerMode);
  if (!slashCommand && effectiveMode === 'ask') {
    sendContent = applyIdeAskPrefix(content, 'ask');
  } else if (!slashCommand && effectiveMode === 'plan') {
    sendContent = applyIdePlanPrefix(content, 'plan');
  }

  const hasExplicitMention = /@\w/.test(content);
  const isDm = options.channelMeta?.type === 'dm';
  const dmPartner = resolveDmPartnerAgent(options.channel ?? '', options.channelMeta, agents);
  let agentType = pickAgentTypeForImplementation(content, activeTab, agents);
  if (isDm && dmPartner?.type) {
    agentType = dmPartner.type;
  } else if (!hasExplicitMention && hasBootFixRoutingSignals(content)) {
    agentType = 'frontend';
  }

  const metadata: Record<string, unknown> = slashCommand
    ? { ...(composerMetadata ?? {}) }
    : {
        ...(composerMetadata ?? {}),
        ...(hasExplicitMention ? {} : { [IDE_ROUTE_AGENT_TYPE_KEY]: agentType }),
        [EDITOR_MODE_KEY]: effectiveMode,
        [EDITOR_AGENT_TRUST_KEY]:
          effectiveMode === 'ask' || effectiveMode === 'plan'
            ? 'interactive'
            : 'auto_apply_edits',
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
    metadata[CONVERSATION_MODE_METADATA_KEY] = 'code';
    metadata[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
  } else if (
    effectiveMode === 'agent' &&
    ideEnabled &&
    !hasCodeReviewSignals(content)
  ) {
    const implMeta = buildImplementationSessionMetadata({
      content,
      agents,
      activeTab,
      editorAgentMode: 'agent',
      editorAgentTrust,
      composerMetadata: metadata,
      channelType: options.channelMeta?.type,
      dmPartnerAgentType: dmPartner?.type,
    });
    if (implMeta[IMPLEMENTATION_SESSION_METADATA_KEY]) {
      metadata[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
    }
  }

  if (api && scopedPaths.length > 0) {
    return {
      content: sendContent,
      metadata: await mergeCodebaseAttachments(api, sendContent, repoPath, metadata, scopedPaths),
    };
  }

  return { content: sendContent, metadata };
}
