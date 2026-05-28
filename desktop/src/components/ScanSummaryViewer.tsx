import { useCallback, useEffect, useMemo, useState } from 'react';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useToastStore } from '../stores/toastStore';
import {
  analyteColor,
  PLATE_COLS,
  PLATE_ROWS,
  runLabelFromDirPath,
  type ScanSummaryData,
  type ScanSummaryWellMeta,
} from '../utils/scanSummary';
import { resolveScanSummaryWellImageSrc } from '../utils/scanSummaryImage';
import { useEditorStore } from '../stores/editorStore';
import { allWellAnalyteConcentrations } from '../utils/scanAnalysis';
import { formatConcDisplay } from '../utils/scanAnalysisHelpers';

interface ScanSummaryViewerProps {
  workspaceId: string;
  summaryDir: string;
  data: ScanSummaryData;
  initialWell?: string;
  linkedAnalysisDir?: string;
}

export function ScanSummaryViewer({
  workspaceId,
  summaryDir,
  data,
  initialWell = 'A1',
  linkedAnalysisDir,
}: ScanSummaryViewerProps) {
  const workspacePath = useFileExplorerStore((s) => {
    const ws = s.workspaces.find((w) => w.id === workspaceId);
    return ws?.path ?? '';
  });
  const { addToast } = useToastStore();
  const { findLinkedAnalysisTab, activateAnalysisWell } = useEditorStore();

  const [selectedWell, setSelectedWell] = useState(initialWell);
  useEffect(() => {
    setSelectedWell(initialWell);
  }, [initialWell, summaryDir]);
  const [imageSrc, setImageSrc] = useState<string | null>(null);
  const [loadingImage, setLoadingImage] = useState(false);
  const [showOverlay, setShowOverlay] = useState(true);
  const [showLabels, setShowLabels] = useState(false);

  const wellMeta: ScanSummaryWellMeta | undefined = data.byWell.get(selectedWell);
  const runLabel = runLabelFromDirPath(summaryDir);

  const loadWellImage = useCallback(async () => {
    if (!workspacePath) return;
    setLoadingImage(true);
    setImageSrc(null);
    try {
      const src = await resolveScanSummaryWellImageSrc({
        workspaceId,
        workspacePath,
        summaryDir,
        wellId: selectedWell,
      });
      setImageSrc(src);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load well image';
      addToast({ type: 'error', title: 'Well image', message });
    } finally {
      setLoadingImage(false);
    }
  }, [workspaceId, workspacePath, summaryDir, selectedWell, addToast]);

  useEffect(() => {
    void loadWellImage();
  }, [loadWellImage]);

  const legendAnalytes = useMemo(() => {
    const set = new Set<string>();
    for (const w of data.metadata) {
      for (const s of w.spots) {
        set.add(s.analyte);
      }
    }
    return Array.from(set).sort();
  }, [data]);

  const linkedAnalysisTab = useMemo(() => {
    if (linkedAnalysisDir) {
      return useEditorStore.getState().tabs.find(
        (t) =>
          t.workspaceId === workspaceId &&
          t.viewMode === 'scan-analysis' &&
          t.scanAnalysisDir === linkedAnalysisDir
      );
    }
    return findLinkedAnalysisTab(workspaceId, summaryDir);
  }, [linkedAnalysisDir, findLinkedAnalysisTab, workspaceId, summaryDir]);

  const analysisAtWell = useMemo(() => {
    if (!linkedAnalysisTab?.scanAnalysisData) return [];
    return allWellAnalyteConcentrations(linkedAnalysisTab.scanAnalysisData, selectedWell);
  }, [linkedAnalysisTab, selectedWell]);

  const handleOpenAnalysis = () => {
    if (!linkedAnalysisTab) {
      addToast({ type: 'info', title: 'Analysis', message: 'No linked analysis tab. Open scan analysis first.' });
      return;
    }
    activateAnalysisWell(linkedAnalysisTab.id, selectedWell);
  };

  const imageSize = 358;

  return (
    <div className="flex flex-col h-full min-h-0 bg-slack-bg text-slack-text">
      <div className="flex-shrink-0 px-4 py-2 border-b border-slack-border flex flex-wrap items-center gap-3">
        <div className="min-w-0">
          <div className="text-sm font-semibold truncate">{runLabel}</div>
          <div className="text-xs text-slack-textMuted">
            Well {selectedWell}
            {wellMeta?.time ? ` · ${wellMeta.time}` : ''}
          </div>
        </div>
        <div className="flex items-center gap-2 ml-auto text-xs">
          <label htmlFor="scan-summary-overlay" className="flex items-center gap-1 cursor-pointer">
            <input
              id="scan-summary-overlay"
              type="checkbox"
              checked={showOverlay}
              onChange={(e) => setShowOverlay(e.target.checked)}
            />
            Spot overlay
          </label>
          <label htmlFor="scan-summary-labels" className="flex items-center gap-1 cursor-pointer">
            <input
              id="scan-summary-labels"
              type="checkbox"
              checked={showLabels}
              onChange={(e) => setShowLabels(e.target.checked)}
            />
            Labels
          </label>
        </div>
      </div>

      <div className="flex flex-1 min-h-0">
        <div className="w-40 flex-shrink-0 border-r border-slack-border overflow-auto p-2">
          <div
            className="grid gap-0.5 text-[10px]"
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
                  const hasData = data.byWell.has(wellId);
                  return (
                    <button
                      key={wellId}
                      type="button"
                      title={wellId}
                      aria-label={`Well ${wellId}`}
                      aria-current={selectedWell === wellId ? 'true' : undefined}
                      onClick={() => setSelectedWell(wellId)}
                      className={`min-w-0 aspect-square rounded-sm border ${
                        selectedWell === wellId
                          ? 'border-slack-accent bg-slack-accent text-white'
                          : hasData
                            ? 'border-slack-border bg-slack-bgHover hover:border-slack-accent'
                            : 'border-slack-border/50 bg-slack-bg text-slack-textMuted'
                      }`}
                    />
                  );
                })}
              </span>
            ))}
          </div>
        </div>

        <div className="flex-1 min-w-0 flex items-center justify-center p-4 overflow-auto bg-slack-bg">
          {loadingImage && (
            <div className="text-sm text-slack-textMuted">Loading well image…</div>
          )}
          {!loadingImage && imageSrc && (
            <div
              className="relative inline-block max-w-full max-h-full shadow-lg"
              style={{ width: imageSize, height: imageSize }}
            >
              <img
                src={imageSrc}
                alt={`Well ${selectedWell}`}
                className="w-full h-full object-contain"
                width={imageSize}
                height={imageSize}
              />
              {showOverlay && wellMeta && (
                <svg
                  className="absolute inset-0 w-full h-full pointer-events-none"
                  viewBox={`0 0 ${imageSize} ${imageSize}`}
                >
                  {wellMeta.spots.map((spot, i) => (
                    <g key={`${spot.row}-${spot.column}-${i}`}>
                      <circle
                        cx={spot.x_px}
                        cy={spot.y_px}
                        r={6}
                        fill={analyteColor(spot.analyte)}
                        fillOpacity={0.55}
                        stroke="#fff"
                        strokeWidth={1}
                      />
                      {showLabels && (
                        <text
                          x={spot.x_px}
                          y={spot.y_px - 8}
                          textAnchor="middle"
                          fontSize={8}
                          fill="#fff"
                        >
                          {spot.analyte === 'BLANK' ? 'B' : spot.analyte === 'POS' ? 'P' : spot.analyte.slice(0, 3)}
                        </text>
                      )}
                    </g>
                  ))}
                </svg>
              )}
            </div>
          )}
          {!loadingImage && !imageSrc && (
            <div className="text-sm text-slack-textMuted">No image for well {selectedWell}</div>
          )}
        </div>

        <div className="w-56 flex-shrink-0 border-l border-slack-border flex flex-col min-h-0">
          <div className="p-2 border-b border-slack-border text-xs font-semibold">Legend</div>
          <div className="p-2 overflow-auto flex-1 text-xs space-y-1">
            {legendAnalytes.map((a) => (
              <div key={a} className="flex items-center gap-2">
                <span
                  className="w-3 h-3 rounded-full shrink-0"
                  style={{ backgroundColor: analyteColor(a) }}
                />
                <span className="truncate">{a}</span>
              </div>
            ))}
          </div>
          <div className="p-2 border-t border-slack-border text-xs font-semibold">
            Spots ({wellMeta?.spots.length ?? 0})
          </div>
          <div className="overflow-auto max-h-40 p-2 text-[10px] font-mono">
            {wellMeta?.spots.map((s, i) => (
              <div key={i} className="truncate">
                {s.analyte} r{s.row}c{s.column} ({s.x_px},{s.y_px})
              </div>
            ))}
          </div>
          {analysisAtWell.length > 0 && (
            <>
              <div className="p-2 border-t border-slack-border text-xs font-semibold flex items-center justify-between gap-1">
                <span>Analysis at well</span>
                <button
                  type="button"
                  className="text-[10px] px-1 py-0.5 border border-slack-border rounded hover:border-slack-accent"
                  onClick={handleOpenAnalysis}
                >
                  Open
                </button>
              </div>
              <div className="overflow-auto max-h-40 p-2 text-[10px] space-y-0.5">
                {analysisAtWell.map((row: { analyte: string; concentration: number | null; withinLoq: boolean | null }) => (
                  <div key={row.analyte} className="flex justify-between gap-1">
                    <span className="truncate">{row.analyte}</span>
                    <span className="font-mono shrink-0">
                      {formatConcDisplay(row.concentration)}
                      {row.withinLoq != null && (row.withinLoq ? ' ✓' : ' ✗')}
                    </span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
