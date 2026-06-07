export const UPDATE_CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000; // 6 hours
export const BANNER_AUTO_DISMISS_MS = 15_000; // 15 seconds

export interface AppUpdateInfo {
  available: boolean;
  version?: string;
  notes?: string;
}

export type UpdateCheckResult =
  | { status: 'available'; update: AppUpdateInfo }
  | { status: 'current' }
  | { status: 'unavailable'; reason: string };

function isTauriRuntime(): boolean {
  return typeof window !== 'undefined' && Boolean((window as Window & { __TAURI__?: unknown }).__TAURI__);
}

export function getUpdateChannelLabel(version: string): string {
  return version.includes('-beta') ? 'Beta updates' : 'Stable updates';
}

export async function checkForAppUpdate(): Promise<UpdateCheckResult> {
  if (!isTauriRuntime()) {
    return { status: 'unavailable', reason: 'Updates are only available in the desktop app.' };
  }

  try {
    const { checkUpdate } = await import('@tauri-apps/api/updater');
    const result = await checkUpdate();
    if (result.shouldUpdate) {
      return {
        status: 'available',
        update: {
          available: true,
          version: result.manifest?.version,
          notes: result.manifest?.body,
        },
      };
    }
    return { status: 'current' };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Update check failed';
    return { status: 'unavailable', reason: message };
  }
}

export async function installAppUpdate(
  onProgress?: (percent: number) => void
): Promise<void> {
  const { listen } = await import('@tauri-apps/api/event');
  const { installUpdate } = await import('@tauri-apps/api/updater');
  const { relaunch } = await import('@tauri-apps/api/process');

  const unlisten = await listen<{ chunkLength: number; contentLength: number }>(
    'tauri://update-download-progress',
    (event) => {
      if (event.payload.contentLength > 0 && onProgress) {
        onProgress(Math.round((event.payload.chunkLength / event.payload.contentLength) * 100));
      }
    }
  );

  try {
    await installUpdate();
  } finally {
    unlisten();
  }

  await relaunch();
}
