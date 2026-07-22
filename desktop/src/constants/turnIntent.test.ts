import { describe, expect, it } from 'vitest';
import {
  resolveTurnCapabilities,
  attachTurnCapabilitiesMetadata,
  TURN_META_GOVERNANCE,
} from './turnIntent';

describe('turnIntent', () => {
  it('export mode requires workspace and impl session', () => {
    const caps = resolveTurnCapabilities({
      composerMode: 'export',
      contextScope: 'outline',
      implementationSession: true,
    });
    expect(caps.canProposeFiles).toBe(true);
    expect(caps.canRunImplSession).toBe(true);
    expect(caps.requiresWorkspace).toBe(true);
  });

  it('ask mode is read-only', () => {
    const caps = resolveTurnCapabilities({
      composerMode: 'ask',
      contextScope: 'none',
    });
    expect(caps.canProposeFiles).toBe(false);
    expect(caps.canRunImplSession).toBe(false);
  });

  it('attachTurnCapabilitiesMetadata emits versioned turn_governance', () => {
    const caps = resolveTurnCapabilities({
      composerMode: 'export',
      contextScope: 'outline',
      implementationSession: true,
    });
    const meta = attachTurnCapabilitiesMetadata({}, caps, {
      trustPreference: 'interactive',
    });
    expect(meta.composer_mode).toBe('export');
    expect(meta.can_run_impl_session).toBeUndefined();
    expect(meta.can_propose_files).toBeUndefined();
    expect(meta.requires_workspace).toBeUndefined();
    expect(meta[TURN_META_GOVERNANCE]).toMatchObject({
      version: 1,
      composer_mode: 'export',
      context_tier: 'outline',
      can_run_impl_session: true,
      provenance: 'desktop_explicit',
      trust_preference: 'interactive',
    });
  });
});
