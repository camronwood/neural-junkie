import { describe, expect, it } from 'vitest';
import { resolveTurnCapabilities, attachTurnCapabilitiesMetadata } from './turnIntent';

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

  it('attachTurnCapabilitiesMetadata sets keys', () => {
    const caps = resolveTurnCapabilities({
      composerMode: 'export',
      contextScope: 'outline',
      implementationSession: true,
    });
    const meta = attachTurnCapabilitiesMetadata({}, caps);
    expect(meta.composer_mode).toBe('export');
    expect(meta.can_run_impl_session).toBe(true);
  });
});
