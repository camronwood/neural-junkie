import { useCallback, useEffect, useRef, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { usePacksStore } from '../stores/packsStore';
import { useToastStore } from '../stores/toastStore';
import { PACK_CAP } from '../stores/packCapabilities';
import { loadScanAnalysisData } from '../utils/scanAnalysisLoad';
import { SearchableTimPicker } from './SearchableTimPicker';

const api = new ChatAPI(getHubBaseURL());

type BrowseTab = 'analyses' | 'scans';

interface TimItem {
  id: string;
  label: string;
}

interface PhoenixBrowserModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function PhoenixBrowserModal({ isOpen, onClose }: PhoenixBrowserModalProps) {
  const activeWorkspaceId = useFileExplorerStore((s) => s.activeWorkspaceId);
  const refreshTreeForPath = useFileExplorerStore((s) => s.refreshTreeForPath);
  const openScanAnalysis = useEditorStore((s) => s.openScanAnalysis);
  const hasPhoenixApi = usePacksStore((s) => s.hasCapability(PACK_CAP.PHOENIX_IMPORT));
  const hasScanAnalysisViewer = usePacksStore((s) => s.hasCapability(PACK_CAP.SCAN_ANALYSIS_VIEWER));
  const { addToast } = useToastStore();

  const [loading, setLoading] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [identity, setIdentity] = useState('');
  const [environment, setEnvironment] = useState('dev');

  const [loginSessionId, setLoginSessionId] = useState<string | null>(null);
  const [userCode, setUserCode] = useState('');
  const [verificationURL, setVerificationURL] = useState('');
  const [loginHint, setLoginHint] = useState<string | null>(null);
  const pollRef = useRef<number | null>(null);

  const [tab, setTab] = useState<BrowseTab>('analyses');
  const [analyses, setAnalyses] = useState<TimItem[]>([]);
  const [scans, setScans] = useState<TimItem[]>([]);
  const [selectedId, setSelectedId] = useState('');
  const [outputDir, setOutputDir] = useState('');
  const [busy, setBusy] = useState(false);
  const [listError, setListError] = useState<string | null>(null);

  const stopPolling = useCallback(() => {
    if (pollRef.current != null) {
      window.clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const loadBrowseLists = useCallback(async () => {
    setListError(null);
    let analysesErr: string | null = null;
    let scanItems: TimItem[] = [];
    let analysisItems: TimItem[] = [];

    try {
      analysisItems = await api.fetchPhoenixAnalyses();
    } catch (e) {
      analysesErr = e instanceof Error ? e.message : 'Failed to load analyses';
    }

    try {
      scanItems = await api.fetchPhoenixScanResults();
    } catch (e) {
      const scanErr = e instanceof Error ? e.message : 'Failed to load scan results';
      if (analysesErr) {
        setListError(`${analysesErr}; ${scanErr}`);
        return;
      }
      setListError(scanErr);
      return;
    }

    setAnalyses(analysisItems);
    setScans(scanItems);

    if (analysesErr) {
      setTab('scans');
      setListError(
        'Analyses list is unavailable on this TIM environment (server error). Use Scan results, or import a known analysis ID later.',
      );
      if (scanItems.length > 0) setSelectedId(scanItems[0].id);
      return;
    }

    const first = tab === 'analyses' ? analysisItems[0]?.id : scanItems[0]?.id;
    if (first) setSelectedId(first);
  }, [tab]);

  const refreshStatus = useCallback(async () => {
    if (!hasPhoenixApi) return;
    setLoading(true);
    setLoginHint(null);
    try {
      const st = await api.fetchPhoenixStatus();
      setEnvironment(st.environment);
      setAuthenticated(st.authenticated);
      setIdentity(st.identity ?? '');
      if (st.authenticated) {
        stopPolling();
        setLoginSessionId(null);
        await loadBrowseLists();
      }
    } catch (e) {
      setLoginHint(e instanceof Error ? e.message : 'Failed to check Phoenix status');
    } finally {
      setLoading(false);
    }
  }, [hasPhoenixApi, loadBrowseLists, stopPolling]);

  const beginLogin = useCallback(async () => {
    setLoginHint(null);
    try {
      const start = await api.phoenixLoginStart();
      setLoginSessionId(start.session_id);
      setUserCode(start.user_code);
      setVerificationURL(start.verification_url);
      setEnvironment(start.environment);
    } catch (e) {
      setLoginHint(e instanceof Error ? e.message : 'Could not start login');
    }
  }, []);

  useEffect(() => {
    if (!isOpen) {
      stopPolling();
      return;
    }
    void refreshStatus();
    return () => stopPolling();
  }, [isOpen, refreshStatus, stopPolling]);

  useEffect(() => {
    if (!isOpen || authenticated || loginSessionId) return;
    if (!hasPhoenixApi) return;
    void beginLogin();
  }, [isOpen, authenticated, loginSessionId, hasPhoenixApi, beginLogin]);

  useEffect(() => {
    stopPolling();
    if (!loginSessionId || authenticated) return;

    const poll = async () => {
      try {
        const result = await api.phoenixLoginPoll(loginSessionId);
        if (result.status === 'success') {
          stopPolling();
          setLoginSessionId(null);
          setAuthenticated(true);
          setIdentity(result.identity ?? '');
          await loadBrowseLists();
          addToast({ type: 'success', title: 'Phoenix', message: 'Signed in to TIM' });
        } else if (result.status === 'pending') {
          if (result.hint) setLoginHint(result.hint);
        } else {
          stopPolling();
          setLoginSessionId(null);
          setLoginHint(result.hint ?? `Login ${result.status}`);
        }
      } catch (e) {
        setLoginHint(e instanceof Error ? e.message : 'Login poll failed');
      }
    };

    void poll();
    pollRef.current = window.setInterval(() => void poll(), 3000);
    return () => stopPolling();
  }, [loginSessionId, authenticated, loadBrowseLists, addToast, stopPolling]);

  useEffect(() => {
    if (!authenticated) return;
    const items = tab === 'analyses' ? analyses : scans;
    if (items.length > 0 && !items.some((i) => i.id === selectedId)) {
      setSelectedId(items[0].id);
    }
  }, [tab, analyses, scans, selectedId, authenticated]);

  const openBrowser = () => {
    if (!verificationURL) return;
    void import('@tauri-apps/api/shell').then(({ open }) => open(verificationURL));
  };

  const handleLogout = async () => {
    try {
      await api.phoenixLogout();
      setAuthenticated(false);
      setIdentity('');
      setAnalyses([]);
      setScans([]);
      setSelectedId('');
      await beginLogin();
    } catch (e) {
      addToast({
        type: 'error',
        title: 'Phoenix',
        message: e instanceof Error ? e.message : 'Logout failed',
      });
    }
  };

  const handleDownload = async () => {
    if (!activeWorkspaceId || !selectedId) return;
    setBusy(true);
    try {
      if (tab === 'analyses') {
        const result = await api.phoenixImport({
          workspace_id: activeWorkspaceId,
          analysis_id: selectedId,
          output_dir: outputDir.trim() || undefined,
        });
        await refreshTreeForPath(activeWorkspaceId, `${result.analysis_dir}/reports/results.json`);
        if (result.validation_dir) {
          await refreshTreeForPath(
            activeWorkspaceId,
            `${result.validation_dir}/reports/validation_report.csv`,
          );
        }
        const validationNote = result.validation_dir
          ? ` and ${result.validation_dir}`
          : '';
        addToast({
          type: 'success',
          title: 'Phoenix',
          message: `Downloaded analysis to ${result.analysis_dir}${validationNote}`,
        });
        if (hasScanAnalysisViewer) {
          const { data, linkedScanDir } = await loadScanAnalysisData(
            api,
            activeWorkspaceId,
            result.analysis_dir,
          );
          openScanAnalysis(activeWorkspaceId, result.analysis_dir, data, {
            linkedScanDir: linkedScanDir ?? result.scan_export_dir,
          });
        }
      } else {
        const result = await api.phoenixImportScan({
          workspace_id: activeWorkspaceId,
          scan_results_id: selectedId,
          output_dir: outputDir.trim() || undefined,
        });
        const path = result.scan_export_dir ?? result.analysis_dir;
        await refreshTreeForPath(activeWorkspaceId, path);
        addToast({
          type: 'success',
          title: 'Phoenix',
          message: `Downloaded scan to ${path}`,
        });
      }
      setOutputDir('');
    } catch (e) {
      addToast({
        type: 'error',
        title: 'Phoenix',
        message: e instanceof Error ? e.message : 'Download failed',
      });
    } finally {
      setBusy(false);
    }
  };

  if (!isOpen) return null;

  const items = tab === 'analyses' ? analyses : scans;

  return (
    <div className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto py-6 px-4" role="presentation">
      <div className="fixed inset-0 bg-black/60" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="phoenix-browser-title"
        className="relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-indigo-700/50 bg-slack-bg shadow-2xl max-h-[min(90vh,820px)]"
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slack-border px-4 py-3">
          <div>
            <h2 id="phoenix-browser-title" className="text-lg font-semibold text-indigo-200">
              Phoenix TIM
            </h2>
            <p className="text-xs text-gray-500 mt-0.5">
              {environment} · custom pack (phoenix-import)
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slack-textMuted hover:text-slack-text px-2 py-1 rounded hover:bg-slack-bgHover"
          >
            ✕
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {!hasPhoenixApi && (
            <p className="text-sm text-amber-400">
              Enable a custom pack with <code className="font-mono">phoenix-import</code> (e.g. brightest-bio-lab) and Life sciences in Settings → Domain packs.
            </p>
          )}

          {hasPhoenixApi && !authenticated && (
            <div className="space-y-4 rounded-lg border border-indigo-800/40 bg-indigo-950/20 p-4">
              <h3 className="text-sm font-semibold text-indigo-200">Sign in</h3>
              <p className="text-xs text-gray-400">
                Open the link below and enter the code to authorize Neural Junkie for Phoenix TIM read access.
              </p>
              {userCode && (
                <p className="text-2xl font-mono font-bold tracking-widest text-center text-white py-2">
                  {userCode}
                </p>
              )}
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  disabled={!verificationURL}
                  onClick={openBrowser}
                  className="px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-500 disabled:opacity-40"
                >
                  Open browser
                </button>
                <button
                  type="button"
                  disabled={loading}
                  onClick={() => void beginLogin()}
                  className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover"
                >
                  New code
                </button>
              </div>
              {loading && <p className="text-xs text-gray-500">Checking credentials…</p>}
              {!loading && loginSessionId && (
                <p className="text-xs text-indigo-300/80">Waiting for authorization…</p>
              )}
              {loginHint && <p className="text-xs text-amber-400">{loginHint}</p>}
            </div>
          )}

          {hasPhoenixApi && authenticated && (
            <>
              <div className="flex items-center justify-between gap-2 text-xs">
                <p className="text-gray-400 font-mono truncate">{identity.split('\n')[0] ?? 'Signed in'}</p>
                <button
                  type="button"
                  onClick={() => void handleLogout()}
                  className="shrink-0 text-gray-500 hover:text-red-400"
                >
                  Sign out
                </button>
              </div>

              <div className="flex rounded-md border border-slack-border overflow-hidden text-xs" role="tablist">
                {(['analyses', 'scans'] as const).map((t) => (
                  <button
                    key={t}
                    type="button"
                    role="tab"
                    aria-selected={tab === t}
                    onClick={() => setTab(t)}
                    className={`flex-1 px-3 py-1.5 font-medium ${
                      tab === t
                        ? 'bg-indigo-600 text-white'
                        : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                    }`}
                  >
                    {t === 'analyses' ? 'Analyses' : 'Scan results'}
                  </button>
                ))}
              </div>

              {listError && <p className="text-xs text-red-400">{listError}</p>}

              <label className="block text-xs">
                <span className="text-gray-400 mb-1 block">
                  {tab === 'analyses' ? 'Analysis' : 'Scan result'}
                </span>
                <SearchableTimPicker
                  key={tab}
                  items={items}
                  value={selectedId}
                  onChange={setSelectedId}
                  searchPlaceholder={
                    tab === 'analyses' ? 'Search analyses…' : 'Search scan results…'
                  }
                  emptyLabel="Nothing listed"
                />
              </label>

              <label className="block text-xs">
                <span className="text-gray-400">Workspace folder (optional)</span>
                <input
                  type="text"
                  placeholder={tab === 'analyses' ? 'phoenix-plate-001' : 'phoenix-scan-001'}
                  value={outputDir}
                  onChange={(e) => setOutputDir(e.target.value)}
                  className="mt-1 w-full px-2 py-1.5 rounded border border-slack-border bg-slack-bg text-sm font-mono"
                />
              </label>

              {!activeWorkspaceId && (
                <p className="text-xs text-amber-400">Open a workspace in the file explorer first.</p>
              )}

              <div className="flex gap-2">
                <button
                  type="button"
                  disabled={busy || !selectedId || !activeWorkspaceId}
                  onClick={() => void handleDownload()}
                  className="px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-500 disabled:opacity-40"
                >
                  {busy ? 'Downloading…' : 'Download to workspace'}
                </button>
                <button
                  type="button"
                  disabled={loading}
                  onClick={() => void loadBrowseLists()}
                  className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover"
                >
                  Refresh
                </button>
              </div>

              {tab === 'analyses' && !hasScanAnalysisViewer && (
                <p className="text-xs text-amber-400">
                  Enable a pack with <code className="font-mono">scan-analysis-viewer</code> to open the scan analysis viewer after download.
                </p>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
