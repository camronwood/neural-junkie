import type { ComposerMode } from './composerMode';
import type { ContextScope } from './promptMetadata';

/** Mirrors internal/protocol/turn_intent.go — UI sets metadata; agent resolves capabilities. */
export type TurnCapabilities = {
  composerMode: ComposerMode;
  contextTier: ContextScope;
  canProposeFiles: boolean;
  canRunImplSession: boolean;
  requiresWorkspace: boolean;
};

export const TURN_META_COMPOSER_MODE = 'composer_mode';
export const TURN_META_CONTEXT_TIER = 'context_tier';
export const TURN_META_CAN_PROPOSE_FILES = 'can_propose_files';
export const TURN_META_CAN_RUN_IMPL_SESSION = 'can_run_impl_session';
export const TURN_META_REQUIRES_WORKSPACE = 'requires_workspace';

export function resolveTurnCapabilities(options: {
  composerMode: ComposerMode;
  contextScope: ContextScope;
  implementationSession?: boolean;
}): TurnCapabilities {
  const { composerMode, contextScope, implementationSession } = options;
  switch (composerMode) {
    case 'ask':
      return {
        composerMode,
        contextTier: contextScope,
        canProposeFiles: false,
        canRunImplSession: false,
        requiresWorkspace: false,
      };
    case 'export':
      return {
        composerMode,
        contextTier: contextScope,
        canProposeFiles: true,
        canRunImplSession: true,
        requiresWorkspace: true,
      };
    default:
      return {
        composerMode: 'agent',
        contextTier: contextScope,
        canProposeFiles: true,
        canRunImplSession: Boolean(implementationSession),
        requiresWorkspace: Boolean(implementationSession),
      };
  }
}

export function attachTurnCapabilitiesMetadata(
  metadata: Record<string, unknown>,
  caps: TurnCapabilities
): Record<string, unknown> {
  return {
    ...metadata,
    [TURN_META_COMPOSER_MODE]: caps.composerMode,
    [TURN_META_CONTEXT_TIER]: caps.contextTier,
    [TURN_META_CAN_PROPOSE_FILES]: caps.canProposeFiles,
    [TURN_META_CAN_RUN_IMPL_SESSION]: caps.canRunImplSession,
    [TURN_META_REQUIRES_WORKSPACE]: caps.requiresWorkspace,
  };
}
