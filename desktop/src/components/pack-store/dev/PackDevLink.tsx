import { useCallback, useState } from 'react';
import { invoke } from '@tauri-apps/api/tauri';
import { usePacksStore } from '../../../stores/packsStore';
import { isTauriRuntime } from '../../../utils/promptAttachments';

function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'string') return e;
  return 'Action failed';
}

interface PackDevLinkProps {
  packDir: string | null;
  onPackDirChange: (dir: string | null) => void;
  onValidated?: () => void;
}

export function PackDevLink({ packDir, onPackDirChange, onValidated }: PackDevLinkProps) {
  const packs = usePacksStore((s) => s.packs);
  const devLinkPack = usePacksStore((s) => s.devLinkPack);
  const devReloadPack = usePacksStore((s) => s.devReloadPack);
  const devUnlinkPack = usePacksStore((s) => s.devUnlinkPack);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const linkedPack = packs.find((p) => p.dev_linked);

  const pickFolder = useCallback(async () => {
    if (!isTauriRuntime()) {
      setError('Dev link requires the desktop app.');
      return;
    }
    setError(null);
    try {
      const selected = await invoke<string | null>('pick_pack_directory', {
        title: 'Select pack folder (contains pack.yaml)',
      });
      if (selected) {
        onPackDirChange(selected);
      }
    } catch (e) {
      setError(errorMessage(e));
    }
  }, [onPackDirChange]);

  const handleLink = useCallback(async () => {
    if (!packDir) {
      setError('Select a pack folder first.');
      return;
    }
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const result = await devLinkPack(packDir);
      setMessage(`Linked and synced ${result.pack_id ?? 'pack'}. Enable required domain packs, then enable this pack.`);
      onValidated?.();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [packDir, devLinkPack, onValidated]);

  const handleReload = useCallback(async () => {
    if (!linkedPack) return;
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      await devReloadPack(linkedPack.id);
      setMessage(`Reloaded ${linkedPack.id} from dev source.`);
      onValidated?.();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [linkedPack, devReloadPack, onValidated]);

  const handleUnlink = useCallback(async () => {
    if (!linkedPack) return;
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      await devUnlinkPack(linkedPack.id);
      setMessage(`Unlinked dev source for ${linkedPack.id} (pack remains installed).`);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [linkedPack, devUnlinkPack]);

  return (
    <div className="space-y-3">
      <p className="text-xs text-slack-textMuted">
        Link a local folder for fast iteration. Changes sync on <strong className="text-slack-text">Reload</strong>{' '}
        without rebuilding a zip.
      </p>
      {error && <p className="text-xs text-red-400">{error}</p>}
      {message && <p className="text-xs text-teal-300">{message}</p>}
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void pickFolder()}
          className="px-3 py-1.5 text-xs font-medium rounded-lg border border-slack-border text-slack-text hover:bg-slack-bgHover disabled:opacity-40"
        >
          Choose folder…
        </button>
        <button
          type="button"
          disabled={busy || !packDir}
          onClick={() => void handleLink()}
          className="px-3 py-1.5 text-xs font-medium rounded-lg border border-teal-600/50 text-teal-200 hover:bg-teal-900/40 disabled:opacity-40"
        >
          {busy ? 'Linking…' : 'Link & sync'}
        </button>
        {linkedPack && (
          <>
            <button
              type="button"
              disabled={busy}
              onClick={() => void handleReload()}
              className="px-3 py-1.5 text-xs font-medium rounded-lg border border-slack-border text-slack-text hover:bg-slack-bgHover disabled:opacity-40"
            >
              Reload
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => void handleUnlink()}
              className="px-3 py-1.5 text-xs font-medium rounded-lg border border-slack-border text-slack-textMuted hover:bg-slack-bgHover disabled:opacity-40"
            >
              Unlink
            </button>
          </>
        )}
      </div>
      {packDir && <p className="text-[11px] font-mono text-slack-textMuted break-all">Folder: {packDir}</p>}
      {linkedPack?.dev_source_path && (
        <p className="text-[11px] font-mono text-teal-300/80 break-all">
          Dev-linked: {linkedPack.id} ← {linkedPack.dev_source_path}
        </p>
      )}
    </div>
  );
}
