import { useCallback, useEffect, useMemo, useState } from 'react';
import { invoke } from '@tauri-apps/api/tauri';
import { usePacksStore } from '../../../stores/packsStore';
import { PACK_CAP } from '../../../stores/packCapabilities';
import { isTauriRuntime } from '../../../utils/promptAttachments';
import { RichMarkdownView } from '../../RichMarkdownView';
import type { CustomerPackContext } from '../../../api/chatAPI';

function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'string') return e;
  return 'Action failed';
}

export function PackTestPanel() {
  const packs = usePacksStore((s) => s.packs);
  const setPackEnabled = usePacksStore((s) => s.setPackEnabled);
  const fetchCustomerPackContext = usePacksStore((s) => s.fetchCustomerPackContext);
  const installPackFromZip = usePacksStore((s) => s.installPackFromZip);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [contexts, setContexts] = useState<CustomerPackContext[]>([]);

  const customPacks = useMemo(
    () => packs.filter((p) => p.custom && p.installed),
    [packs],
  );

  const [selectedPackId, setSelectedPackId] = useState<string | null>(null);
  const targetPack = useMemo(() => {
    if (customPacks.length === 0) return undefined;
    if (selectedPackId) {
      return customPacks.find((p) => p.id === selectedPackId) ?? customPacks[0];
    }
    return customPacks.find((p) => p.dev_linked) ?? customPacks[0];
  }, [customPacks, selectedPackId]);

  const refreshContext = useCallback(async () => {
    try {
      const res = await fetchCustomerPackContext();
      setContexts(res.packs ?? []);
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
    }
  }, [fetchCustomerPackContext]);

  useEffect(() => {
    void refreshContext();
  }, [refreshContext, packs]);

  const toggleEnabled = useCallback(async () => {
    if (!targetPack) return;
    setBusy(true);
    setError(null);
    try {
      await setPackEnabled(targetPack.id, !targetPack.enabled);
      await refreshContext();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [targetPack, setPackEnabled, refreshContext]);

  const buildReleaseZip = useCallback(async () => {
    const src = targetPack?.dev_source_path;
    if (!src) {
      setError('No dev-linked source path for zip build.');
      return;
    }
    if (!isTauriRuntime()) {
      setError('Zip build requires the desktop app.');
      return;
    }
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const base64 = await invoke<string>('zip_pack_directory', { absoluteDir: src });
      const result = await installPackFromZip(base64);
      setMessage(`Release zip smoke test installed ${result.pack_id ?? 'pack'}.`);
      await refreshContext();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [targetPack, installPackFromZip, refreshContext]);

  const checklist = useMemo(() => {
    if (!targetPack) return [];
    const items: Array<{ id: string; label: string; ok: boolean; hint?: string }> = [];
    const caps = targetPack.capabilities ?? [];
    const ctx = contexts.find((c) => c.id === targetPack.id);

    items.push({
      id: 'installed',
      label: 'Pack installed',
      ok: targetPack.installed,
    });
    items.push({
      id: 'enabled',
      label: 'Pack enabled',
      ok: targetPack.enabled,
    });
    if (caps.includes(PACK_CAP.CUSTOMER_PACK)) {
      items.push({
        id: 'workspace-guide',
        label: 'Workspace guide available when enabled',
        ok: Boolean(ctx?.workspace_guide?.trim()),
        hint: 'Enable pack and ensure assets.workspace_guide exists',
      });
    }
    if (caps.includes(PACK_CAP.PHOENIX_IMPORT)) {
      items.push({
        id: 'phoenix',
        label: 'Phoenix import (PHX chip in toolbar when enabled)',
        ok: targetPack.enabled,
        hint: 'Sign in via PHX chip after enable; check phoenix overlay keys',
      });
    }
    if (caps.includes(PACK_CAP.SCAN_SUMMARY_VIEWER)) {
      items.push({
        id: 'scan-summary-viewer',
        label: 'Scan summary viewer (Phoenix TIFF + spot overlay)',
        ok: targetPack.enabled,
        hint: 'Open imageMetadata.json or scan-export folder in file explorer',
      });
    }
    if (caps.includes(PACK_CAP.SCAN_ANALYSIS_VIEWER)) {
      items.push({
        id: 'scan-analysis-viewer',
        label: 'Scan analysis viewer (results.json plate maps)',
        ok: targetPack.enabled,
        hint: 'Open reports/results.json after Phoenix import',
      });
    }
    if (caps.includes(PACK_CAP.SECONDARY_ANALYSIS_API)) {
      items.push({
        id: 'secondary-analysis-api',
        label: 'Secondary analysis (12-Plex QC, comparator)',
        ok: targetPack.enabled,
        hint: 'Enable pack; check settings_overlay for tools path, python, panel profile',
      });
    }
    for (const req of targetPack.requires_packs ?? []) {
      const dep = packs.find((p) => p.id === req);
      items.push({
        id: `req-${req}`,
        label: `Requires ${req}`,
        ok: Boolean(dep?.installed && dep?.enabled),
        hint: 'Install and enable in Pack store',
      });
    }
    return items;
  }, [targetPack, contexts, packs]);

  if (customPacks.length === 0) {
    return (
      <p className="text-xs text-slack-textMuted">
        Install or dev-link a custom pack to run the test checklist.
      </p>
    );
  }

  if (!targetPack) {
    return null;
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        {customPacks.length > 1 ? (
          <label className="text-xs text-slack-textMuted flex items-center gap-2">
            Pack
            <select
              value={targetPack?.id ?? ''}
              onChange={(e) => setSelectedPackId(e.target.value)}
              className="rounded border border-slack-border bg-slack-bg px-2 py-1 text-sm text-slack-text"
            >
              {customPacks.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.title}
                </option>
              ))}
            </select>
          </label>
        ) : (
          <span className="text-sm text-slack-text">
            Testing: <strong>{targetPack?.title}</strong> ({targetPack?.id})
          </span>
        )}
        <button
          type="button"
          disabled={busy}
          onClick={() => void toggleEnabled()}
          className="px-3 py-1.5 text-xs rounded border border-slack-border text-slack-text hover:bg-slack-bgHover disabled:opacity-40"
        >
          {targetPack.enabled ? 'Disable' : 'Enable'}
        </button>
        {targetPack.dev_source_path && (
          <button
            type="button"
            disabled={busy}
            onClick={() => void buildReleaseZip()}
            className="px-3 py-1.5 text-xs rounded border border-teal-600/50 text-teal-200 hover:bg-teal-900/40 disabled:opacity-40"
          >
            Build release zip & smoke install
          </button>
        )}
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
      {message && <p className="text-xs text-teal-300">{message}</p>}

      <ul className="space-y-2 text-xs">
        {checklist.map((item) => (
          <li
            key={item.id}
            className={`flex items-start gap-2 ${item.ok ? 'text-emerald-300' : 'text-slack-textMuted'}`}
          >
            <span>{item.ok ? '✓' : '○'}</span>
            <span>
              {item.label}
              {item.hint && !item.ok && (
                <span className="block text-[11px] text-slack-textMuted/80">{item.hint}</span>
              )}
            </span>
          </li>
        ))}
      </ul>

      {contexts.map((ctx) =>
        ctx.workspace_guide ? (
          <div key={ctx.id} className="border border-slack-border rounded-lg p-3 max-h-56 overflow-y-auto">
            <p className="text-xs font-medium text-slack-text mb-2">
              Customer context — {ctx.title}
            </p>
            <RichMarkdownView content={ctx.workspace_guide} />
          </div>
        ) : null,
      )}
    </div>
  );
}
