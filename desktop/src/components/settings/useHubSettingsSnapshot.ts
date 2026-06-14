import { useCallback, useEffect, useState } from 'react';
import { mergeSettingsPut } from './settingsShared';

export function useHubSettingsSnapshot(hubHttp: string, isActive: boolean) {
  const [config, setConfig] = useState<Record<string, unknown> | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const r = await fetch(`${hubHttp}/api/settings`);
      if (!r.ok) {
        throw new Error(await r.text());
      }
      const cfg = (await r.json()) as Record<string, unknown>;
      setConfig(cfg);
      return cfg;
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return null;
    } finally {
      setLoading(false);
    }
  }, [hubHttp]);

  useEffect(() => {
    if (!isActive) return;
    void reload();
  }, [isActive, reload]);

  const save = useCallback(
    async (patch: (cfg: Record<string, unknown>) => Record<string, unknown>) => {
      await mergeSettingsPut(hubHttp, patch);
      await reload();
    },
    [hubHttp, reload]
  );

  return { config, error, loading, reload, save };
}
