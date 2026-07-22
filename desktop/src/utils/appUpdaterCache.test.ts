import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { AppUpdateInfo } from './appUpdater';
import {
  clearAcceptedUpdate,
  loadAcceptedUpdate,
  saveAcceptedUpdate,
} from './appUpdaterCache';

const update: AppUpdateInfo = {
  available: true,
  version: '1.3.0',
  notes: 'Release notes',
  mandatory: false,
  policy: {
    schemaVersion: 1,
    channel: 'stable',
    severity: 'critical',
    enforcement: 'mandatory',
    mandatoryAfter: '2026-07-22T00:00:00Z',
    rolloutPercentage: 100,
    rolloutSeed: 'v1.3.0',
  },
};

describe('accepted update cache', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useRealTimers();
  });

  it('hydrates accepted metadata without claiming installable bytes', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-21T12:00:00Z'));
    saveAcceptedUpdate(update);
    expect(loadAcceptedUpdate('1.2.0', 'stable')).toMatchObject({
      version: '1.3.0',
      mandatory: false,
    });
  });

  it('recomputes mandatory policy using the current clock', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-21T12:00:00Z'));
    saveAcceptedUpdate(update);
    vi.setSystemTime(new Date('2026-07-22T00:00:00Z'));
    expect(loadAcceptedUpdate('1.2.0', 'stable')?.mandatory).toBe(true);
  });

  it('clears evidence after installation or on channel/version mismatch', () => {
    saveAcceptedUpdate(update);
    expect(loadAcceptedUpdate('1.2.0', 'beta')).toBeNull();

    saveAcceptedUpdate(update);
    expect(loadAcceptedUpdate('1.3.0', 'stable')).toBeNull();

    saveAcceptedUpdate(update);
    clearAcceptedUpdate();
    expect(loadAcceptedUpdate('1.2.0', 'stable')).toBeNull();
  });

  it('fails open for malformed local data', () => {
    localStorage.setItem('nj-updater-accepted-update-v1', '{"schemaVersion":1}');
    expect(loadAcceptedUpdate('1.2.0', 'stable')).toBeNull();
  });
});
