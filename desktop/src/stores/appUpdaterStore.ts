import { create } from 'zustand';
import {
  UPDATE_CHECK_INTERVAL_MS,
  checkForAppUpdate,
  downloadAppUpdate,
  installDownloadedAppUpdate,
  type AppUpdateInfo,
} from '../utils/appUpdater';
import {
  clearAcceptedUpdate,
  loadAcceptedUpdate,
  saveAcceptedUpdate,
} from '../utils/appUpdaterCache';
import { saveRestartBlockers, type RestartBlocker } from '../utils/restartSafety';

const LAST_CHECK_KEY = 'nj-updater-last-successful-check-v1';

export type AppUpdaterStatus =
  | 'idle'
  | 'checking'
  | 'current'
  | 'downloading'
  | 'waiting'
  | 'ready'
  | 'installing'
  | 'error'
  | 'unsupported';

interface AppUpdaterState {
  status: AppUpdaterStatus;
  update: AppUpdateInfo | null;
  progress: number | null;
  error: string | null;
  blockers: RestartBlocker[];
  check: (force?: boolean) => Promise<void>;
  restartToUpdate: () => Promise<boolean>;
}

let checkPromise: Promise<void> | null = null;

function checkedRecently(): boolean {
  const value = Number(localStorage.getItem(LAST_CHECK_KEY));
  return Number.isFinite(value) && Date.now() - value < UPDATE_CHECK_INTERVAL_MS;
}

export const useAppUpdaterStore = create<AppUpdaterState>((set, get) => ({
  status: 'idle',
  update: null,
  progress: null,
  error: null,
  blockers: [],

  check: async (force = false) => {
    if (checkPromise) return checkPromise;
    if (['downloading', 'ready', 'installing'].includes(get().status)) {
      return;
    }
    checkPromise = (async () => {
      let cachedUpdate: AppUpdateInfo | null = null;
      try {
        const { getVersion } = await import('@tauri-apps/api/app');
        const currentVersion = await getVersion();
        const configuredChannel = import.meta.env.VITE_UPDATE_CHANNEL;
        const channel =
          configuredChannel === 'beta' || configuredChannel === 'stable'
            ? configuredChannel
            : currentVersion.includes('-beta')
              ? 'beta'
              : 'stable';
        cachedUpdate = loadAcceptedUpdate(currentVersion, channel);
      } catch {
        // Browser/dev mode and storage failures must not block startup.
      }

      if (!force && checkedRecently() && get().status === 'idle') {
        set({
          status: cachedUpdate ? 'waiting' : 'current',
          update: cachedUpdate,
          progress: null,
        });
        return;
      }

      set({
        status: 'checking',
        error: null,
        blockers: [],
        update: cachedUpdate,
      });
      const result = await checkForAppUpdate();
      if (result.status === 'current' || result.status === 'deferred') {
        localStorage.setItem(LAST_CHECK_KEY, String(Date.now()));
        set({
          status: cachedUpdate ? 'waiting' : 'current',
          update: cachedUpdate,
          progress: null,
        });
        return;
      }
      if (result.status === 'unavailable') {
        const unsupported = result.reason.includes('Linux');
        set({
          status: cachedUpdate ? 'waiting' : unsupported ? 'unsupported' : 'error',
          error: result.reason,
          update: cachedUpdate,
          progress: null,
        });
        return;
      }

      set({ status: 'downloading', update: result.update, progress: 0 });
      try {
        await downloadAppUpdate((progress) => set({ progress }));
        saveAcceptedUpdate(result.update);
        localStorage.setItem(LAST_CHECK_KEY, String(Date.now()));
        set({ status: 'ready', progress: 100 });
      } catch (error) {
        set({
          status: 'waiting',
          error: error instanceof Error ? error.message : 'Update download failed',
          progress: null,
        });
      }
    })().finally(() => {
      checkPromise = null;
    });
    return checkPromise;
  },

  restartToUpdate: async () => {
    if (get().status !== 'ready') return false;
    try {
      const blockers = await saveRestartBlockers();
      if (blockers.length > 0) {
        set({ blockers, error: 'Finish active work before restarting.' });
        return false;
      }
      set({ status: 'installing', error: null, blockers: [] });
      await installDownloadedAppUpdate();
      clearAcceptedUpdate();
      return true;
    } catch (error) {
      set({
        status: 'error',
        error: error instanceof Error ? error.message : 'Update installation failed',
      });
      return false;
    }
  },
}));
