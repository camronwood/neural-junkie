import { describe, expect, it } from 'vitest';
import {
  agentsForTrack,
  inferWizardTrackFromPacks,
  modelsToEnsureForTrack,
  ollamaModelForTrack,
  UTILITY_OLLAMA_MODEL,
} from './wizardProfiles';

describe('inferWizardTrackFromPacks', () => {
  it('returns general when packs are missing or empty', () => {
    expect(inferWizardTrackFromPacks(undefined)).toBe('general');
    expect(inferWizardTrackFromPacks(null)).toBe('general');
    expect(inferWizardTrackFromPacks({})).toBe('general');
  });

  it('prefers life-sciences when enabled', () => {
    expect(
      inferWizardTrackFromPacks({
        'life-sciences': true,
        ide: true,
        cad: true,
      }),
    ).toBe('lifeSciences');
  });

  it('returns cad when cad is enabled and life-sciences is not', () => {
    expect(
      inferWizardTrackFromPacks({
        cad: true,
        ide: true,
      }),
    ).toBe('cad');
  });

  it('returns developer when ide or software-development is enabled', () => {
    expect(inferWizardTrackFromPacks({ ide: true })).toBe('developer');
    expect(inferWizardTrackFromPacks({ 'software-development': true })).toBe('developer');
  });

  it('developer track seeds Assistant + BackendEngineer on 9b', () => {
    expect(agentsForTrack('developer')).toEqual([
      { type: 'assistant', name: 'Assistant', enabled: true },
      { type: 'backend', name: 'BackendEngineer', enabled: true },
    ]);
    expect(ollamaModelForTrack('developer')).toBe(UTILITY_OLLAMA_MODEL);
    expect(modelsToEnsureForTrack('developer', 'ollama')).toEqual([UTILITY_OLLAMA_MODEL]);
  });

  it('ignores disabled pack flags', () => {
    expect(
      inferWizardTrackFromPacks({
        'life-sciences': false,
        cad: false,
        ide: false,
        'software-development': false,
      }),
    ).toBe('general');
  });
});
