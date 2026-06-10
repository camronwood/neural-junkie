import { useCallback, useState } from 'react';
import { open } from '@tauri-apps/api/dialog';
import { invoke } from '@tauri-apps/api/tauri';
import { usePacksStore } from '../../stores/packsStore';
import { isTauriRuntime } from '../../utils/promptAttachments';
import { ipcWorkspaceRoots, registerPackPickerPath } from '../../utils/ipcWorkspaceRoots';

function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'string') return e;
  if (e && typeof e === 'object' && 'message' in e && typeof (e as { message: unknown }).message === 'string') {
    return (e as { message: string }).message;
  }
  return 'Install failed';
}

export function CustomPackInstall() {
  const installPackFromZip = usePacksStore((s) => s.installPackFromZip);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleInstall = useCallback(async () => {
    if (!isTauriRuntime()) {
      setError('Custom pack install requires the desktop app.');
      return;
    }
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const selected = await open({
        multiple: false,
        filters: [{ name: 'Pack zip', extensions: ['zip'] }],
        title: 'Select customer pack zip',
      });
      if (!selected || typeof selected !== 'string') {
        return;
      }
      await registerPackPickerPath(selected);
      const base64 = await invoke<string>('read_pack_zip_base64', {
        absolutePath: selected,
        ...ipcWorkspaceRoots(),
      });
      const result = await installPackFromZip(base64);
      const pack = result.packs?.find((p) => p.id === result.pack_id);
      setMessage(
        pack
          ? `Installed ${pack.title} (${pack.id}). Enable required domain packs first, then enable this pack.`
          : `Installed pack ${result.pack_id ?? 'unknown'}.`
      );
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [installPackFromZip]);

  return (
    <div className="border border-teal-700/40 rounded-xl p-4 bg-teal-950/20 space-y-3">
      <div>
        <h4 className="text-sm font-semibold text-teal-200">Custom / customer packs</h4>
        <p className="text-xs text-gray-400 mt-1">
          Install a private pack zip from your organization (workspace SOPs, data layout, optional tool paths).
          Use together with official packs such as <strong className="text-gray-300">Life sciences</strong> — customer
          packs do not replace domain analysis features.
        </p>
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
      {message && <p className="text-xs text-teal-300">{message}</p>}
      <button
        type="button"
        disabled={busy}
        onClick={() => void handleInstall()}
        className="px-3 py-1.5 text-xs font-medium rounded-lg border border-teal-600/50 text-teal-200 hover:bg-teal-900/40 disabled:opacity-40"
      >
        {busy ? 'Installing…' : 'Install custom pack (zip…)'}
      </button>
    </div>
  );
}
