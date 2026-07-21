import { useCallback, useEffect, useMemo, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useToastStore } from '../stores/toastStore';
import { usePacksStore } from '../stores/packsStore';
import { PACK_CAP } from '../stores/packCapabilities';
import {
  analyteColor,
  concentrationAt,
  PLATE_COLS,
  PLATE_ROWS,
  plotRelativePath,
  plateCellValue,
  processReportRelativePath,
  runLabelFromAnalysisDir,
  type PlateGridMode,
  type ScanAnalysisData,
  validationAt,
} from '../utils/scanAnalysis';
import {
  fetchScanAnalysisProcessReport,
  resolveScanAnalysisAssetSrc,
} from '../utils/scanAnalysisPlot';
import { parseScanSummaryMetadata } from '../utils/scanSummary';
import { normalizeScanLinkInput, validateScanLink, scanMetadataRelativePath } from '../utils/scanAnalysisLink';
import { workspaceAbsolutePath, workspaceRelativePath } from '../utils/editorFileKind';
import { formatConcDisplay } from '../utils/scanAnalysisHelpers';
import type { PanelQCReport } from '../utils/secondaryAnalysis';
import {
  formatBiologyExpertQcPrompt,
  panelQcExportRelativePaths,
  panelQcJsonContent,
  panelQcToCsv,
  qcFailureWellsForAnalyte,
  qcReportRelativePath,
} from '../utils/panelQcUtils';
import { isTauriRuntime } from '../utils/promptAttachments';
import { useComposerPrefillStore } from '../stores/composerPrefillStore';

interface ScanAnalysisViewerProps {
  workspaceId: string;
  analysisDir: string;
  data: ScanAnalysisData;
  initialWell?: string;
  initialAnalyte?: string;
  linkedScanDir?: string;
  tabId: string;
}

const api = new ChatAPI(getHubBaseURL());

function formatConc(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) return '—';
  if (value >= 100) return value.toFixed(1);
  if (value >= 1) return value.toFixed(2);
  return value.toFixed(4);
}

function concColor(value: number | null, min: number, max: number): string {
  if (value == null || Number.isNaN(value)) return '#374151';
  if (max <= min) return '#6366f1';
  const t = Math.min(1, Math.max(0, (Math.log10(value + 1e-10) - Math.log10(min + 1e-10)) / (Math.log10(max + 1e-10) - Math.log10(min + 1e-10))));
  const r = Math.round(30 + t * 180);
  const g = Math.round(60 + (1 - Math.abs(t - 0.5) * 2) * 80);
  const b = Math.round(200 - t * 160);
  return `rgb(${r},${g},${b})`;
}

function wellTypeColor(wellType: string | null): string {
  switch (wellType) {
    case 'standard':
      return '#3b82f6';
    case 'unknown':
      return '#8b5cf6';
    case 'blank':
      return '#6b7280';
    default:
      return '#374151';
  }
}

