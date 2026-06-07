import { describe, expect, it } from 'vitest';
import { getUpdateChannelLabel } from './appUpdater';

describe('getUpdateChannelLabel', () => {
  it('returns Beta for beta semver', () => {
    expect(getUpdateChannelLabel('1.0.0-beta.25')).toBe('Beta updates');
  });

  it('returns Stable for release semver', () => {
    expect(getUpdateChannelLabel('1.0.0')).toBe('Stable updates');
  });
});
