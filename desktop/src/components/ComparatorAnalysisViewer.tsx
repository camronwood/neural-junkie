import { useEffect, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useSecondaryAnalysisStore } from '../stores/secondaryAnalysisStore';
import type { ComparatorSummary } from '../utils/secondaryAnalysis';

const api = new ChatAPI(getHubBaseURL());

interface ComparatorAnalysisViewerProps {
  workspaceId: string;
  analysisDir: string;
}

function CsvTable({ rows, maxRows = 25 }: { rows: string[][]; maxRows?: number }) {
  if (!rows.length) return <p className="text-xs text-slack-textMuted">No data</p>;
  return (
    <div className="overflow-auto max-h-48 border border-slack-border rounded">
      <table className="w-full text-[10px] font-mono">
        <tbody>
          {rows.slice(0, maxRows).map((row, i) => (
            <tr key={i} className="border-t border-slack-border/50">
              {row.map((cell, j) => (
                <td key={j} className="p-1">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ArtifactImage({ workspaceId, path }: { workspaceId: string; path: string }) {
  const [src, setSrc] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const url = await api.fetchWorkspaceImageDataUrl(workspaceId, path);
        if (!cancelled) setSrc(url);
      } catch {
        if (!cancelled) setSrc(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [workspaceId, path]);
  if (!src) return <span className="text-[10px] text-slack-textMuted">{path}</span>;
  return (
    <img src={src} alt={path} className="max-w-full rounded border border-slack-border mb-2" />
  );
}

export function ComparatorAnalysisViewer({ workspaceId, analysisDir }: ComparatorAnalysisViewerProps) {
  const [summary, setSummary] = useState<ComparatorSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selectedArtifact, setSelectedArtifact] = useState<string | null>(null);
  const [expandedPlate, setExpandedPlate] = useState<string | null>(null);
  const setPanelOpen = useSecondaryAnalysisStore((s) => s.setPanelOpen);
  const setWorkflow = useSecondaryAnalysisStore((s) => s.setWorkflow);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const s = await api.fetchComparatorSummary(workspaceId, analysisDir);
        if (!cancelled) setSummary(s);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load');
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [workspaceId, analysisDir]);

  const plateNames = summary?.plate_stats ? Object.keys(summary.plate_stats) : [];
  const interplateNames = summary?.interplate_stats ? Object.keys(summary.interplate_stats) : [];

  return (
    <div className="flex flex-col h-full min-h-0 bg-slack-bg text-slack-text p-4 overflow-auto">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-sm font-semibold">Comparator analysis</h2>
          <p className="text-xs text-slack-textMuted font-mono truncate">{analysisDir}</p>
        </div>
        <button
          type="button"
          className="text-xs px-2 py-1 border border-slack-border rounded hover:border-slack-accent"
          onClick={() => {
            setWorkflow('endogenous');
            setPanelOpen(true);
          }}
        >
          Run endogenous…
        </button>
      </div>

      {error && <p className="text-sm text-red-400">{error}</p>}

      {summary && (
        <div className="space-y-4 text-sm">
          {summary.source_plates && summary.source_plates.length > 0 && (
            <div>
              <div className="text-xs font-semibold text-slack-textMuted mb-1">Source plates</div>
              <ul className="text-xs font-mono list-disc pl-4">
                {summary.source_plates.map((p) => (
                  <li key={p}>{p}</li>
                ))}
              </ul>
            </div>
          )}

          {summary.conditions && summary.conditions.length > 0 && (
            <div>
              <div className="text-xs font-semibold text-slack-textMuted mb-1">Conditions</div>
              <ul className="list-disc pl-4 text-xs">
                {summary.conditions.map((c) => (
                  <li key={c}>{c}</li>
                ))}
              </ul>
            </div>
          )}

          <div>
            <div className="text-xs font-semibold text-slack-textMuted mb-1">LLOQ / ULOQ</div>
            {summary.lloq_uloq_rows && summary.lloq_uloq_rows.length > 0 ? (
              <CsvTable rows={summary.lloq_uloq_rows} />
            ) : (
              <p className="text-xs text-slack-textMuted">No LLOQ table loaded</p>
            )}
          </div>

          {plateNames.length > 0 && (
            <div>
              <div className="text-xs font-semibold text-slack-textMuted mb-1">
                Plate summary stats ({plateNames.length})
              </div>
              <div className="space-y-2">
                {plateNames.map((p) => (
                  <div key={p} className="border border-slack-border rounded">
                    <button
                      type="button"
                      className="w-full text-left px-2 py-1 text-xs font-mono hover:bg-slack-bgHover"
                      onClick={() => setExpandedPlate(expandedPlate === p ? null : p)}
                    >
                      {p} {expandedPlate === p ? '▾' : '▸'}
                    </button>
                    {expandedPlate === p && summary.plate_stats?.[p] && (
                      <div className="px-2 pb-2">
                        <CsvTable rows={summary.plate_stats[p]} maxRows={50} />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {interplateNames.length > 0 && (
            <div>
              <div className="text-xs font-semibold text-slack-textMuted mb-1">Interplate statistics</div>
              {interplateNames.map((name) => (
                <div key={name} className="mb-2">
                  <div className="text-[10px] font-mono text-slack-textMuted mb-1">{name}</div>
                  <CsvTable rows={summary.interplate_stats?.[name] ?? []} />
                </div>
              ))}
            </div>
          )}

          {summary.artifacts && summary.artifacts.length > 0 && (
            <div>
              <div className="text-xs font-semibold text-slack-textMuted mb-1">Artifacts</div>
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                <ul className="text-xs font-mono space-y-1 max-h-64 overflow-auto">
                  {summary.artifacts.map((g) =>
                    g.files.map((f) => (
                      <li key={f.relative_path}>
                        <button
                          type="button"
                          className={`text-left truncate w-full hover:text-slack-accent ${
                            selectedArtifact === f.relative_path ? 'text-slack-accent' : ''
                          }`}
                          onClick={() => setSelectedArtifact(f.relative_path)}
                          title={f.relative_path}
                        >
                          {g.condition}/{g.plate}/{f.name}
                        </button>
                      </li>
                    ))
                  )}
                </ul>
                {selectedArtifact && (
                  <div>
                    {summary.artifacts
                      .flatMap((g) => g.files)
                      .find((f) => f.relative_path === selectedArtifact)?.kind === 'image' ? (
                      <ArtifactImage workspaceId={workspaceId} path={selectedArtifact} />
                    ) : (
                      <p className="text-xs text-slack-textMuted">{selectedArtifact}</p>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
