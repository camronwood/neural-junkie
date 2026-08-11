import { useEffect } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { useLocationShareStore } from '../stores/locationShareStore';
import { usePacksStore } from '../stores/packsStore';

interface LocationRequestModalProps {
  api: ChatAPI;
}

export function LocationRequestModal({ api }: LocationRequestModalProps) {
  const mapsOn = usePacksStore((s) => s.hasCapability('maps-tools') || s.hasCapability('maps-location'));
  const syncPending = useLocationShareStore((s) => s.syncPending);

  useEffect(() => {
    if (!mapsOn) return;
    void syncPending(api);
    const id = window.setInterval(() => {
      void syncPending(api);
    }, 2000);
    return () => window.clearInterval(id);
  }, [api, mapsOn, syncPending]);

  const pending = useLocationShareStore((s) => s.pendingRequests[0]);
  const busy = useLocationShareStore((s) => s.busy);
  const error = useLocationShareStore((s) => s.error);
  const fulfillPending = useLocationShareStore((s) => s.fulfillPending);
  const rejectPending = useLocationShareStore((s) => s.rejectPending);

  if (!pending) return null;

  return (
    <div className="fixed inset-0 z-[300] flex items-center justify-center p-4" role="presentation">
      <div className="absolute inset-0 bg-black/60" aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="location-request-title"
        className="relative z-10 w-full max-w-md rounded-xl border border-slack-border bg-slack-bg shadow-2xl"
      >
        <div className="px-5 py-4 border-b border-slack-border">
          <h2 id="location-request-title" className="text-base font-semibold text-slack-text">
            Share a fresh location reading?
          </h2>
          <p className="mt-2 text-sm text-slack-textMuted leading-relaxed">
            {pending.agentName || 'Assistant'} wants your current device location for this reply.
            Coordinates are not saved in chat history.
          </p>
        </div>
        {error ? <p className="px-5 pt-3 text-xs text-amber-300">{error}</p> : null}
        <div className="px-5 py-4 flex justify-end gap-2">
          <button
            type="button"
            disabled={busy}
            className="px-3 py-1.5 rounded-md text-sm border border-slack-border text-slack-textMuted hover:text-slack-text"
            onClick={() => {
              void rejectPending(api, pending.id);
            }}
          >
            Deny
          </button>
          <button
            type="button"
            disabled={busy}
            className="px-3 py-1.5 rounded-md text-sm bg-sky-700 hover:bg-sky-600 text-white disabled:opacity-60"
            onClick={() => {
              void fulfillPending(api, pending.id);
            }}
          >
            {busy ? 'Reading…' : 'Share location'}
          </button>
        </div>
      </div>
    </div>
  );
}
