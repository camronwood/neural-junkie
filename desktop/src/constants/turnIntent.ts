import type { EffectiveComposerMode } from './composerMode';
import type { ContextScope } from './promptMetadata';

/** Mirrors internal/protocol/turn_intent.go — UI sets metadata; agent resolves capabilities. */
export type TurnCapabilities = {
  composerMode: EffectiveComposerMode;
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
export const TURN_META_GOVERNANCE = 'turn_governance';
export const TURN_GOVERNANCE_VERSION = 1;

export function resolveTurnCapabilities(options: {
  composerMode: EffectiveComposerMode;
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
    case 'plan':
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

/**
 * Attach a versioned turn_governance envelope. Flat can_* keys are omitted —
 * the server stamps canonical governance and capabilities at ingress.
 * composer_mode remains as a compatibility reader for older ingress paths.
 */
export function attachTurnCapabilitiesMetadata(
  metadata: Record<string, unknown>,
  caps: TurnCapabilities,
  options?: { trustPreference?: string }
): Record<string, unknown> {
  const next: Record<string, unknown> = {
    ...metadata,
    [TURN_META_COMPOSER_MODE]: caps.composerMode,
    [TURN_META_GOVERNANCE]: {
      version: TURN_GOVERNANCE_VERSION,
      composer_mode: caps.composerMode,
      context_tier: caps.contextTier,
      trust_preference: options?.trustPreference ?? '',
      can_propose_files: caps.canProposeFiles,
      can_run_impl_session: caps.canRunImplSession,
      requires_workspace: caps.requiresWorkspace,
      provenance: 'desktop_explicit',
    },
  };
  delete next[TURN_META_CONTEXT_TIER];
  delete next[TURN_META_CAN_PROPOSE_FILES];
  delete next[TURN_META_CAN_RUN_IMPL_SESSION];
  delete next[TURN_META_REQUIRES_WORKSPACE];
  return next;
}
