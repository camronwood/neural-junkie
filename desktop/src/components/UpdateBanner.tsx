import { useEffect } from 'react';
import { UPDATE_CHECK_INTERVAL_MS } from '../utils/appUpdater';
import { useAppUpdaterStore } from '../stores/appUpdaterStore';

export function UpdateBanner() {
  const status = useAppUpdaterStore((state) => state.status);
  const update = useAppUpdaterStore((state) => state.update);
  const progress = useAppUpdaterStore((state) => state.progress);
  const error = useAppUpdaterStore((state) => state.error);
  const blockers = useAppUpdaterStore((state) => state.blockers);
  const check = useAppUpdaterStore((state) => state.check);
  const restartToUpdate = useAppUpdaterStore((state) => state.restartToUpdate);

  useEffect(() => {
    void check();

    const interval = window.setInterval(() => {
      if (document.visibilityState === 'visible') {
        void check();
      }
    }, UPDATE_CHECK_INTERVAL_MS);

    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        void check();
      }
    };
    document.addEventListener('visibilitychange', onVisible);

    return () => {
      window.clearInterval(interval);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [check]);

  useEffect(() => {
    if (status !== 'ready') return;
    let disposed = false;
    let unlisten: (() => void) | undefined;
    void import('@tauri-apps/api/window').then(async ({ getCurrentWindow }) => {
      const stopListening = await getCurrentWindow().onCloseRequested((event) => {
        // The first close attempt discovers restart blockers. A second attempt
        // must still allow an explicit quit without trapping the user.
        if (useAppUpdaterStore.getState().blockers.length > 0) return;
        event.preventDefault();
        void restartToUpdate();
      });
      if (disposed) stopListening();
      else unlisten = stopListening;
    });
    return () => {
      disposed = true;
      unlisten?.();
    };
  }, [status, restartToUpdate]);

  if (status === 'error' && !update) {
    return (
      <div className="flex items-center justify-between px-4 py-2 bg-amber-700/90 text-white text-sm">
        <span className="text-xs">Could not check for updates: {error}</span>
        <button
          onClick={() => void check(true)}
          className="px-2 py-1 text-amber-100 hover:text-white text-xs"
        >
          Retry
        </button>
      </div>
    );
  }

  if (!update || !['downloading', 'waiting', 'ready', 'installing', 'error'].includes(status)) return null;

  const hardGate = update.mandatory && (status === 'ready' || status === 'installing') && blockers.length === 0;
  if (hardGate) {
    return (
      <div className="fixed inset-0 z-[100] bg-black/75 flex items-center justify-center p-6">
        <div className="max-w-lg w-full rounded-lg bg-slack-sidebar border border-slack-border p-6 text-white shadow-2xl">
          <h2 className="text-xl font-semibold">Neural Junkie {update.version} is required</h2>
          <p className="mt-2 text-sm text-slack-textMuted">
            This critical update has reached its enforcement deadline.
          </p>
          {update.notes && <p className="mt-3 text-sm">{update.notes}</p>}
          <div className="mt-5">
            {status === 'ready' && (
              <div className="flex gap-3">
                <button
                  onClick={() => void restartToUpdate()}
                  className="px-4 py-2 bg-blue-600 rounded font-medium hover:bg-blue-700"
                >
                  Restart and update
                </button>
                <button
                  onClick={() => {
                    void import('@tauri-apps/api/window').then(({ getCurrentWindow }) =>
                      getCurrentWindow().destroy()
                    );
                  }}
                  className="px-4 py-2 border border-slack-border rounded font-medium hover:bg-slack-bgHover"
                >
                  Quit
                </button>
              </div>
            )}
            {status === 'installing' && <span>Installing update…</span>}
            {error && <p className="mt-3 text-sm text-red-300">{error}</p>}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-between px-4 py-2 bg-blue-600/90 text-white text-sm">
      <div className="flex items-center gap-2">
        <span className="font-medium">
          Neural Junkie {update.version} {update.mandatory ? 'is required' : 'is available'}
        </span>
        {update.notes && (
          <span className="text-blue-200 text-xs truncate max-w-md">
            — {update.notes}
          </span>
        )}
        {error && <span className="text-red-200 text-xs">— {error}</span>}
      </div>
      <div className="flex items-center gap-2">
        {status === 'downloading' && (
          <span className="text-xs">Downloading… {progress === null ? '' : `${progress}%`}</span>
        )}
        {status === 'waiting' && (
          <>
            <span className="text-xs">
              {error ? 'Update download is paused.' : 'Connect to download the signed update.'}
            </span>
            <button
              onClick={() => void check(true)}
              className="px-3 py-1 bg-white text-blue-600 rounded text-xs font-medium hover:bg-blue-50"
            >
              Retry
            </button>
          </>
        )}
        {status === 'ready' && (
          <button
            onClick={() => void restartToUpdate()}
            className="px-3 py-1 bg-white text-blue-600 rounded text-xs font-medium hover:bg-blue-50"
          >
            Restart to update
          </button>
        )}
        {status === 'installing' && <span className="text-xs">Installing…</span>}
        {blockers.map((blocker) => (
          <span key={blocker.id} className="text-xs text-amber-100">{blocker.message}</span>
        ))}
      </div>
    </div>
  );
}
