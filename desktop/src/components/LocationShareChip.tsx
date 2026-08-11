import { ChatAPI } from '../api/chatAPI';
import { useLocationShareStore } from '../stores/locationShareStore';
import { usePacksStore } from '../stores/packsStore';

interface LocationShareChipProps {
  api: ChatAPI;
}

export function LocationShareChip({ api }: LocationShareChipProps) {
  const mapsOn = usePacksStore((s) => s.hasCapability('maps-tools') || s.hasCapability('maps-location'));
  const sharing = useLocationShareStore((s) => s.sharing);
  const snapshot = useLocationShareStore((s) => s.snapshot);
  const error = useLocationShareStore((s) => s.error);
  const busy = useLocationShareStore((s) => s.busy);
  const startSharing = useLocationShareStore((s) => s.startSharing);
  const stopSharing = useLocationShareStore((s) => s.stopSharing);

  if (!mapsOn) return null;

  const label = sharing
    ? snapshot?.display_name
      ? `Sharing · ${snapshot.display_name}`
      : 'Sharing location'
    : 'Share location';

  return (
    <div className="mx-3 mb-1 flex flex-col gap-1">
      <button
        type="button"
        disabled={busy}
        onClick={() => {
          if (sharing) {
            void stopSharing(api);
            return;
          }
          void startSharing(api);
        }}
        className={`self-start px-2.5 py-1 rounded-md text-xs border transition-colors ${
          sharing
            ? 'border-sky-600/70 bg-sky-950/50 text-sky-100'
            : 'border-slack-border bg-slack-bgHover/40 text-slack-textMuted hover:text-slack-text'
        } disabled:opacity-60`}
        title={
          sharing
            ? 'Stop sharing this device location with Assistant for this session'
            : 'Share this device location with Assistant for this session'
        }
      >
        {busy ? 'Reading location…' : label}
      </button>
      {error ? <p className="text-xs text-amber-300">{error}</p> : null}
    </div>
  );
}
