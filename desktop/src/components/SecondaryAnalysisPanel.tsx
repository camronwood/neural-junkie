import { useCallback, useEffect, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import {
  SECONDARY_ANALYSIS_HISTORY_FILE,
  useSecondaryAnalysisStore,
  type SecondaryAnalysisHistoryEntry,
} from '../stores/secondaryAnalysisStore';
import { useToastStore } from '../stores/toastStore';
import { usePacksStore } from '../stores/packsStore';
import { PACK_CAP } from '../stores/packCapabilities';
import type { SecondaryAnalysisJob, SecondaryAnalysisWorkflow } from '../utils/secondaryAnalysis';

const api = new ChatAPI(getHubBaseURL());

const WORKFLOWS: { id: SecondaryAnalysisWorkflow; label: string }[] = [
  { id: 'comparator', label: 'Plate comparator' },
  { id: 'endogenous', label: 'Endogenous analysis' },
  { id: 'std_curves', label: 'Standard curves' },
  { id: 'print_order', label: 'Print order QC' },
  { id: '12plex_qc_excel', label: '12-Plex QC (Excel)' },
  { id: 'spc_charts', label: 'SPC charts' },
];

interface SecondaryAnalysisPanelProps {
  onClose: () => void;
}

export function SecondaryAnalysisPanel({ onClose }: SecondaryAnalysisPanelProps) {
  const activeWorkspaceId = useFileExplorerStore((s) => s.activeWorkspaceId);
  const workspacePath = useFileExplorerStore((s) => {
    const ws = s.workspaces.find((w) => w.id === activeWorkspaceId);
    return ws?.path ?? '';
  });
  const hasPython = usePacksStore((s) => s.hasCapability(PACK_CAP.SECONDARY_ANALYSIS_PYTHON));
  const { addToast } = useToastStore();
  const {
    workflow,
    basket,
    jobId,
    jobHistory,
    setWorkflow,
    removeFromBasket,
    clearBasket,
    setJobId,
    updateBasketCondition,
    setJobHistory,
    appendJobHistory,
  } = useSecondaryAnalysisStore();

  const [job, setJob] = useState<SecondaryAnalysisJob | null>(null);
  const [running, setRunning] = useState(false);
  const [appendCumulative, setAppendCumulative] = useState(false);
  const [appendSpc, setAppendSpc] = useState(false);
  const [generateHeatmaps, setGenerateHeatmaps] = useState(true);
  const [generateStdCurves, setGenerateStdCurves] = useState(true);
  const [spikeRecovery, setSpikeRecovery] = useState(false);
  const [serumLayoutJson, setSerumLayoutJson] = useState('');
  const [experimentDetailsCsv, setExperimentDetailsCsv] = useState('');
  const [comparatorDir, setComparatorDir] = useState('');
  const [endogenousDilution, setEndogenousDilution] = useState('4');
  const [includePerSample, setIncludePerSample] = useState(false);
  const [excludePlates, setExcludePlates] = useState('');
  const [spcPreviewImages, setSpcPreviewImages] = useState<string[]>([]);

  const persistHistory = useCallback(
    async (entries: SecondaryAnalysisHistoryEntry[]) => {
      if (!activeWorkspaceId) return;
      try {
        await api.saveFileContent(
          activeWorkspaceId,
          SECONDARY_ANALYSIS_HISTORY_FILE,
          JSON.stringify(entries, null, 2)
        );
      } catch {
        /* ignore write failures */
      }
    },
    [activeWorkspaceId]
  );

  useEffect(() => {
    if (!activeWorkspaceId) return;
    void (async () => {
      try {
        const raw = await api.fetchFileContent(activeWorkspaceId, SECONDARY_ANALYSIS_HISTORY_FILE);
        const parsed = JSON.parse(raw) as SecondaryAnalysisHistoryEntry[];
        if (Array.isArray(parsed)) setJobHistory(parsed.slice(0, 20));
      } catch {
        setJobHistory([]);
      }
    })();
  }, [activeWorkspaceId, setJobHistory]);

  useEffect(() => {
    if (!jobId) return;
    let cancelled = false;
    const poll = async () => {
      try {
        const j = await api.fetchSecondaryAnalysisJob(jobId);
        if (cancelled) return;
        setJob(j);
        if (j.status === 'done' || j.status === 'failed' || j.status === 'cancelled') {
          setRunning(false);
          const entry: SecondaryAnalysisHistoryEntry = {
            id: j.id,
            workflow: j.workflow,
            status: j.status,
            output_dir: j.output_dir,
            created_at: j.created_at ?? new Date().toISOString(),
          };
          appendJobHistory(entry);
          const next = [
            entry,
            ...useSecondaryAnalysisStore.getState().jobHistory.filter((e) => e.id !== entry.id),
          ].slice(0, 20);
          void persistHistory(next);
          if (j.status === 'done') {
            addToast({ type: 'success', title: 'Secondary analysis', message: 'Job completed' });
            if (j.workflow === 'spc_charts' && j.output_dir && activeWorkspaceId) {
              void loadSpcImages(activeWorkspaceId, j.output_dir);
            }
          } else if (j.error) {
            addToast({ type: 'error', title: 'Secondary analysis', message: j.error });
          }
          return;
        }
        setTimeout(poll, 2000);
      } catch (err) {
        if (!cancelled) {
          setRunning(false);
          addToast({
            type: 'error',
            title: 'Job poll',
            message: err instanceof Error ? err.message : 'Poll failed',
          });
        }
      }
    };
    void poll();
    return () => {
      cancelled = true;
    };
  }, [jobId, addToast, appendJobHistory, persistHistory, activeWorkspaceId]);

  const loadSpcImages = async (workspaceId: string, outputDir: string) => {
    try {
      const rel = outputDir.includes('.neural-junkie/')
        ? outputDir.slice(outputDir.indexOf('.neural-junkie/'))
        : `.neural-junkie/analysis-runs/${jobId ?? ''}`;
      const files = await api.fetchFiles(workspaceId, rel);
      const pngs = (files ?? [])
        .filter((f: { is_dir?: boolean; name?: string }) => !f.is_dir && /\.png$/i.test(f.name ?? ''))
        .map((f: { name: string }) => `${rel.replace(/\/$/, '')}/${f.name}`.replace(/\/+/g, '/'));
      setSpcPreviewImages(pngs);
    } catch {
      setSpcPreviewImages([]);
    }
  };

  const buildConfig = useCallback((): Record<string, unknown> => {
    switch (workflow) {
      case 'comparator': {
        const cfg: Record<string, unknown> = {
          plates: basket.map((e) => ({ path: e.path, condition: e.condition ?? 'Condition' })),
          std_curve: generateStdCurves,
          heatmaps: generateHeatmaps,
          spike_recovery: spikeRecovery,
        };
        if (spikeRecovery && serumLayoutJson.trim()) {
          try {
            cfg.serum_layout = JSON.parse(serumLayoutJson);
          } catch {
            /* invalid JSON ignored at run time */
          }
        }
        if (spikeRecovery && experimentDetailsCsv.trim()) {
          cfg.experiment_details_csv = experimentDetailsCsv.trim();
        }
        return cfg;
      }
      case 'endogenous':
        return {
          analysis_dir: comparatorDir,
          dilution: parseInt(endogenousDilution, 10) || 4,
          include_per_sample: includePerSample,
        };
      case 'std_curves':
        return {
          plates: basket.map((e) => ({ path: e.path, label: e.condition ?? 'Condition' })),
          mode: 'normal',
        };
      case 'print_order':
        return {
          printer: 2,
          plates: basket.map((e) => e.path),
          omit_rows: [],
          omit_cols: [],
        };
      case '12plex_qc_excel':
        return {
          summary_dir: basket[0]?.path ?? '',
          append_cumulative_qc: appendCumulative,
          append_cumulative_spc: appendSpc,
        };
      case 'spc_charts':
        return {
          exclude_plates: excludePlates
            .split(/[,;\n]/)
            .map((s) => s.trim())
            .filter(Boolean),
        };
      default:
        return {};
    }
  }, [
    workflow,
    basket,
    comparatorDir,
    generateHeatmaps,
    generateStdCurves,
    spikeRecovery,
    serumLayoutJson,
    experimentDetailsCsv,
    appendCumulative,
    appendSpc,
    endogenousDilution,
    includePerSample,
    excludePlates,
  ]);

  const handleRun = async () => {
    if (!activeWorkspaceId) {
      addToast({ type: 'error', title: 'Secondary analysis', message: 'Select a workspace first' });
      return;
    }
    if (!hasPython) {
      addToast({
        type: 'error',
        title: 'Secondary analysis',
        message: 'Python workflows require tools path in Settings → Life sciences tools',
      });
      return;
    }
    if (workflow === 'endogenous' && !comparatorDir.trim()) {
      addToast({ type: 'error', title: 'Endogenous', message: 'Enter comparator analysis folder path' });
      return;
    }
    if (workflow !== 'endogenous' && workflow !== 'spc_charts' && basket.length === 0) {
      addToast({ type: 'error', title: 'Secondary analysis', message: 'Add at least one folder to the basket' });
      return;
    }
    setRunning(true);
    setJob(null);
    setSpcPreviewImages([]);
    try {
      const started = await api.runSecondaryAnalysis({
        workflow,
        workspace_id: activeWorkspaceId,
        config: buildConfig(),
      });
      setJobId(started.id);
      setJob(started);
    } catch (err) {
      setRunning(false);
      addToast({
        type: 'error',
        title: 'Secondary analysis',
        message: err instanceof Error ? err.message : 'Failed to start job',
      });
    }
  };

  const handleCancel = async () => {
    if (!jobId) return;
    try {
      const j = await api.cancelSecondaryAnalysisJob(jobId);
      setJob(j);
      setRunning(false);
    } catch (err) {
      addToast({
        type: 'error',
        title: 'Cancel',
        message: err instanceof Error ? err.message : 'Cancel failed',
      });
    }
  };

  return (
    <div
      className="flex flex-col h-full border-l border-slack-border bg-slack-bg"
      style={{ width: 430, minWidth: 260 }}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-slack-border">
        <span className="text-sm font-semibold text-slack-text">Secondary analysis</span>
        <button type="button" onClick={onClose} className="text-slack-textMuted hover:text-slack-text">
          ×
        </button>
      </div>

      <div className="flex-1 overflow-auto p-3 space-y-3 text-sm">
        <label className="block">
          <span className="text-xs text-slack-textMuted">Workflow</span>
          <select
            value={workflow}
            onChange={(e) => setWorkflow(e.target.value)}
            className="mt-1 w-full px-2 py-1 border border-slack-border rounded bg-slack-bg text-slack-text"
          >
            {WORKFLOWS.map((w) => (
              <option key={w.id} value={w.id}>
                {w.label}
              </option>
            ))}
          </select>
        </label>

        {workflow === 'endogenous' && (
          <>
            <label className="block">
              <span className="text-xs text-slack-textMuted">Comparator analysis folder</span>
              <input
                type="text"
                value={comparatorDir}
                onChange={(e) => setComparatorDir(e.target.value)}
                placeholder="Comparator Analysis …"
                className="mt-1 w-full px-2 py-1 border border-slack-border rounded bg-slack-bg font-mono text-xs"
              />
            </label>
            <label className="block">
              <span className="text-xs text-slack-textMuted">Sample dilution factor</span>
              <input
                type="number"
                value={endogenousDilution}
                onChange={(e) => setEndogenousDilution(e.target.value)}
                className="mt-1 w-20 px-2 py-1 border border-slack-border rounded bg-slack-bg text-xs"
              />
            </label>
            <label className="flex items-center gap-2 text-xs cursor-pointer">
              <input
                type="checkbox"
                checked={includePerSample}
                onChange={(e) => setIncludePerSample(e.target.checked)}
              />
              Include per-sample matrix in CSV
            </label>
          </>
        )}

        {workflow !== 'endogenous' && workflow !== 'spc_charts' && (
          <div>
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs text-slack-textMuted">Folder basket</span>
              <button type="button" className="text-xs text-slack-accent" onClick={() => clearBasket()}>
                Clear
              </button>
            </div>
            {basket.length === 0 ? (
              <p className="text-xs text-slack-textMuted">
                Right-click summary folders → Add to analysis basket
              </p>
            ) : (
              <ul className="space-y-2">
                {basket.map((e) => (
                  <li
                    key={e.path}
                    className="border border-slack-border rounded px-2 py-1 text-xs"
                  >
                    <div className="flex items-center gap-2">
                      <span className="flex-1 truncate font-mono" title={e.path}>
                        {e.path}
                      </span>
                      <button type="button" onClick={() => removeFromBasket(e.path)} className="text-red-400">
                        ✕
                      </button>
                    </div>
                    {workflow === 'comparator' || workflow === 'std_curves' ? (
                      <input
                        type="text"
                        value={e.condition ?? ''}
                        onChange={(ev) => updateBasketCondition(e.path, ev.target.value)}
                        placeholder="Condition label"
                        className="mt-1 w-full px-1 py-0.5 border border-slack-border rounded bg-slack-bg font-mono"
                      />
                    ) : null}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}

        {workflow === 'comparator' && (
          <div className="space-y-1 text-xs">
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={generateHeatmaps} onChange={(e) => setGenerateHeatmaps(e.target.checked)} />
              Generate heatmaps
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={generateStdCurves} onChange={(e) => setGenerateStdCurves(e.target.checked)} />
              Per-plate standard curves
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={spikeRecovery} onChange={(e) => setSpikeRecovery(e.target.checked)} />
              Spike recovery (requires layout below)
            </label>
            {spikeRecovery && (
              <>
                <label className="block mt-2">
                  <span className="text-slack-textMuted">Serum layout JSON (optional)</span>
                  <textarea
                    value={serumLayoutJson}
                    onChange={(e) => setSerumLayoutJson(e.target.value)}
                    rows={3}
                    placeholder='{"PlateName-summary": {"grid": [["S1 4 4", ...], ...]}}'
                    className="mt-1 w-full px-2 py-1 border border-slack-border rounded bg-slack-bg font-mono text-[10px]"
                  />
                </label>
                <label className="block">
                  <span className="text-slack-textMuted">Or Experiment_Details.csv path</span>
                  <input
                    type="text"
                    value={experimentDetailsCsv}
                    onChange={(e) => setExperimentDetailsCsv(e.target.value)}
                    placeholder="Comparator Analysis …/Experiment_Details.csv"
                    className="mt-1 w-full px-2 py-1 border border-slack-border rounded bg-slack-bg font-mono text-[10px]"
                  />
                </label>
              </>
            )}
          </div>
        )}

        {workflow === '12plex_qc_excel' && (
          <div className="space-y-1 text-xs">
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={appendCumulative} onChange={(e) => setAppendCumulative(e.target.checked)} />
              Append to cumulative QC spreadsheet
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={appendSpc} onChange={(e) => setAppendSpc(e.target.checked)} />
              Append to cumulative SPC spreadsheet
            </label>
          </div>
        )}

        {workflow === 'spc_charts' && (
          <label className="block">
            <span className="text-xs text-slack-textMuted">Exclude plates (comma-separated names)</span>
            <input
              type="text"
              value={excludePlates}
              onChange={(e) => setExcludePlates(e.target.value)}
              className="mt-1 w-full px-2 py-1 border border-slack-border rounded bg-slack-bg font-mono text-xs"
            />
          </label>
        )}

        <div className="flex gap-2">
          <button
            type="button"
            disabled={running}
            onClick={() => void handleRun()}
            className="px-3 py-1.5 bg-purple-600 text-white rounded text-xs disabled:opacity-50"
          >
            {running ? 'Running…' : 'Run'}
          </button>
          {running && jobId && (
            <button
              type="button"
              onClick={() => void handleCancel()}
              className="px-3 py-1.5 border border-slack-border rounded text-xs"
            >
              Cancel
            </button>
          )}
        </div>

        {job && (
          <div className="border border-slack-border rounded p-2 text-xs">
            <div className="font-mono">
              Job {job.id.slice(0, 8)} — <span className="text-slack-accent">{job.status}</span>
            </div>
            {job.output_dir && (
              <div className="mt-1 truncate" title={job.output_dir}>
                Output: {job.output_dir}
              </div>
            )}
            {job.error && <div className="mt-1 text-red-400">{job.error}</div>}
            {job.log_tail && job.log_tail.length > 0 && (
              <pre className="mt-2 max-h-32 overflow-y-auto text-[10px] font-mono bg-slack-bgHover p-1 rounded">
                {job.log_tail.slice(-12).join('\n')}
              </pre>
            )}
          </div>
        )}

        {spcPreviewImages.length > 0 && activeWorkspaceId && (
          <div>
            <div className="text-xs font-semibold text-slack-textMuted mb-1">SPC charts</div>
            <div className="space-y-2 max-h-48 overflow-auto">
              {spcPreviewImages.map((rel) => (
                <SpcChartThumb key={rel} workspaceId={activeWorkspaceId} relativePath={rel} />
              ))}
            </div>
          </div>
        )}

        {jobHistory.length > 0 && (
          <div>
            <div className="text-xs font-semibold text-slack-textMuted mb-1">Recent jobs</div>
            <ul className="space-y-1 max-h-32 overflow-auto">
              {jobHistory.map((h) => (
                <li key={h.id} className="text-[10px] font-mono border border-slack-border/50 rounded px-2 py-1">
                  {h.workflow} — {h.status} — {h.id.slice(0, 8)}
                  {h.output_dir && (
                    <div className="truncate text-slack-textMuted" title={h.output_dir}>
                      {h.output_dir}
                    </div>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        {workspacePath && (
          <p className="text-[10px] text-slack-textMuted">
            Cumulative QC: {workspacePath}/.neural-junkie/cumulative-qc (Settings override)
          </p>
        )}
      </div>
    </div>
  );
}

function SpcChartThumb({ workspaceId, relativePath }: { workspaceId: string; relativePath: string }) {
  const [src, setSrc] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const url = await api.fetchWorkspaceImageDataUrl(workspaceId, relativePath);
        if (!cancelled) setSrc(url);
      } catch {
        if (!cancelled) setSrc(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [workspaceId, relativePath]);
  if (!src) return <div className="text-[10px] text-slack-textMuted">{relativePath}</div>;
  return (
    <img
      src={src}
      alt={relativePath}
      className="max-w-full rounded border border-slack-border"
    />
  );
}
