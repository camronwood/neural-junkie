import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  arenaBenchmarkCancel,
  arenaBenchmarkGet,
  arenaBenchmarkHistory,
  arenaBenchmarkStart,
  arenaBenchmarkSuites,
  fetchInstalledOllamaModels,
  fetchProviders,
  type ArenaBenchmarkJob,
  type ArenaBenchmarkSuite,
} from './arenaSidecarApi';
import {
  benchmarkProgressLabel,
  formatLogicAccuracy,
  formatPassRate,
  isBenchmarkJobActive,
  rankingFromJob,
  toggleModelSelection,
} from './arenaBenchmarkHelpers';
import { useToastStore } from '../../stores/toastStore';

const POLL_MS = 1200;

type ArenaBenchmarkPanelProps = {
  providerId: string;
  onProviderChange: (id: string) => void;
  onLeaderboardRefresh?: () => void;
};

export function ArenaBenchmarkPanel({
  providerId,
  onProviderChange,
  onLeaderboardRefresh,
}: ArenaBenchmarkPanelProps) {
  const { addToast } = useToastStore();
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [suites, setSuites] = useState<ArenaBenchmarkSuite[]>([]);
  const [suiteId, setSuiteId] = useState('quick');
  const [providers, setProviders] = useState<Array<{ id: string; name: string }>>([]);
  const [job, setJob] = useState<ArenaBenchmarkJob | null>(null);
  const [history, setHistory] = useState<ArenaBenchmarkJob[]>([]);
  const [busy, setBusy] = useState(false);
  const [loadError, setLoadError] = useState('');
  const pollRef = useRef<number | null>(null);

  const stopPoll = useCallback(() => {
    if (pollRef.current != null) {
      window.clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const refreshModels = useCallback(async () => {
    const tags = await fetchInstalledOllamaModels();
    setOllamaModels(tags);
    setSelected((prev) => prev.filter((m) => tags.includes(m)));
    return tags;
  }, []);

  const refreshHistory = useCallback(async () => {
    try {
      const data = await arenaBenchmarkHistory(8);
      setHistory(Array.isArray(data.runs) ? data.runs : []);
    } catch {
      /* optional */
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const [tags, suiteResp, provs] = await Promise.all([
          refreshModels(),
          arenaBenchmarkSuites(),
          fetchProviders(),
        ]);
        if (cancelled) return;
        setLoadError('');
        setSuites(suiteResp.suites ?? []);
        setSuiteId(suiteResp.default_suite || suiteResp.suites?.[0]?.id || 'quick');
        setProviders(provs.map((p) => ({ id: p.id, name: p.name || p.id })));
        if (!providerId && provs[0]?.id) onProviderChange(provs[0].id);
        if (tags.length > 0 && selected.length === 0) {
          setSelected([tags[0]]);
        }
        await refreshHistory();
      } catch (err) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : String(err));
        }
      }
    })();
    return () => {
      cancelled = true;
    };
    // intentionally once on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => () => stopPoll(), [stopPoll]);

  const startPolling = useCallback(
    (id: string) => {
      stopPoll();
      pollRef.current = window.setInterval(() => {
        void arenaBenchmarkGet(id)
          .then((next) => {
            setJob(next);
            if (!isBenchmarkJobActive(next.status)) {
              stopPoll();
              setBusy(false);
              void refreshHistory();
              onLeaderboardRefresh?.();
            }
          })
          .catch(() => {
            /* keep polling until cancel/complete surfaces */
          });
      }, POLL_MS);
    },
    [onLeaderboardRefresh, refreshHistory, stopPoll],
  );

  const runBenchmark = useCallback(async () => {
    if (selected.length === 0) {
      addToast({
        type: 'error',
        title: 'Benchmark',
        message: 'Select at least one installed Ollama model.',
      });
      return;
    }
    setBusy(true);
    setLoadError('');
    try {
      const started = await arenaBenchmarkStart({
        models: selected,
        suite: suiteId,
        provider_id: providerId,
      });
      setJob(started);
      startPolling(started.id);
    } catch (err) {
      setBusy(false);
      const message = err instanceof Error ? err.message : String(err);
      setLoadError(message);
      addToast({ type: 'error', title: 'Benchmark', message });
    }
  }, [addToast, providerId, selected, startPolling, suiteId]);

  const cancelBenchmark = useCallback(async () => {
    if (!job?.id) return;
    try {
      const next = await arenaBenchmarkCancel(job.id);
      setJob(next);
      if (!isBenchmarkJobActive(next.status)) {
        stopPoll();
        setBusy(false);
        void refreshHistory();
      }
    } catch (err) {
      addToast({
        type: 'error',
        title: 'Benchmark',
        message: err instanceof Error ? err.message : String(err),
      });
    }
  }, [addToast, job?.id, refreshHistory, stopPoll]);

  const ranking = useMemo(() => rankingFromJob(job), [job]);
  const active = isBenchmarkJobActive(job?.status);
  const suiteMeta = suites.find((s) => s.id === suiteId);

  return (
    <div className="arena-benchmark" data-testid="arena-benchmark-panel">
      <p className="arena-retro-hint mb-3">
        Rank installed Ollama models on verifiable Arena challenges. Default suite runs logic puzzles plus a Connect
        Four completion check.
      </p>

      {loadError && (
        <div className="arena-retro-hint mb-3 border border-amber-500/50 bg-amber-500/10 px-3 py-2 text-amber-100">
          {loadError}
        </div>
      )}

      <div className="arena-benchmark-controls">
        <label>
          <span className="arena-retro-label">Suite</span>
          <select
            className="arena-retro-select"
            value={suiteId}
            disabled={busy || active}
            onChange={(e) => setSuiteId(e.target.value)}
          >
            {suites.length === 0 && <option value="quick">Quick</option>}
            {suites.map((s) => (
              <option key={s.id} value={s.id}>
                {s.title || s.id}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span className="arena-retro-label">AI provider</span>
          <select
            className="arena-retro-select min-w-[10rem]"
            value={providerId}
            disabled={busy || active}
            onChange={(e) => onProviderChange(e.target.value)}
          >
            {providers.length === 0 && <option value="">No providers</option>}
            {providers.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name || p.id}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          className="arena-retro-btn secondary"
          disabled={busy || active}
          onClick={() => void refreshModels()}
        >
          Refresh models
        </button>
      </div>

      {suiteMeta?.description && <p className="arena-retro-hint mb-2">{suiteMeta.description}</p>}

      <div className="arena-benchmark-models">
        <span className="arena-retro-label">Installed models</span>
        {ollamaModels.length === 0 ? (
          <p className="arena-retro-hint">
            No Ollama tags found. Pull models in the model library (⇧⌘M), then refresh.
          </p>
        ) : (
          <ul className="arena-benchmark-model-list">
            {ollamaModels.map((tag) => {
              const checked = selected.includes(tag);
              return (
                <li key={tag}>
                  <label className="arena-benchmark-model-row">
                    <input
                      type="checkbox"
                      checked={checked}
                      disabled={busy || active}
                      onChange={() => setSelected((prev) => toggleModelSelection(prev, tag))}
                    />
                    <span title={tag}>{tag}</span>
                  </label>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <div className="arena-retro-toolbar">
        <button
          type="button"
          className="arena-retro-btn"
          disabled={busy || active || selected.length === 0}
          onClick={() => void runBenchmark()}
        >
          {active ? 'Running…' : 'Run benchmark'}
        </button>
        {active && (
          <button type="button" className="arena-retro-btn secondary" onClick={() => void cancelBenchmark()}>
            Cancel
          </button>
        )}
      </div>

      {job && (
        <div className="arena-benchmark-status" role="status" aria-live="polite">
          <div className="arena-retro-label">Status · {(job.status || '').toUpperCase()}</div>
          <div>{benchmarkProgressLabel(job)}</div>
          {job.suite_title && (
            <div className="arena-retro-hint">
              Suite · {job.suite_title} ({(job.challenges || []).join(', ')})
            </div>
          )}
        </div>
      )}

      <div className="arena-benchmark-results">
        <h3>Scoreboard</h3>
        {ranking.length === 0 ? (
          <p className="arena-retro-hint text-center">No results yet</p>
        ) : (
          <table className="arena-benchmark-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Model</th>
                <th>Pass</th>
                <th>Logic</th>
                <th>C4</th>
              </tr>
            </thead>
            <tbody>
              {ranking.map((row) => (
                <tr key={row.model}>
                  <td>{row.rank}</td>
                  <td title={row.model}>{row.model}</td>
                  <td>{formatPassRate(row.pass_rate)}</td>
                  <td>{formatLogicAccuracy(row.logic_correct, row.logic_total)}</td>
                  <td>{row.connect4_completed ? '✓' : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {history.length > 0 && (
        <div className="arena-benchmark-history">
          <h3>Recent runs</h3>
          <ul>
            {history.map((run) => (
              <li key={run.id}>
                <button
                  type="button"
                  className="arena-benchmark-history-btn"
                  onClick={() => setJob(run)}
                  disabled={active}
                >
                  {(run.suite_title || run.suite) ?? 'suite'} · {(run.status || '').toUpperCase()} ·{' '}
                  {(run.models || []).length} model{(run.models || []).length === 1 ? '' : 's'}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
