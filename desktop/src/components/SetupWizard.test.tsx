import { describe, expect, it } from 'vitest';
import { packsEnabledForTrack } from '../config/wizardProfiles';

describe('SetupWizard track packs', () => {
  it('developer track enables software-development pack', () => {
    const enabled = packsEnabledForTrack('developer');
    expect(enabled['software-development']).toBe(true);
    expect(enabled.ide).toBe(true);
  });

  it('life sciences track enables life-sciences pack', () => {
    const enabled = packsEnabledForTrack('lifeSciences');
    expect(enabled['life-sciences']).toBe(true);
  });
});
