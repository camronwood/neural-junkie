import {
  COLLAB_SOURCE_MODE_KEY,
  COLLAB_SOURCE_PATH_KEY,
} from '../constants/collabWorkspace';
import { buildHumanOutboundMetadata } from './outboundChatMetadata';
import type { Collaboration } from '../types/protocol';
import type { WorkspaceContextMode } from '../constants/promptMetadata';

/**
 * Metadata for human messages sent on a collaboration channel (slash commands, revise, etc.).
 */
export function buildCollabChannelOutboundMetadata(
  collaboration: Collaboration,
  message: string,
  options?: {
    contextMode?: WorkspaceContextMode;
    workspacePath?: string;
  }
): Record<string, unknown> | undefined {
  const contextMode = options?.contextMode ?? 'auto';
  const repoPath = (options?.workspacePath ?? collaboration.source_repo_path ?? '').trim();
  const composerMetadata: Record<string, unknown> = {};
  if (repoPath) {
    composerMetadata[COLLAB_SOURCE_MODE_KEY] = 'path';
    composerMetadata[COLLAB_SOURCE_PATH_KEY] = repoPath;
  } else if (collaboration.working_directory?.trim()) {
    composerMetadata[COLLAB_SOURCE_MODE_KEY] = 'path';
    composerMetadata[COLLAB_SOURCE_PATH_KEY] = collaboration.working_directory.trim();
  }

  return buildHumanOutboundMetadata({
    contextMode,
    conversationMode: 'auto',
    message,
    channel: collaboration.channel,
    channelType: 'collaboration',
    composerMetadata,
  });
}
