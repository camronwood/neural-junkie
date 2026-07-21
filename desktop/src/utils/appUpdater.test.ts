import { describe, expect, it } from 'vitest';
import {
  getUpdateChannelLabel,
  isPolicyMandatory,
  isRolloutEligible,
  isVersionBelow,
  rolloutBucket,
  type AppUpdatePolicy,
} from './appUpdater';

describe('getUpdateChannelLabel', () => {
  it('returns Beta for beta semver', () => {
    expect(getUpdateChannelLabel('1.0.0-beta.25')).toBe('Beta updates');
  });

  it('returns Stable for release semver', () => {
    expect(getUpdateChannelLabel('1.0.0', {
      schemaVersion: 1,
      channel: 'stable',
      severity: 'normal',
      enforcement: 'optional',
      rolloutPercentage: 100,
      rolloutSeed: 'v1.0.0',
    })).toBe('Stable updates');
  });
});

const policy: AppUpdatePolicy = {
  schemaVersion: 1,
  channel: 'stable',
  severity: 'normal',
  enforcement: 'optional',
  rolloutPercentage: 25,
  rolloutSeed: 'v1.2.3',
};

describe('update policy', () => {
  it('uses deterministic rollout buckets', () => {
    expect(rolloutBucket('seed', 'installation')).toBe(rolloutBucket('seed', 'installation'));
    expect(rolloutBucket('seed', 'installation')).toBeGreaterThanOrEqual(0);
    expect(rolloutBucket('seed', 'installation')).toBeLessThan(10_000);
  });

  it('honors zero and full rollouts', () => {
    expect(isRolloutEligible({ ...policy, rolloutPercentage: 0 }, 'id')).toBe(false);
    expect(isRolloutEligible({ ...policy, rolloutPercentage: 100 }, 'id')).toBe(true);
  });

  it('enforces mandatory updates only after their deadline', () => {
    const mandatory = {
      ...policy,
      enforcement: 'mandatory' as const,
      mandatoryAfter: '2026-07-22T00:00:00Z',
    };
    expect(isPolicyMandatory(mandatory, Date.parse('2026-07-21T23:59:59Z'))).toBe(false);
    expect(isPolicyMandatory(mandatory, Date.parse('2026-07-22T00:00:00Z'))).toBe(true);
  });

  it('makes clients below the minimum supported version mandatory', () => {
    expect(isVersionBelow('1.2.0-beta.7', '1.2.0-beta.8')).toBe(true);
    expect(isVersionBelow('1.2.0-7', '1.2.0-beta.8')).toBe(true);
    expect(isPolicyMandatory(
      { ...policy, minimumSupportedVersion: '1.2.0-beta.8' },
      Date.now(),
      '1.2.0-beta.7'
    )).toBe(true);
  });

  it('orders stable releases after prereleases and normalizes Windows beta versions', () => {
    expect(isVersionBelow('1.2.0-beta.7', '1.2.0')).toBe(true);
    expect(isVersionBelow('1.2.0', '1.2.0-beta.7')).toBe(false);
    expect(isVersionBelow('1.2.0-7', '1.2.0-beta.7')).toBe(false);
    expect(isVersionBelow('v1.2.0-7', '1.2.0-beta.8')).toBe(true);
  });
});
