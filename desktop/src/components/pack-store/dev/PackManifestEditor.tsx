import { useCallback, useEffect, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import { invoke } from '@tauri-apps/api/core';
import { open } from '@tauri-apps/plugin-dialog';
import { usePacksStore } from '../../../stores/packsStore';
import { isTauriRuntime } from '../../../utils/promptAttachments';
import { ipcWorkspaceRoots, registerPackPickerPath } from '../../../utils/ipcWorkspaceRoots';
import { useSettingsStore } from '../../../stores/settingsStore';
import { getMonacoThemeId, registerMonacoThemes } from '../../../utils/editorThemes';
import { MANIFEST_FIELD_HINTS } from './packDevConstants';
import { PackValidatePreview } from './PackValidatePreview';
import type { PackValidationReport } from '../../../api/chatAPI';
import { registerRestartBlocker } from '../../../utils/restartSafety';

function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'string') return e;
  return 'Action failed';
}

interface PackManifestEditorProps {
  packDir: string | null;
  initialYaml?: string;
  onPackDirChange: (dir: string | null) => void;
}

export function PackManifestEditor({ packDir, initialYaml, onPackDirChange }: PackManifestEditorProps) {
  const colorTheme = useSettingsStore((s) => s.settings.colorTheme ?? 'slack');
  const validatePack = usePacksStore((s) => s.validatePack);
  const devLinkPack = usePacksStore((s) => s.devLinkPack);
  const installPackFromZip = usePacksStore((s) => s.installPackFromZip);
  const [yaml, setYaml] = useState(initialYaml ?? '');
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [report, setReport] = useState<PackValidationReport | null>(null);
  const [validating, setValidating] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (initialYaml) {
      setYaml(initialYaml);
      setDirty(false);
    }
  }, [initialYaml]);

  const runValidate = useCallback(
    async (text: string, dir: string | null) => {
      setValidating(true);
      setError(null);
      try {
        const result = await validatePack({
          pack_yaml: text,
          pack_dir: dir ?? undefined,
        });
        setReport(result);
      } catch (e) {
        setError(errorMessage(e));
        setReport(null);
      } finally {
        setValidating(false);
      }
    },
    [validatePack],
  );

  useEffect(() => {
    if (!yaml.trim()) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      void runValidate(yaml, packDir);
    }, 600);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [yaml, packDir, runValidate]);

  const openFolder = useCallback(async () => {
    if (!isTauriRuntime()) {
      setError('Editor requires the desktop app.');
      return;
    }
    const selected = await open({
      directory: true,
      multiple: false,
      title: 'Open pack folder',
    });
    if (!selected || typeof selected !== 'string') return;
    await registerPackPickerPath(selected);
    onPackDirChange(selected);
    try {
      const content = await invoke<string>('read_pack_yaml_from_dir', {
        absoluteDir: selected,
        ...ipcWorkspaceRoots(),
      });
      setYaml(content);
      setDirty(false);
    } catch (e) {
      setError(errorMessage(e));
    }
  }, [onPackDirChange]);

  const saveYaml = useCallback(async () => {
    if (!packDir) {
      setError('Open or scaffold a pack folder first.');
      return false;
    }
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      await invoke('write_pack_yaml_to_dir', {
        absoluteDir: packDir,
        yaml,
        ...ipcWorkspaceRoots(),
      });
      setDirty(false);
      setMessage('Saved pack.yaml');
      await runValidate(yaml, packDir);
      return true;
    } catch (e) {
      setError(errorMessage(e));
      return false;
    } finally {
      setBusy(false);
    }
  }, [packDir, yaml, runValidate]);

  useEffect(
    () =>
      registerRestartBlocker('pack-manifest-editor', () =>
        dirty
          ? {
              id: 'pack-manifest-editor',
              message: 'Unsaved pack manifest changes must be saved before restarting.',
              ...(packDir ? { save: saveYaml } : {}),
            }
          : null
      ),
    [dirty, packDir, saveYaml]
  );

  const linkAndSync = useCallback(async () => {
    if (!packDir) {
      setError('Open a pack folder first.');
      return;
    }
    if (dirty) await saveYaml();
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      await devLinkPack(packDir);
      setMessage('Linked and synced to hub.');
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [packDir, dirty, saveYaml, devLinkPack]);

  const validateZip = useCallback(async () => {
    if (!packDir) {
      setError('Open a pack folder first.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const base64 = await invoke<string>('zip_pack_directory', {
        absoluteDir: packDir,
        ...ipcWorkspaceRoots(),
      });
      const result = await validatePack({ pack_zip_base64: base64 });
      setReport(result);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [packDir, validatePack]);

  const installZipSmoke = useCallback(async () => {
    if (!packDir) return;
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const base64 = await invoke<string>('zip_pack_directory', {
        absoluteDir: packDir,
        ...ipcWorkspaceRoots(),
      });
      const result = await installPackFromZip(base64);
      setMessage(`Installed ${result.pack_id ?? 'pack'} from release zip.`);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [packDir, installPackFromZip]);

  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_280px]">
      <div className="space-y-3">
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => void openFolder()}
            className="px-3 py-1.5 text-xs rounded border border-slack-border text-slack-text hover:bg-slack-bgHover"
          >
            Open folder…
          </button>
          <button
            type="button"
            disabled={busy || !packDir}
            onClick={() => void saveYaml()}
            className="px-3 py-1.5 text-xs rounded border border-slack-border text-slack-text hover:bg-slack-bgHover disabled:opacity-40"
          >
            Save
          </button>
          <button
            type="button"
            disabled={busy || !packDir}
            onClick={() => void linkAndSync()}
            className="px-3 py-1.5 text-xs rounded border border-teal-600/50 text-teal-200 hover:bg-teal-900/40 disabled:opacity-40"
          >
            Link & sync
          </button>
          <button
            type="button"
            disabled={busy || !packDir}
            onClick={() => void validateZip()}
            className="px-3 py-1.5 text-xs rounded border border-slack-border text-slack-text hover:bg-slack-bgHover disabled:opacity-40"
          >
            Validate zip
          </button>
          <button
            type="button"
            disabled={busy || !packDir}
            onClick={() => void installZipSmoke()}
            className="px-3 py-1.5 text-xs rounded border border-slack-border text-slack-textMuted hover:bg-slack-bgHover disabled:opacity-40"
          >
            Install zip (smoke)
          </button>
        </div>
        {packDir && (
          <p className="text-[11px] font-mono text-slack-textMuted break-all">{packDir}</p>
        )}
        {error && <p className="text-xs text-red-400">{error}</p>}
        {message && <p className="text-xs text-teal-300">{message}</p>}
        <div className="border border-slack-border rounded-lg overflow-hidden min-h-[280px]">
          <Editor
            height="280px"
            language="yaml"
            theme={getMonacoThemeId(colorTheme)}
            value={yaml}
            onChange={(v) => {
              setYaml(v ?? '');
              setDirty(true);
            }}
            beforeMount={registerMonacoThemes}
            options={{
              minimap: { enabled: false },
              fontSize: 13,
              wordWrap: 'on',
              scrollBeyondLastLine: false,
            }}
          />
        </div>
      </div>
      <div className="space-y-4">
        <div>
          <p className="text-xs font-medium text-slack-text mb-2">Field hints</p>
          <ul className="text-[11px] text-slack-textMuted space-y-1 list-disc pl-4">
            {MANIFEST_FIELD_HINTS.map((h) => (
              <li key={h}>{h}</li>
            ))}
          </ul>
        </div>
        <div>
          <p className="text-xs font-medium text-slack-text mb-2">Live validation</p>
          <PackValidatePreview report={report} loading={validating} />
        </div>
      </div>
    </div>
  );
}
