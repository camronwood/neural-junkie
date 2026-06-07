import { useState, useEffect, useCallback } from 'react';
import {
  checkForAppUpdate,
  installAppUpdate,
  UPDATE_CHECK_INTERVAL_MS,
  BANNER_AUTO_DISMISS_MS,
  type AppUpdateInfo,
} from '../utils/appUpdater';

export function UpdateBanner() {
  const [update, setUpdate] = useState<AppUpdateInfo | null>(null);
  const [downloading, setDownloading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [dismissed, setDismissed] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const runUpdateCheck = useCallback(async () => {
    const result = await checkForAppUpdate();
    if (result.status === 'available') {
      setUpdate(result.update);
      setError(null);
    } else if (result.status === 'unavailable') {
      if (result.reason !== 'Updates are only available in the desktop app.') {
        setError(result.reason);
      } else {
        setError(null);
      }
    } else {
      setError(null);
    }
  }, []);

  useEffect(() => {
    void runUpdateCheck();

    const interval = window.setInterval(() => {
      if (document.visibilityState === 'visible') {
        void runUpdateCheck();
      }
    }, UPDATE_CHECK_INTERVAL_MS);

    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        void runUpdateCheck();
      }
    };
    document.addEventListener('visibilitychange', onVisible);

    return () => {
      window.clearInterval(interval);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [runUpdateCheck]);

  useEffect(() => {
    if (downloading) return;

    const showError = Boolean(error && !update?.available);
    const showUpdate = Boolean(update?.available && !dismissed);
    if (!showError && !showUpdate) return;

    const timer = window.setTimeout(() => {
      if (showError) setError(null);
      if (showUpdate) setDismissed(true);
    }, BANNER_AUTO_DISMISS_MS);

    return () => window.clearTimeout(timer);
  }, [error, update?.available, dismissed, downloading]);

  async function installUpdate() {
    setDownloading(true);
    setError(null);
    try {
      await installAppUpdate(setProgress);
    } catch (e) {
      console.error('Update failed:', e);
      setError(e instanceof Error ? e.message : 'Update failed');
      setDownloading(false);
    }
  }

  if (error && !update?.available) {
    return (
      <div className="flex items-center justify-between px-4 py-2 bg-amber-700/90 text-white text-sm">
        <span className="text-xs">Could not check for updates: {error}</span>
        <button
          onClick={() => {
            setError(null);
            void runUpdateCheck();
          }}
          className="px-2 py-1 text-amber-100 hover:text-white text-xs"
        >
          Retry
        </button>
      </div>
    );
  }

  if (!update?.available || dismissed) return null;

  return (
    <div className="flex items-center justify-between px-4 py-2 bg-blue-600/90 text-white text-sm">
      <div className="flex items-center gap-2">
        <span className="font-medium">
          Neural Junkie {update.version} is available
        </span>
        {update.notes && (
          <span className="text-blue-200 text-xs truncate max-w-md">
            — {update.notes}
          </span>
        )}
        {error && <span className="text-red-200 text-xs">— {error}</span>}
      </div>
      <div className="flex items-center gap-2">
        {downloading ? (
          <span className="text-xs">Downloading... {progress}%</span>
        ) : (
          <>
            <button
              onClick={() => void installUpdate()}
              className="px-3 py-1 bg-white text-blue-600 rounded text-xs font-medium hover:bg-blue-50"
            >
              Update Now
            </button>
            <button
              onClick={() => setDismissed(true)}
              className="px-2 py-1 text-blue-200 hover:text-white text-xs"
            >
              Later
            </button>
          </>
        )}
      </div>
    </div>
  );
}
