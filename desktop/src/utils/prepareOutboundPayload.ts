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
  buildImplementationSessionMetadata,
  mergeCodebaseAttachments,
  pickAgentTypeForImplementation,
} from './ideComposer';
import { hasCodeReviewSignals } from './codeReviewSignals';

export type PrepareOutboundPayloadOptions = {
  content: string;
  composerMode: ComposerMode;
  agents: AgentInfo[];
  activeTab: EditorTab | null;
  editorAgentTrust: string;
  composerMetadata?: Record<string, unknown>;
  api?: ChatAPI;
  repoPath?: string;
  devPackEnabled?: boolean;
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
    devPackEnabled = false,
  } = options;

  let sendContent = content;
  const effectiveMode = resolveEffectiveComposerMode(content, composerMode);
  if (effectiveMode === 'ask') {
    sendContent = applyIdeAskPrefix(content, 'ask');
  }

  const hasExplicitMention = /@\w/.test(content);
  const agentType = pickAgentTypeForImplementation(content, activeTab, agents);

  const metadata: Record<string, unknown> = {
    ...(composerMetadata ?? {}),
    ...(hasExplicitMention ? {} : { [IDE_ROUTE_AGENT_TYPE_KEY]: agentType }),
    [EDITOR_MODE_KEY]: effectiveMode,
    [EDITOR_AGENT_TRUST_KEY]: editorAgentTrust,
  };

  if (effectiveMode === 'export') {
    metadata[CONVERSATION_MODE_METADATA_KEY] = 'code';
    metadata[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
  } else if (
    effectiveMode === 'agent' &&
    devPackEnabled &&
    !hasCodeReviewSignals(content)
  ) {
    const implMeta = buildImplementationSessionMetadata({
      content,
      agents,
      activeTab,
      editorAgentMode: 'agent',
      editorAgentTrust,
      composerMetadata: metadata,
    });
    if (implMeta[IMPLEMENTATION_SESSION_METADATA_KEY]) {
      metadata[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
    }
  }

  if (api && repoPath?.trim()) {
    return {
      content: sendContent,
      metadata: await mergeCodebaseAttachments(api, sendContent, repoPath, metadata),
    };
  }

  return { content: sendContent, metadata };
}
