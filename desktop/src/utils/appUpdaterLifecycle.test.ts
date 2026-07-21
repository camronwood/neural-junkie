import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  check: vi.fn(),
  invoke: vi.fn(),
  relaunch: vi.fn(),
  type: vi.fn(() => 'macos'),
}));

vi.mock('@tauri-apps/api/core', () => ({
  isTauri: () => true,
  invoke: mocks.invoke,
}));
vi.mock('@tauri-apps/plugin-os', () => ({ type: mocks.type }));
vi.mock('@tauri-apps/plugin-updater', () => ({ check: mocks.check }));
vi.mock('@tauri-apps/plugin-process', () => ({ relaunch: mocks.relaunch }));

import {
  checkForAppUpdate,
  downloadAppUpdate,
  installDownloadedAppUpdate,
} from './appUpdater';

function fakeUpdate() {
  return {
    version: '1.3.0',
    body: 'Release notes',
    rawJson: {
      policy: {
        schema_version: 1,
        channel: 'stable',
        rollout: { percentage: 100, seed: 'v1.3.0' },
      },
    },
    close: vi.fn(),
    download: vi.fn(async (onEvent) => {
      onEvent({ event: 'Started', data: { contentLength: 100 } });
      onEvent({ event: 'Progress', data: { chunkLength: 25 } });
      onEvent({ event: 'Progress', data: { chunkLength: 75 } });
      onEvent({ event: 'Finished' });
    }),
    install: vi.fn(),
  };
}

describe('Tauri v2 updater lifecycle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.type.mockReturnValue('macos');
  });

  it('checks, downloads with cumulative progress, prepares, installs, and relaunches', async () => {
    const update = fakeUpdate();
    mocks.check.mockResolvedValue(update);
    await expect(checkForAppUpdate()).resolves.toMatchObject({
      status: 'available',
      update: { version: '1.3.0' },
    });

    const progress: Array<number | null> = [];
    await downloadAppUpdate((value) => progress.push(value));
    expect(progress).toEqual([0, 25, 100, 100]);

    await installDownloadedAppUpdate();
    expect(mocks.invoke).toHaveBeenCalledWith('prepare_for_update');
    expect(update.install).toHaveBeenCalledOnce();
    expect(mocks.relaunch).toHaveBeenCalledOnce();
  });

  it('gates Linux until package upgrades are validated', async () => {
    mocks.type.mockReturnValue('linux');
    await expect(checkForAppUpdate()).resolves.toEqual({
      status: 'unavailable',
      reason: 'Automatic updates are not yet enabled for Linux packages.',
    });
    expect(mocks.check).not.toHaveBeenCalled();
  });
});