export function ScanAnalysisViewer({
  workspaceId,
  analysisDir,
  data,
  initialWell = 'A1',
  initialAnalyte,
  linkedScanDir: linkedScanDirProp,
  tabId,
}: ScanAnalysisViewerProps) {
  const workspacePath = useFileExplorerStore((s) => {
    const ws = s.workspaces.find((w) => w.id === workspaceId);
    return ws?.path ?? '';
  });
  const { addToast } = useToastStore();
  const {
    linkScanToAnalysisTab,
    openScanSummary,
    activateScanWell,
    findLinkedScanTab,
    setPanelQCReport,
  } = useEditorStore();
  const panelQCReport = useEditorStore((s) => s.tabs.find((t) => t.id === tabId)?.panelQCReport);
  const refreshTreeForPath = useFileExplorerStore((s) => s.refreshTreeForPath);
  const hasSecondaryQC = usePacksStore((s) => s.hasCapability(PACK_CAP.SECONDARY_ANALYSIS_VIEWER));
  const requestComposerPrefill = useComposerPrefillStore((s) => s.requestPrefill);

  const [selectedWell, setSelectedWell] = useState(initialWell);
  const [selectedAnalyte, setSelectedAnalyte] = useState(
    initialAnalyte && data.analytes.includes(initialAnalyte) ? initialAnalyte : data.analytes[0] ?? ''
  );
  const [gridMode, setGridMode] = useState<PlateGridMode>('concentration');
  const [linkedScanDir, setLinkedScanDir] = useState(linkedScanDirProp ?? '');
  const [heatMapSrc, setHeatMapSrc] = useState<string | null>(null);
  const [calibrationSrc, setCalibrationSrc] = useState<string | null>(null);
  const [processReport, setProcessReport] = useState<string | null>(null);
  const [showProcessReport, setShowProcessReport] = useState(false);
  const [showQCReport, setShowQCReport] = useState(true);
  const [qcRunning, setQcRunning] = useState(false);
  const [showFitParams, setShowFitParams] = useState(false);
  const [linkInput, setLinkInput] = useState('');
  const [linkingScan, setLinkingScan] = useState(false);
  const [browsingScan, setBrowsingScan] = useState(false);

  useEffect(() => {
    setSelectedWell(initialWell);
  }, [initialWell, analysisDir]);

  useEffect(() => {
    if (initialAnalyte && data.analytes.includes(initialAnalyte)) {
      setSelectedAnalyte(initialAnalyte);
    }
  }, [initialAnalyte, analysisDir, data.analytes]);

  useEffect(() => {
    if (linkedScanDirProp) setLinkedScanDir(linkedScanDirProp);
  }, [linkedScanDirProp]);

  const runLabel = runLabelFromAnalysisDir(analysisDir);
  const wellRow = validationAt(data, selectedWell, selectedAnalyte);
  const dilution = data.experiment.dilutionFactor;

  const concRange = useMemo(() => {
    let min = Infinity;
    let max = -Infinity;
    for (const row of data.validation) {
      if (row.analyte !== selectedAnalyte) continue;
      const c = row.calculatedConcentration;
      if (c != null && !Number.isNaN(c) && c > 0) {
        if (c < min) min = c;
        if (c > max) max = c;
      }
    }
    if (!Number.isFinite(min)) return { min: 0, max: 1 };
    return { min, max };
  }, [data, selectedAnalyte]);

  const loadPlots = useCallback(async () => {
    if (!workspacePath || !selectedAnalyte) return;
    try {
      const heatPath = plotRelativePath(analysisDir, selectedAnalyte, 'heat_map');
      const calPath = plotRelativePath(analysisDir, selectedAnalyte, 'calibration_curve');
      const [heat, cal] = await Promise.all([
        resolveScanAnalysisAssetSrc({ workspaceId, workspacePath, relativePath: heatPath }).catch(() => ''),
        resolveScanAnalysisAssetSrc({ workspaceId, workspacePath, relativePath: calPath }).catch(() => ''),
      ]);
      setHeatMapSrc(heat || null);
      setCalibrationSrc(cal || null);
    } catch {
      setHeatMapSrc(null);
      setCalibrationSrc(null);
    }
  }, [workspaceId, workspacePath, analysisDir, selectedAnalyte]);

  const loadProcessReport = useCallback(async () => {
    if (!workspacePath) return;
    try {
      const text = await fetchScanAnalysisProcessReport({
        workspaceId,
        relativePath: processReportRelativePath(analysisDir),
      });
      setProcessReport(text);
    } catch {
      setProcessReport(null);
    }
  }, [workspaceId, workspacePath, analysisDir]);

  useEffect(() => {
    void loadPlots();
  }, [loadPlots]);

  useEffect(() => {
    void loadProcessReport();
  }, [loadProcessReport]);

  useEffect(() => {
    if (panelQCReport) return;
    let cancelled = false;
    void (async () => {
      try {
        const raw = await api.fetchFileContent(workspaceId, qcReportRelativePath(analysisDir));
        if (cancelled || !raw) return;
        const parsed = JSON.parse(raw) as PanelQCReport;
        if (parsed?.analytes) setPanelQCReport(tabId, parsed);
      } catch {
        /* no saved report */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [workspaceId, analysisDir, tabId, panelQCReport, setPanelQCReport]);

  const qcFailureWells = useMemo(() => {
    if (!panelQCReport || gridMode !== 'qcFailures') return new Set<string>();
    return qcFailureWellsForAnalyte(panelQCReport, selectedAnalyte);
  }, [panelQCReport, gridMode, selectedAnalyte]);

  const handleOpenScanImage = async () => {
    if (!linkedScanDir) {
      addToast({ type: 'info', title: 'Scan link', message: 'Link a scan folder first.' });
      return;
    }
    const existingScanTab = findLinkedScanTab(workspaceId, analysisDir);
    if (existingScanTab) {
      activateScanWell(existingScanTab.id, selectedWell);
      return;
    }
    const metaPath = scanMetadataRelativePath(linkedScanDir);
    try {
      const raw = await api.fetchFileContent(workspaceId, metaPath);
      const scanData = parseScanSummaryMetadata(raw);
      openScanSummary(workspaceId, normalizeScanLinkInput(linkedScanDir), scanData, selectedWell, {
        linkedAnalysisDir: analysisDir,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to open scan summary';
      addToast({
        type: 'error',
        title: 'Scan image',
        message: message.includes('Not Found') || message.includes('no such file')
          ? `Scan metadata not found. Link the folder containing imageMetadata.json (e.g. scan-export), not the file path. Tried: ${metaPath}`
          : message,
      });
    }
  };

  const applyScanLink = async (dir: string) => {
    const trimmed = dir.trim();
    if (!trimmed) return;
    setLinkingScan(true);
    try {
      const result = await validateScanLink(workspaceId, trimmed, (ws, path) =>
        api.fetchFileContent(ws, path)
      );
      if (!result.ok) {
        addToast({ type: 'error', title: 'Scan link', message: result.reason });
        return;
      }
      setLinkedScanDir(result.linkedScanDir);
      linkScanToAnalysisTab(tabId, result.linkedScanDir);
      addToast({
        type: 'success',
        title: 'Scan linked',
        message: result.linkedScanDir || '(workspace root)',
      });
      setLinkInput('');
    } finally {
      setLinkingScan(false);
    }
  };

  const handleLinkScan = () => void applyScanLink(linkInput);

  const handleBrowseScanLink = async () => {
    if (!isTauriRuntime()) {
      addToast({
        type: 'error',
        title: 'Browse',
        message: 'Folder picker requires the desktop app.',
      });
      return;
    }
    if (!workspacePath) {
      addToast({
        type: 'error',
        title: 'Browse',
        message: 'Workspace path is not available.',
      });
      return;
    }
    setBrowsingScan(true);
    try {
      const { open } = await import('@tauri-apps/plugin-dialog');
      const scanHint = analysisDir ? `${analysisDir.replace(/[/\\]+$/, '')}/scan-export` : 'scan-export';
      const defaultPath = workspaceAbsolutePath(workspacePath, scanHint);
      const selected = await open({
        directory: true,
        multiple: false,
        title: 'Select scan export folder (contains imageMetadata.json)',
        defaultPath,
      });
      if (!selected || typeof selected !== 'string') return;

      const relative = workspaceRelativePath(workspacePath, selected);
      if (relative == null) {
        addToast({
          type: 'error',
          title: 'Scan link',
          message: 'Selected folder must be inside the workspace root.',
        });
        return;
      }
      setLinkInput(relative);
      await applyScanLink(relative);
    } catch (err) {
      addToast({
        type: 'error',
        title: 'Browse',
        message: err instanceof Error ? err.message : 'Failed to open folder picker',
      });
    } finally {
      setBrowsingScan(false);
    }
  };

  const handleRun12PlexQC = async () => {
    setQcRunning(true);
    try {
      const report = await api.run12PlexQC({
        workspace_id: workspaceId,
        analysis_dir: analysisDir,
        write_report: true,
      });
      setPanelQCReport(tabId, report);
      setGridMode('qcFailures');
      await refreshTreeForPath(workspaceId, qcReportRelativePath(analysisDir));
      addToast({
        type: report.overall_pass ? 'success' : 'info',
        title: '12-Plex QC',
        message: report.overall_pass ? 'Plate passed SOP QC' : 'Plate failed one or more QC checks',
      });
    } catch (err) {
      addToast({
        type: 'error',
        title: '12-Plex QC',
        message: err instanceof Error ? err.message : 'QC failed',
      });
    } finally {
      setQcRunning(false);
    }
  };

  const handleExportQcJson = async () => {
    if (!panelQCReport) return;
    const { json } = panelQcExportRelativePaths(analysisDir, runLabel);
    try {
      await api.saveFileContent(workspaceId, json, panelQcJsonContent(panelQCReport));
      await refreshTreeForPath(workspaceId, json);
      addToast({
        type: 'success',
        title: 'Export JSON',
        message: `Saved ${json}`,
      });
    } catch (err) {
      addToast({
        type: 'error',
        title: 'Export JSON',
        message: err instanceof Error ? err.message : 'Export failed',
      });
    }
  };

  const handleExportQcCsv = async () => {
    if (!panelQCReport) return;
    const { csv } = panelQcExportRelativePaths(analysisDir, runLabel);
    try {
      await api.saveFileContent(workspaceId, csv, panelQcToCsv(panelQCReport));
      await refreshTreeForPath(workspaceId, csv);
      addToast({
        type: 'success',
        title: 'Export CSV',
        message: `Saved ${csv}`,
      });
    } catch (err) {
      addToast({
        type: 'error',
        title: 'Export CSV',
        message: err instanceof Error ? err.message : 'Export failed',
      });
    }
  };

  const handleAskBiologyExpert = () => {
    if (!panelQCReport) {
      addToast({ type: 'info', title: '12-Plex QC', message: 'Run QC first.' });
      return;
    }
    requestComposerPrefill(formatBiologyExpertQcPrompt(panelQCReport, analysisDir));
    addToast({
      type: 'info',
      title: 'BiologyExpert',
      message: 'QC summary added to chat composer — send when ready.',
    });
  };

  const standardRows = data.standardReport[selectedAnalyte] ?? [];
  const unknownRows = data.unknownReport[selectedAnalyte] ?? [];
  const loq = data.limitsOfQuant[selectedAnalyte];
  const fit = data.fitParameters[selectedAnalyte];
  const spots = data.spotsByWellAnalyte.get(`${selectedWell}|${selectedAnalyte}`) ?? [];

  return (
    <div className="flex flex-col h-full min-h-0 bg-slack-bg text-slack-text">
      <div className="flex-shrink-0 px-4 py-2 border-b border-slack-border space-y-2">
        <div className="flex flex-wrap items-center gap-3">
          <div className="min-w-0">
            <div className="text-sm font-semibold truncate">{runLabel}</div>
            <div className="text-xs text-slack-textMuted truncate">
              {data.experiment.productName}
              {data.experiment.plateBarcode ? ` · ${data.experiment.plateBarcode}` : ''}
            </div>
          </div>
          <div className="ml-auto flex items-center gap-2 text-xs">
            {hasSecondaryQC && (
              <>
                <button
                  type="button"
                  className="px-2 py-1 rounded border border-purple-600/50 text-purple-300 hover:bg-purple-600/20 disabled:opacity-50"
                  disabled={qcRunning}
                  onClick={() => void handleRun12PlexQC()}
                >
                  {qcRunning ? 'QC…' : 'Run 12-Plex QC'}
                </button>
                {panelQCReport && (
                  <>
                    <button
                      type="button"
                      className="px-2 py-1 rounded border border-slack-border hover:border-slack-accent"
                      onClick={() => void handleExportQcJson()}
                    >
                      Export JSON
                    </button>
                    <button
                      type="button"
                      className="px-2 py-1 rounded border border-slack-border hover:border-slack-accent"
                      onClick={() => void handleExportQcCsv()}
                    >
                      Export CSV
                    </button>
                    <button
                      type="button"
                      className="px-2 py-1 rounded border border-teal-600/50 text-teal-300 hover:bg-teal-600/20"
                      onClick={handleAskBiologyExpert}
                    >
                      Ask BiologyExpert
                    </button>
                  </>
                )}
              </>
            )}
            <button
              type="button"
              className="px-2 py-1 rounded border border-slack-border hover:border-slack-accent disabled:opacity-40"
              disabled={!linkedScanDir}
              onClick={() => void handleOpenScanImage()}
            >
              View scan image
            </button>
          </div>
        </div>
        {dilution != null && dilution !== 1 && (
          <div className="text-xs text-amber-400/90 bg-amber-950/30 border border-amber-800/40 rounded px-2 py-1">
            Unknown concentrations do not include dilution factor ({dilution}×). Multiply for final pg/ml.
          </div>
        )}
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <span className="text-slack-textMuted">Scan link:</span>
          {linkedScanDir ? (
            <span className="font-mono truncate max-w-xs">{linkedScanDir || '(workspace root)'}</span>
          ) : (
            <span className="text-slack-textMuted">Not linked</span>
          )}
          <input
            type="text"
            placeholder="Scan folder (e.g. scan-export)"
            value={linkInput}
            onChange={(e) => setLinkInput(e.target.value)}
            className="px-2 py-0.5 rounded border border-slack-border bg-slack-bg text-xs min-w-[8rem]"
          />
          <button
            type="button"
            className="px-2 py-0.5 rounded border border-slack-border hover:border-slack-accent disabled:opacity-50"
            title={isTauriRuntime() ? 'Browse for scan export folder' : 'Folder picker requires the desktop app'}
            onClick={() => void handleBrowseScanLink()}
            disabled={browsingScan || linkingScan || !isTauriRuntime()}
          >
            {browsingScan ? '…' : 'Browse…'}
          </button>
          <button
            type="button"
            className="px-2 py-0.5 rounded border border-slack-border hover:border-slack-accent disabled:opacity-50"
            onClick={() => void handleLinkScan()}
            disabled={linkingScan || browsingScan || !linkInput.trim()}
          >
            {linkingScan ? 'Linking…' : 'Link'}
          </button>
        </div>
      </div>

      <div className="flex flex-1 min-h-0">
        <div className="w-44 flex-shrink-0 border-r border-slack-border overflow-auto p-2 space-y-2">
          <div className="text-[10px] font-semibold text-slack-textMuted">Plate mode</div>
          {(['concentration', 'intensity', 'loq', 'wellType', ...(panelQCReport ? (['qcFailures'] as const) : [])] as PlateGridMode[]).map((mode) => (
            <label key={mode} className="flex items-center gap-1 text-[10px] cursor-pointer">
              <input
                type="radio"
                name="grid-mode"
                checked={gridMode === mode}
                onChange={() => setGridMode(mode)}
              />
              {mode === 'qcFailures' ? 'QC failures' : mode}
            </label>
          ))}
          <div
            className="grid gap-0.5 text-[10px] mt-2"
            style={{ gridTemplateColumns: 'auto repeat(12, minmax(0, 1fr))' }}
          >
            <div />
            {PLATE_COLS.map((c) => (
              <div key={c} className="text-center text-slack-textMuted font-medium py-0.5">
                {c}
              </div>
            ))}
            {PLATE_ROWS.map((row) => (
              <span key={row} className="contents">
                <div className="text-slack-textMuted font-medium flex items-center justify-center pr-1">
                  {row}
                </div>
                {PLATE_COLS.map((col) => {
                  const wellId = `${row}${col}`;
                  const cellVal = plateCellValue(data, wellId, selectedAnalyte, gridMode);
                  let bg = '#1f2937';
                  if (gridMode === 'qcFailures') {
                    bg = qcFailureWells.has(wellId) ? '#dc2626' : '#1f2937';
                  } else if (gridMode === 'concentration') {
                    bg = concColor(cellVal as number | null, concRange.min, concRange.max);
                  } else if (gridMode === 'intensity' && typeof cellVal === 'number') {
                    bg = concColor(cellVal, 0, cellVal * 2 || 1);
                  } else if (gridMode === 'loq') {
                    bg = cellVal === true ? '#166534' : cellVal === false ? '#991b1b' : '#374151';
                  } else if (gridMode === 'wellType') {
                    bg = wellTypeColor(cellVal as string | null);
                  }
                  return (
                    <button
                      key={wellId}
                      type="button"
                      title={wellId}
                      aria-label={`Well ${wellId}`}
                      aria-current={selectedWell === wellId ? 'true' : undefined}
                      onClick={() => setSelectedWell(wellId)}
                      style={{ backgroundColor: bg }}
                      className={`min-w-0 aspect-square rounded-sm border ${
                        selectedWell === wellId
                          ? 'border-white ring-1 ring-slack-accent'
                          : 'border-slack-border/50 hover:border-slack-accent'
                      }`}
                    />
                  );
                })}
              </span>
            ))}
          </div>
        </div>

        <div className="flex-1 min-w-0 flex flex-col overflow-hidden">
          <div className="flex-shrink-0 border-b border-slack-border px-2 py-1 flex flex-wrap gap-1">
            {data.analytes.map((a) => {
              const analyteQc = panelQCReport?.analytes.find((r) => r.analyte === a);
              return (
              <button
                key={a}
                type="button"
                onClick={() => setSelectedAnalyte(a)}
                className={`text-xs px-2 py-0.5 rounded border ${
                  selectedAnalyte === a
                    ? 'border-slack-accent text-white'
                    : 'border-slack-border text-slack-textMuted hover:border-slack-accent'
                }`}
                style={selectedAnalyte === a ? { backgroundColor: analyteColor(a) } : undefined}
              >
                {a}
                {analyteQc && (
                  <span className={analyteQc.pass ? ' text-green-300' : ' text-red-300'}>
                    {analyteQc.pass ? ' ✓' : ' ✗'}
                  </span>
                )}
              </button>
            );})}
          </div>
          <div className="flex-1 overflow-auto p-3 space-y-4">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
              {heatMapSrc && (
                <div>
                  <div className="text-xs font-semibold mb-1">Heat map</div>
                  <img src={heatMapSrc} alt={`${selectedAnalyte} heat map`} className="max-w-full rounded border border-slack-border" />
                </div>
              )}
              {calibrationSrc && (
                <div>
                  <div className="text-xs font-semibold mb-1">Calibration curve</div>
                  <img src={calibrationSrc} alt={`${selectedAnalyte} calibration`} className="max-w-full rounded border border-slack-border" />
                </div>
              )}
            </div>

            <div>
              <div className="text-xs font-semibold mb-1">Standard report ({selectedAnalyte})</div>
              <div className="overflow-auto max-h-48 text-[10px] font-mono border border-slack-border rounded">
                <table className="w-full">
                  <thead className="bg-slack-bgHover sticky top-0">
                    <tr>
                      <th className="text-left p-1">Well</th>
                      <th className="text-left p-1">Conc</th>
                      <th className="text-left p-1">Calc</th>
                      <th className="text-left p-1">Bias%</th>
                      <th className="text-left p-1">LOQ</th>
                    </tr>
                  </thead>
                  <tbody>
                    {standardRows.map((r) => (
                      <tr key={r.wellLabel} className="border-t border-slack-border/50">
                        <td className="p-1">{r.wellLabel}</td>
                        <td className="p-1">{formatConc(r.concentration)}</td>
                        <td className="p-1">{formatConc(r.meanReplicateCalculatedConcentration)}</td>
                        <td className="p-1">{formatConc(r.percentBias)}</td>
                        <td className="p-1">{r.withinLimitsOfQuantificationV2 ? 'Yes' : 'No'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <div className="text-xs font-semibold mb-1">Unknown report ({selectedAnalyte})</div>
              <div className="overflow-auto max-h-48 text-[10px] font-mono border border-slack-border rounded">
                <table className="w-full">
                  <thead className="bg-slack-bgHover sticky top-0">
                    <tr>
                      <th className="text-left p-1">Well</th>
                      <th className="text-left p-1">Conc pg/ml</th>
                      <th className="text-left p-1">LOQ</th>
                    </tr>
                  </thead>
                  <tbody>
                    {unknownRows.map((r) => (
                      <tr key={r.wellLabel} className="border-t border-slack-border/50">
                        <td className="p-1">{r.wellLabel}</td>
                        <td className="p-1">{formatConc(r.meanReplicateConcentration)}</td>
                        <td className="p-1">{r.withinLimitsOfQuantification ? 'Yes' : 'No'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <div className="w-56 flex-shrink-0 border-l border-slack-border flex flex-col min-h-0 overflow-auto">
          <div className="p-2 border-b border-slack-border text-xs font-semibold">Well {selectedWell}</div>
          <div className="p-2 text-xs space-y-1">
            {wellRow ? (
              <>
                <div><span className="text-slack-textMuted">Label:</span> {wellRow.wellLabel}</div>
                <div><span className="text-slack-textMuted">Type:</span> {wellRow.wellType}</div>
                <div><span className="text-slack-textMuted">Signal:</span> {formatConcDisplay(wellRow.signal)}</div>
                <div>
                  <span className="text-slack-textMuted">{selectedAnalyte}:</span>{' '}
                  {formatConc(concentrationAt(data, selectedWell, selectedAnalyte))} pg/ml
                </div>
              </>
            ) : (
              <div className="text-slack-textMuted">No data for this well/analyte</div>
            )}
          </div>
          {loq && (
            <div className="p-2 border-t border-slack-border text-[10px] space-y-0.5">
              <div className="font-semibold text-xs">LOQ ({selectedAnalyte})</div>
              <div>LLOQ: {loq.LLOQ} {loq.concentration_units}</div>
              <div>ULOQ: {loq.ULOQ} {loq.concentration_units}</div>
              <div>LOD: {loq.LOD}</div>
            </div>
          )}
          {fit && (
            <div className="p-2 border-t border-slack-border text-[10px]">
              <button
                type="button"
                className="text-xs font-semibold hover:text-slack-accent"
                onClick={() => setShowFitParams((v) => !v)}
              >
                5PL fit params {showFitParams ? '▾' : '▸'}
              </button>
              {showFitParams && (
                <pre className="mt-1 font-mono whitespace-pre-wrap">
                  {JSON.stringify(fit, null, 2)}
                </pre>
              )}
            </div>
          )}
          <div className="p-2 border-t border-slack-border text-[10px] font-semibold">
            Spots ({spots.length})
          </div>
          <div className="overflow-auto max-h-32 p-2 text-[10px] font-mono">
            {spots.map((s, i) => (
              <div key={i} className="truncate">
                r{s.row}c{s.column} sig={formatConcDisplay(s.signal)} bg={formatConcDisplay(s.background)}
              </div>
            ))}
          </div>
        </div>
      </div>

      {panelQCReport && (
        <div className="flex-shrink-0 border-t border-slack-border">
          <button
            type="button"
            className="w-full text-left px-3 py-1 text-xs font-semibold hover:bg-slack-bgHover flex items-center gap-2"
            onClick={() => setShowQCReport((v) => !v)}
          >
            <span className={panelQCReport.overall_pass ? 'text-green-400' : 'text-red-400'}>
              12-Plex QC: {panelQCReport.overall_pass ? 'PASS' : 'FAIL'}
            </span>
            {showQCReport ? '▾' : '▸'}
          </button>
          {showQCReport && (
            <div className="px-3 pb-2 max-h-48 overflow-auto space-y-2">
              {panelQCReport.analytes.map((a) => (
                <div key={a.analyte} className="text-[10px] border border-slack-border/50 rounded p-1">
                  <div className="font-semibold flex justify-between">
                    <span>{a.analyte}</span>
                    <span className={a.pass ? 'text-green-400' : 'text-red-400'}>
                      {a.pass ? 'Pass' : 'Fail'}
                    </span>
                  </div>
                  {a.checks
                    .filter((c) => !c.pass)
                    .map((c) => (
                      <div key={c.name} className="text-red-300/90 font-mono">
                        {c.name}: {c.detail || c.value}
                      </div>
                    ))}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {processReport && (
        <div className="flex-shrink-0 border-t border-slack-border">
          <button
            type="button"
            className="w-full text-left px-3 py-1 text-xs font-semibold hover:bg-slack-bgHover"
            onClick={() => setShowProcessReport((v) => !v)}
          >
            Process report {showProcessReport ? '▾' : '▸'}
          </button>
          {showProcessReport && (
            <pre className="max-h-40 overflow-auto px-3 pb-2 text-[10px] font-mono whitespace-pre-wrap text-slack-textMuted">
              {processReport}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
