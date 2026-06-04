import { useCallback, useEffect, useState } from 'react';
import { ChatAPI } from '../../api/chatAPI';
import { getHubBaseURL } from '../../config/hubUrl';
import { useEditorStore } from '../../stores/editorStore';
import { useFileExplorerStore } from '../../stores/fileExplorerStore';
import { usePacksStore } from '../../stores/packsStore';
import { useToastStore } from '../../stores/toastStore';
import { PACK_CAP } from '../../stores/packCapabilities';
import { loadScanAnalysisData } from '../../utils/scanAnalysisLoad';

const api = new ChatAPI(getHubBaseURL());

interface PhoenixStatus {
  environment: string;
  credentials_path?: string;
  authenticated: boolean;
  logged_in: boolean;
  identity?: string;
  hint?: string;
}

interface PhoenixAnalysis {
  id: string;
  label: string;
}

export function PhoenixImportPanel() {
  const activeWorkspaceId = useFileExplorerStore((s) => s.activeWorkspaceId);
  const refreshTreeForPath = useFileExplorerStore((s) => s.refreshTreeForPath);
  const openScanAnalysis = useEditorStore((s) => s.openScanAnalysis);
  const hasPhoenix = usePacksStore((s) =>
    s.hasCapability(PACK_CAP.PHOENIX_IMPORT) || s.hasCapability(PACK_CAP.CUSTOMER_PACK)
  );
  const hasLifeSciences = usePacksStore((s) => s.hasCapability(PACK_CAP.SCAN_ANALYSIS_VIEWER));
  const { addToast } = useToastStore();

  const [status, setStatus] = useState<PhoenixStatus | null>(null);
  const [analyses, setAnalyses] = useState<PhoenixAnalysis[]>([]);
  const [analysisId, setAnalysisId] = useState('');
  const [outputDir, setOutputDir] = useState('');
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(false);

  const refresh = useCallback(async () => {
    if (!hasPhoenix) return;
    setLoading(true);
    try {
      const st = await api.fetchPhoenixStatus();
      setStatus(st);
      if (st.logged_in) {
        const list = await api.fetchPhoenixAnalyses();
        setAnalyses(list);
        if (!analysisId && list.length > 0) {
          setAnalysisId(list[0].id);
        }
      } else {
        setAnalyses([]);
      }
    } catch (e) {
      addToast({
        type: 'error',
        title: 'Phoenix',
        message: e instanceof Error ? e.message : 'Failed to load Phoenix status',
      });
    } finally {
      setLoading(false);
    }
  }, [hasPhoenix, addToast]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleImport = async () => {
    if (!activeWorkspaceId || !analysisId.trim()) return;
    setBusy(true);
    try {
      const result = await api.phoenixImport({
        workspace_id: activeWorkspaceId,
        analysis_id: analysisId.trim(),
        output_dir: outputDir.trim() || undefined,
      });
      await refreshTreeForPath(activeWorkspaceId, `${result.analysis_dir}/reports/results.json`);
      addToast({
        type: 'success',
        title: 'Phoenix import',
        message: `Imported to ${result.analysis_dir}`,
      });
      if (hasLifeSciences) {
        const { data, linkedScanDir } = await loadScanAnalysisData(api, activeWorkspaceId, result.analysis_dir);
        openScanAnalysis(activeWorkspaceId, result.analysis_dir, data, {
          linkedScanDir: linkedScanDir ?? result.scan_export_dir,
        });
      }
      setOutputDir('');
    } catch (e) {
      addToast({
        type: 'error',
        title: 'Phoenix import',
        message: e instanceof Error ? e.message : 'Import failed',
      });
    } finally {
      setBusy(false);
    }
  };

  if (!hasPhoenix) {
    return null;
  }

  return (
    <div className="border border-indigo-700/40 rounded-xl p-4 bg-indigo-950/20 space-y-3">
      <div>
        <h4 className="text-sm font-semibold text-indigo-200">Import from Phoenix</h4>
        <p className="text-xs text-gray-400 mt-1">
          Use the <strong className="text-indigo-300">PHX</strong> toolbar chip to sign in, browse TIM data, and
          download into your workspace. Life sciences viewers open after analysis download.
        </p>
      </div>

      {loading && <p className="text-xs text-gray-500">Checking Phoenix credentials…</p>}

      {status && !status.authenticated && status.hint && (
        <p className="text-xs text-amber-400">{status.hint}</p>
      )}

      {status?.authenticated && (
        <>
          <p className="text-[10px] text-gray-500 font-mono truncate">
            {status.environment} · {status.identity?.split('\n')[0] ?? 'authenticated'}
          </p>
          <label className="block text-xs">
            <span className="text-gray-400">Analysis</span>
            <select
              className="mt-1 w-full px-2 py-1.5 rounded border border-slack-border bg-slack-bg text-sm"
              value={analysisId}
              onChange={(e) => setAnalysisId(e.target.value)}
            >
              {analyses.length === 0 && <option value="">No analyses listed</option>}
              {analyses.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.label} ({a.id.slice(0, 8)}…)
                </option>
              ))}
            </select>
          </label>
          <label className="block text-xs">
            <span className="text-gray-400">Workspace folder (optional)</span>
            <input
              type="text"
              placeholder="phoenix-plate-001"
              value={outputDir}
              onChange={(e) => setOutputDir(e.target.value)}
              className="mt-1 w-full px-2 py-1.5 rounded border border-slack-border bg-slack-bg text-sm font-mono"
            />
          </label>
          <div className="flex gap-2">
            <button
              type="button"
              disabled={busy || !analysisId || !activeWorkspaceId}
              onClick={() => void handleImport()}
              className="px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-500 disabled:opacity-40"
            >
              {busy ? 'Importing…' : 'Import into workspace'}
            </button>
            <button
              type="button"
              disabled={loading}
              onClick={() => void refresh()}
              className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover"
            >
              Refresh
            </button>
          </div>
          {!activeWorkspaceId && (
            <p className="text-xs text-amber-400">Open a workspace in the file explorer first.</p>
          )}
          {!hasLifeSciences && (
            <p className="text-xs text-amber-400">Enable Life sciences to open the scan analysis viewer after import.</p>
          )}
        </>
      )}
    </div>
  );
}
