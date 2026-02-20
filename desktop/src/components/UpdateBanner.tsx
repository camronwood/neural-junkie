import { useState, useEffect } from 'react';

interface UpdateInfo {
  available: boolean;
  version?: string;
  notes?: string;
}

export function UpdateBanner() {
  const [update, setUpdate] = useState<UpdateInfo | null>(null);
  const [downloading, setDownloading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    checkForUpdate();
  }, []);

  async function checkForUpdate() {
    try {
      const { checkUpdate } = await import('@tauri-apps/api/updater');
      const result = await checkUpdate();
      if (result.shouldUpdate) {
        setUpdate({
          available: true,
          version: result.manifest?.version,
          notes: result.manifest?.body,
        });
      }
    } catch {
      // Updater not configured or check failed -- silent
    }
  }

  async function installUpdate() {
    setDownloading(true);
    try {
      const { listen } = await import('@tauri-apps/api/event');
      const { installUpdate: install } = await import('@tauri-apps/api/updater');
      const { relaunch } = await import('@tauri-apps/api/process');

      const unlisten = await listen<{ chunkLength: number; contentLength: number }>(
        'tauri://update-download-progress',
        (event) => {
          if (event.payload.contentLength > 0) {
            setProgress(Math.round((event.payload.chunkLength / event.payload.contentLength) * 100));
          }
        }
      );

      await install();
      unlisten();
      await relaunch();
    } catch (e) {
      console.error('Update failed:', e);
      setDownloading(false);
    }
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
      </div>
      <div className="flex items-center gap-2">
        {downloading ? (
          <span className="text-xs">Downloading... {progress}%</span>
        ) : (
          <>
            <button
              onClick={installUpdate}
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
