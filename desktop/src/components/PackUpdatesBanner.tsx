import { useCallback, useEffect, useState } from 'react';
import type { PackUpdateInfo } from '../api/chatAPI';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { usePacksStore } from '../stores/packsStore';

export interface PackUpdatesBannerProps {
  /** When false, skip polling (e.g. modal closed). */
  active?: boolean;
  onUpgradeStart?: () => void;
  onUpgradeDone?: () => void;
}

export function PackUpdatesBanner({
  active = true,
  onUpgradeStart,
  onUpgradeDone,
}: PackUpdatesBannerProps) {
  const catalog = usePacksStore((s) => s.catalog);
  const upgradePack = usePacksStore((s) => s.upgradePack);
  const fetchPackCatalog = usePacksStore((s) => s.fetchPackCatalog);
  const [updates, setUpdates] = useState<PackUpdateInfo[]>([]);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [dismissed, setDismissed] = useState(false);

  const refresh = useCallback(async () => {
    if (!active) return;
    try {
      const resp = await new ChatAPI(getHubBaseURL()).fetchPackUpdates();
      setUpdates(resp.updates ?? []);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [active]);

  useEffect(() => {
    void refresh();
  }, [refresh, catalog]);

  useEffect(() => {
    if (!active) return;
    const id = window.setInterval(() => {
      if (document.visibilityState === 'visible') void refresh();
    }, 5 * 60 * 1000);
    return () => window.clearInterval(id);
  }, [active, refresh]);

  const pending: PackUpdateInfo[] =
    updates.length > 0
      ? updates
      : catalog
          .filter((e) => e.update_available && e.installed && !e.custom)
          .map((e) => ({
            id: e.id,
            title: e.title,
            installed_version: e.installed_version ?? '',
            latest_version: e.version,
            enabled: e.enabled,
          }));

  if (pending.length === 0 || dismissed) {
    return error ? (
      <p className="text-xs text-red-400 mb-3">Pack update check failed: {error}</p>
    ) : null;
  }

  const upgradeOne = async (packId: string): Promise<boolean> => {
    setBusyId(packId);
    setError(null);
    onUpgradeStart?.();
    try {
      await upgradePack(packId);
      await fetchPackCatalog();
      await refresh();
      return true;
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return false;
    } finally {
      setBusyId(null);
      onUpgradeDone?.();
    }
  };

  const upgradeAll = async () => {
    for (const u of pending) {
      const ok = await upgradeOne(u.id);
      if (!ok) break;
    }
  };

  const label =
    pending.length === 1
      ? `${pending[0].title} v${pending[0].latest_version} available (installed v${pending[0].installed_version || '?'})`
      : `${pending.length} pack updates available`;

  return (
    <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-950/30 px-4 py-3 text-sm">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="font-medium text-amber-100">{label}</p>
          <p className="mt-1 text-xs text-amber-200/80">
            Upgrades re-download the pack bundle from GitHub. Sidecars restart automatically.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {pending.length > 1 && (
            <button
              type="button"
              disabled={busyId !== null}
              onClick={() => void upgradeAll()}
              className="rounded bg-amber-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-500 disabled:opacity-50"
            >
              {busyId ? 'Updating…' : 'Update all'}
            </button>
          )}
          <button
            type="button"
            disabled={busyId !== null}
            onClick={() => void upgradeOne(pending[0].id)}
            className="rounded bg-amber-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-500 disabled:opacity-50"
          >
            {busyId ? 'Updating…' : pending.length === 1 ? 'Update' : `Update ${pending[0].title}`}
          </button>
          <button
            type="button"
            onClick={() => setDismissed(true)}
            className="rounded border border-amber-500/30 px-3 py-1.5 text-xs text-amber-100 hover:bg-amber-900/40"
          >
            Dismiss
          </button>
        </div>
      </div>
      {error && <p className="mt-2 text-xs text-red-300">{error}</p>}
    </div>
  );
}
