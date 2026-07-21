import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { AppUpdateInfo } from '../utils/appUpdater';

const mocks = vi.hoisted(() => ({
  checkForAppUpdate: vi.fn(),
  downloadAppUpdate: vi.fn(),
  installDownloadedAppUpdate: vi.fn(),
  loadAcceptedUpdate: vi.fn(),
  saveAcceptedUpdate: vi.fn(),
  clearAcceptedUpdate: vi.fn(),
  saveRestartBlockers: vi.fn(),
  getVersion: vi.fn(async () => '1.2.0'),
}));

vi.mock('../utils/appUpdater', () => ({
  UPDATE_CHECK_INTERVAL_MS: 6 * 60 * 60 * 1000,
  checkForAppUpdate: mocks.checkForAppUpdate,
  downloadAppUpdate: mocks.downloadAppUpdate,
  installDownloadedAppUpdate: mocks.installDownloadedAppUpdate,
}));
vi.mock('../utils/appUpdaterCache', () => ({
  loadAcceptedUpdate: mocks.loadAcceptedUpdate,
  saveAcceptedUpdate: mocks.saveAcceptedUpdate,
  clearAcceptedUpdate: mocks.clearAcceptedUpdate,
}));
vi.mock('../utils/restartSafety', () => ({
  saveRestartBlockers: mocks.saveRestartBlockers,
}));
vi.mock('@tauri-apps/api/app', () => ({ getVersion: mocks.getVersion }));

import { useAppUpdaterStore } from './appUpdaterStore';

const update: AppUpdateInfo = {
  available: true,
  version: '1.3.0',
  mandatory: true,
  policy: {
    schemaVersion: 1,
    channel: 'stable',
    severity: 'critical',
    enforcement: 'mandatory',
    rolloutPercentage: 100,
    rolloutSeed: 'v1.3.0',
  },
};

describe('app updater controller', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mocks.loadAcceptedUpdate.mockReturnValue(null);
    mocks.saveRestartBlockers.mockResolvedValue([]);
    useAppUpdaterStore.setState({
      status: 'idle',
      update: null,
      progress: null,
      error: null,
      blockers: [],
    });
  });

  it('keeps an offline mandatory cached update advisory', async () => {
    mocks.loadAcceptedUpdate.mockReturnValue(update);
    mocks.checkForAppUpdate.mockResolvedValue({
      status: 'unavailable',
      reason: 'offline',
    });

    await useAppUpdaterStore.getState().check(true);

    expect(useAppUpdaterStore.getState()).toMatchObject({
      status: 'waiting',
      update,
      error: 'offline',
    });
  });

  it('becomes ready only after the signed download resolves', async () => {
    mocks.checkForAppUpdate.mockResolvedValue({ status: 'available', update });
    mocks.downloadAppUpdate.mockImplementation(
      async (onProgress?: (progress: number | null) => void) => onProgress?.(100)
    );

    await useAppUpdaterStore.getState().check(true);

    expect(mocks.saveAcceptedUpdate).toHaveBeenCalledWith(update);
    expect(useAppUpdaterStore.getState()).toMatchObject({
      status: 'ready',
      progress: 100,
    });
  });

  it('returns to waiting when download verification fails', async () => {
    mocks.checkForAppUpdate.mockResolvedValue({ status: 'available', update });
    mocks.downloadAppUpdate.mockRejectedValue(new Error('invalid signature'));

    await useAppUpdaterStore.getState().check(true);

    expect(mocks.saveAcceptedUpdate).not.toHaveBeenCalled();
    expect(useAppUpdaterStore.getState()).toMatchObject({
      status: 'waiting',
      update,
      error: 'invalid signature',
    });
  });
});
