import { useState, useEffect } from 'react';
import { usePacksStore } from '../../stores/packsStore';
import { PACK_CAP } from '../../stores/packCapabilities';
import { ChatAPI } from '../../api/chatAPI';
import { PackStoreBrowse } from '../pack-store/PackStoreBrowse';
import { PackDevStudio } from '../pack-store/dev/PackDevStudio';
import { mergeSettingsPut, type SettingsTabProps } from './settingsShared';

export function DomainPacksSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
    const [packsLoading, setPacksLoading] = useState(false);
    const packs = usePacksStore((s) => s.packs);
    const layoutOwner = usePacksStore((s) => s.layoutOwner);
    const setLayoutOwner = usePacksStore((s) => s.setLayoutOwner);
    const bioPackTools = usePacksStore((s) =>
      s.packs.some((p) => p.id === 'life-sciences' && p.enabled),
    );
    const bioSecondaryAnalysisTools = usePacksStore(
      (s) =>
        s.hasCapability(PACK_CAP.SECONDARY_ANALYSIS_API) ||
        s.hasCapability(PACK_CAP.SECONDARY_ANALYSIS_PYTHON),
    );
    const cadPackTools = usePacksStore((s) => s.hasCapability(PACK_CAP.CAD_API));
    const [packsErr, setPacksErr] = useState<string | null>(null);
    const [mcpEnabled, setMcpEnabled] = useState(true);
    const [mcpAgents, setMcpAgents] = useState<Record<string, boolean>>({});
    const [bioChatModel, setBioChatModel] = useState('koesn/llama3-openbiollm-8b:latest');
    const [bioToolModel, setBioToolModel] = useState('qwen2.5:7b');
    const [bioMaxFold, setBioMaxFold] = useState('400');
    const [bioMaxAnalyze, setBioMaxAnalyze] = useState('10000');
    const [bioEsmfoldModel, setBioEsmfoldModel] = useState('facebook/esmfold_v1');
    const [bioArtifactsDir, setBioArtifactsDir] = useState('');
    const [bioSecondaryToolsPath, setBioSecondaryToolsPath] = useState('');
    const [bioPythonExecutable, setBioPythonExecutable] = useState('python3');
    const [bioCumulativeQCDir, setBioCumulativeQCDir] = useState('');
    const [bioDefaultPanelProfile, setBioDefaultPanelProfile] = useState('human-inflammatory-12plex-v1');
    const [cadOpenSCADPath, setCadOpenSCADPath] = useState('openscad');
    const [cadFreeCADPath, setCadFreeCADPath] = useState('');
    const [cadArtifactsDir, setCadArtifactsDir] = useState('');
    const [cadRenderTimeout, setCadRenderTimeout] = useState('120');
    const [cadChatModel, setCadChatModel] = useState('qwen2.5-coder:14b');
    const [cadToolModel, setCadToolModel] = useState('qwen2.5:7b');
    const [cadSettingsSaving, setCadSettingsSaving] = useState(false);
    const [cadSettingsErr, setCadSettingsErr] = useState<string | null>(null);
    const [cadSettingsOk, setCadSettingsOk] = useState<string | null>(null);
    const [cadTestResult, setCadTestResult] = useState<string | null>(null);
    const [bioSettingsSaving, setBioSettingsSaving] = useState(false);
    const [bioSettingsErr, setBioSettingsErr] = useState<string | null>(null);
    const [bioSettingsOk, setBioSettingsOk] = useState<string | null>(null);
    const refreshDomainPacks = async () => {
      setPacksLoading(true);
      setPacksErr(null);
      try {
        const api = new ChatAPI(hubHttp);
        const data = await api.fetchPacks();
        usePacksStore.getState().applyPacksResponse(data);
      } catch (e) {
        setPacksErr(e instanceof Error ? e.message : String(e));
      } finally {
        setPacksLoading(false);
      }
    };
    useEffect(() => {
      if (!isActive) return;
      void refreshDomainPacks();
    }, [isActive, hubHttp]);
    useEffect(() => {
      if (!isActive) return;
      let cancelled = false;
      (async () => {
        try {
          const r = await fetch(`${hubHttp}/api/settings`);
          if (!r.ok) {
            throw new Error(await r.text());
          }
          const cfg = await r.json();
          if (!cancelled) {
            setMcpEnabled(cfg.mcp?.enabled !== false);
            setMcpAgents(
              cfg.mcp?.agents && typeof cfg.mcp.agents === 'object'
                ? (cfg.mcp.agents as Record<string, boolean>)
                : {}
            );
            const bio = cfg.mcp?.biology ?? {};
            setBioChatModel(bio.chat_model || 'koesn/llama3-openbiollm-8b:latest');
            setBioToolModel(bio.tool_model || 'qwen2.5:7b');
            setBioMaxFold(String(bio.max_fold_length || 400));
            setBioMaxAnalyze(String(bio.max_analyze_length || 10000));
            setBioEsmfoldModel(bio.esmfold_model || 'facebook/esmfold_v1');
            setBioArtifactsDir(bio.artifacts_dir || '');
            setBioSecondaryToolsPath(bio.secondary_analysis_tools_path || '');
            setBioPythonExecutable(bio.python_executable || 'python3');
            setBioCumulativeQCDir(bio.cumulative_qc_dir || '');
            setBioDefaultPanelProfile(bio.default_panel_profile || 'human-inflammatory-12plex-v1');
            const cadCfg = cfg.mcp?.cad ?? {};
            setCadOpenSCADPath(cadCfg.openscad_path || 'openscad');
            setCadFreeCADPath(cadCfg.freecad_path || '');
            setCadArtifactsDir(cadCfg.artifacts_dir || '');
            setCadRenderTimeout(String(cadCfg.render_timeout_sec || 120));
            setCadChatModel(cadCfg.chat_model || 'qwen2.5-coder:14b');
            setCadToolModel(cadCfg.tool_model || 'qwen2.5:7b');
          }
        } catch (e) {
          if (!cancelled) {
            setPacksErr(e instanceof Error ? e.message : String(e));
          }
        }
      })();
      return () => {
        cancelled = true;
      };
    }, [isActive, hubHttp]);
    const saveBioMcpSettings = async () => {
      setBioSettingsSaving(true);
      setBioSettingsErr(null);
      setBioSettingsOk(null);
      try {
        const maxFold = parseInt(bioMaxFold, 10);
        const maxAnalyze = parseInt(bioMaxAnalyze, 10);
        if (!Number.isFinite(maxFold) || maxFold <= 0 || !Number.isFinite(maxAnalyze) || maxAnalyze <= 0) {
          throw new Error('Max lengths must be positive integers');
        }
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          mcp: {
            ...(cfg.mcp as object | undefined),
            enabled: mcpEnabled,
            biology: {
              chat_model: bioChatModel.trim() || 'koesn/llama3-openbiollm-8b:latest',
              tool_model: bioToolModel.trim() || 'qwen2.5:7b',
              esmfold_model: bioEsmfoldModel.trim() || 'facebook/esmfold_v1',
              max_fold_length: maxFold,
              max_analyze_length: maxAnalyze,
              artifacts_dir: bioArtifactsDir.trim(),
              secondary_analysis_tools_path: bioSecondaryToolsPath.trim(),
              python_executable: bioPythonExecutable.trim() || 'python3',
              cumulative_qc_dir: bioCumulativeQCDir.trim(),
              default_panel_profile: bioDefaultPanelProfile.trim() || 'human-inflammatory-12plex-v1',
            },
          },
        }));
        setBioSettingsOk('Life sciences settings saved. Restart BiologyExpert if it is already running.');
      } catch (e) {
        setBioSettingsErr(e instanceof Error ? e.message : String(e));
      } finally {
        setBioSettingsSaving(false);
      }
    };

    const saveCadMcpSettings = async () => {
      setCadSettingsSaving(true);
      setCadSettingsErr(null);
      setCadSettingsOk(null);
      try {
        const timeout = parseInt(cadRenderTimeout, 10);
        if (!Number.isFinite(timeout) || timeout <= 0) {
          throw new Error('Render timeout must be a positive integer');
        }
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          mcp: {
            ...(cfg.mcp as object | undefined),
            enabled: mcpEnabled,
            cad: {
              openscad_path: cadOpenSCADPath.trim() || 'openscad',
              freecad_path: cadFreeCADPath.trim(),
              artifacts_dir: cadArtifactsDir.trim(),
              render_timeout_sec: timeout,
              chat_model: cadChatModel.trim() || 'qwen2.5-coder:14b',
              tool_model: cadToolModel.trim() || 'qwen2.5:7b',
            },
          },
        }));
        setCadSettingsOk('CAD tool settings saved. Restart CADExpert if it is already running.');
      } catch (e) {
        setCadSettingsErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCadSettingsSaving(false);
      }
    };

    const testCadOpenSCAD = async () => {
      setCadTestResult(null);
      try {
        const api = new ChatAPI(hubHttp);
        const res = await api.testOpenSCAD(cadOpenSCADPath.trim() || undefined);
        setCadTestResult(res.ok ? res.message : `Failed: ${res.message}`);
      } catch (e) {
        setCadTestResult(e instanceof Error ? e.message : String(e));
      }
    };

    const handleMcpMasterToggle = async (enabled: boolean) => {
      setMcpEnabled(enabled);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          mcp: { ...(cfg.mcp as object | undefined), enabled },
        }));
      } catch (e) {
        setMcpEnabled(!enabled);
        setBioSettingsErr(e instanceof Error ? e.message : String(e));
      }
    };

    const handleMcpAgentToggle = async (agentKey: string, enabled: boolean) => {
      setMcpAgents((prev) => ({ ...prev, [agentKey]: enabled }));
      try {
        await mergeSettingsPut(hubHttp, (cfg) => {
          const mcp = (cfg.mcp ?? {}) as Record<string, unknown>;
          const prevAgents =
            mcp.agents && typeof mcp.agents === 'object'
              ? (mcp.agents as Record<string, boolean>)
              : {};
          return {
            ...cfg,
            mcp: {
              ...mcp,
              enabled: mcp.enabled !== false,
              agents: { ...prevAgents, [agentKey]: enabled },
            },
          };
        });
      } catch (e) {
        setMcpAgents((prev) => ({ ...prev, [agentKey]: !enabled }));
        setBioSettingsErr(e instanceof Error ? e.message : String(e));
      }
    };
  if (!isActive) return null;

  return (
  <div className="space-y-8">
    <div className="border border-slack-border rounded-lg p-4 bg-slack-bgHover/30 text-sm text-slack-textMuted">
      <strong className="text-slack-text">Always on:</strong> ChatModerator, Assistant, and CLI agents (Cursor, Claude, Copilot, Codex, Gemini, Aider, OpenCode, and more) when installed on your PATH. Domain packs add optional in-process specialists and tools below.
    </div>

    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Pack store</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Install official domain packs from the catalog. You can enable multiple packs; choose which enabled pack controls the UI layout (IDE vs team). Later packs add specialists and tools without changing layout unless you switch the owner below.
      </p>
      {(() => {
        const layoutCandidates = packs.filter((p) => p.enabled && !p.custom);
        if (layoutCandidates.length < 2) return null;
        return (
          <div className="mb-4">
            <label className="block text-sm font-medium text-slack-text mb-1" htmlFor="layout-owner-select">
              UI layout owner
            </label>
            <select
              id="layout-owner-select"
              className="w-full max-w-md rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
              value={layoutOwner}
              onChange={(e) => void setLayoutOwner(e.target.value)}
            >
              {layoutCandidates.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.title} ({p.layout_profile === 'ide' ? 'IDE' : 'Team'})
                </option>
              ))}
            </select>
            <p className="text-xs text-slack-textMuted mt-1">
              Controls IDE vs team layout and which pack&apos;s default Ollama model applies when enabled.
            </p>
          </div>
        );
      })()}
      {packsLoading && <p className="text-sm text-slack-textMuted mb-2">Loading packs…</p>}
      {packsErr && <p className="text-sm text-red-600 mb-2">{packsErr}</p>}
      <PackStoreBrowse />
    </div>

    <PackDevStudio />

    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">MCP specialist tools</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Per-agent MCP tool servers. Enablement follows domain packs by default; override individual specialists here. Repo and Confluence agents always use in-process search tools when indexed.
      </p>
      <label className="flex items-center gap-3 cursor-pointer mb-4">
        <input
          type="checkbox"
          checked={mcpEnabled}
          onChange={(e) => void handleMcpMasterToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-sm text-slack-text">Enable MCP tool servers (master)</span>
      </label>
      {mcpEnabled && (
        <div className="grid gap-2 sm:grid-cols-2">
          {[
            ['backend', 'BackendEngineer'],
            ['frontend', 'FrontendEngineer'],
            ['devops', 'PlatformEngineer'],
            ['database', 'DatabaseSpecialist'],
            ['security', 'SecurityReviewer'],
            ['code-review', 'CodeReviewer'],
            ['architecture', 'SoftwareArchitect'],
            ['biology', 'BiologyExpert'],
            ['cad', 'CADExpert'],
            ['rust', 'RustExpert'],
          ].map(([key, label]) => (
            <label key={key} className="flex items-center gap-2 cursor-pointer text-sm">
              <input
                type="checkbox"
                checked={mcpAgents[key] !== false}
                onChange={(e) => void handleMcpAgentToggle(key, e.target.checked)}
                className="rounded border-slack-border"
              />
              <span className="text-slack-text">{label}</span>
            </label>
          ))}
        </div>
      )}
    </div>

    {bioPackTools && (
      <div className="border border-slack-border rounded-lg p-6">
        <h3 className="text-lg font-semibold text-slack-text mb-2">Life sciences tools</h3>
        <p className="text-sm text-slack-textMuted mb-4">
          Model layering for BiologyExpert: OpenBio (or your chat tag) for reasoning; a tool-capable model
          for MCP (<code className="font-mono text-xs bg-slack-bgHover px-1 rounded">analyze_sequence</code>,{' '}
          <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">fold_protein</code>, QC tools).
        </p>
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="block text-sm">
            <span className="text-slack-textMuted">Chat model (domain reasoning)</span>
            <input
              type="text"
              value={bioChatModel}
              onChange={(e) => setBioChatModel(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
          <label className="block text-sm">
            <span className="text-slack-textMuted">Tool model (MCP loop)</span>
            <input
              type="text"
              value={bioToolModel}
              onChange={(e) => setBioToolModel(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
          <label className="block text-sm">
            <span className="text-slack-textMuted">Max fold length (aa)</span>
            <input
              type="number"
              value={bioMaxFold}
              onChange={(e) => setBioMaxFold(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text"
            />
          </label>
          <label className="block text-sm">
            <span className="text-slack-textMuted">Max analyze length</span>
            <input
              type="number"
              value={bioMaxAnalyze}
              onChange={(e) => setBioMaxAnalyze(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text"
            />
          </label>
          <label className="block text-sm sm:col-span-2">
            <span className="text-slack-textMuted">ESMFold model (Hub id)</span>
            <input
              type="text"
              value={bioEsmfoldModel}
              onChange={(e) => setBioEsmfoldModel(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
          <label className="block text-sm sm:col-span-2">
            <span className="text-slack-textMuted">Artifacts directory (empty = ~/.neural-junkie/bio)</span>
            <input
              type="text"
              value={bioArtifactsDir}
              onChange={(e) => setBioArtifactsDir(e.target.value)}
              placeholder="~/.neural-junkie/bio"
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
          {bioSecondaryAnalysisTools && (
            <>
              <p className="text-xs text-slack-textMuted sm:col-span-2">
                Customer pack (<code className="font-mono">settings_overlay</code>). Override below
                if needed.
              </p>
              <label className="block text-sm sm:col-span-2">
                <span className="text-slack-textMuted">Secondary analysis tools path</span>
                <input
                  type="text"
                  value={bioSecondaryToolsPath}
                  onChange={(e) => setBioSecondaryToolsPath(e.target.value)}
                  placeholder="/path/to/secondary-analysis-tools"
                  className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                />
              </label>
              <label className="block text-sm">
                <span className="text-slack-textMuted">Python executable</span>
                <input
                  type="text"
                  value={bioPythonExecutable}
                  onChange={(e) => setBioPythonExecutable(e.target.value)}
                  className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                />
              </label>
              <label className="block text-sm">
                <span className="text-slack-textMuted">Default panel profile</span>
                <input
                  type="text"
                  value={bioDefaultPanelProfile}
                  onChange={(e) => setBioDefaultPanelProfile(e.target.value)}
                  className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                />
              </label>
              <label className="block text-sm sm:col-span-2">
                <span className="text-slack-textMuted">
                  Cumulative QC folder override (empty = workspace/.neural-junkie/cumulative-qc)
                </span>
                <input
                  type="text"
                  value={bioCumulativeQCDir}
                  onChange={(e) => setBioCumulativeQCDir(e.target.value)}
                  className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                />
              </label>
            </>
          )}
        </div>
        <button
          type="button"
          onClick={() => void saveBioMcpSettings()}
          disabled={bioSettingsSaving}
          className="mt-4 px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
        >
          {bioSettingsSaving ? 'Saving…' : 'Save life sciences tools'}
        </button>
        {bioSettingsErr && <p className="text-sm text-red-600 mt-2">{bioSettingsErr}</p>}
        {bioSettingsOk && <p className="text-sm text-green-600 mt-2">{bioSettingsOk}</p>}
      </div>
    )}

    {cadPackTools && (
      <div className="border border-slack-border rounded-lg p-6">
        <h3 className="text-lg font-semibold text-slack-text mb-2">CAD tools</h3>
        <p className="text-sm text-slack-textMuted mb-4">
          OpenSCAD rendering for <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">render_openscad</code> and the CAD workbench. Install OpenSCAD from{' '}
          <a href="https://openscad.org" className="text-slack-accent hover:underline" target="_blank" rel="noreferrer">openscad.org</a>.
        </p>
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="block text-sm sm:col-span-2">
            <span className="text-slack-textMuted">OpenSCAD path</span>
            <input
              type="text"
              value={cadOpenSCADPath}
              onChange={(e) => setCadOpenSCADPath(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
          <label className="block text-sm">
            <span className="text-slack-textMuted">Chat model</span>
            <input
              type="text"
              value={cadChatModel}
              onChange={(e) => setCadChatModel(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
          <label className="block text-sm">
            <span className="text-slack-textMuted">Tool model</span>
            <input
              type="text"
              value={cadToolModel}
              onChange={(e) => setCadToolModel(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
          <label className="block text-sm">
            <span className="text-slack-textMuted">Render timeout (sec)</span>
            <input
              type="number"
              value={cadRenderTimeout}
              onChange={(e) => setCadRenderTimeout(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text"
            />
          </label>
          <label className="block text-sm sm:col-span-2">
            <span className="text-slack-textMuted">Artifacts directory (empty = ~/.neural-junkie/cad)</span>
            <input
              type="text"
              value={cadArtifactsDir}
              onChange={(e) => setCadArtifactsDir(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
          <label className="block text-sm sm:col-span-2">
            <span className="text-slack-textMuted">FreeCAD path (optional, for STEP export)</span>
            <input
              type="text"
              value={cadFreeCADPath}
              onChange={(e) => setCadFreeCADPath(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
            />
          </label>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => void saveCadMcpSettings()}
            disabled={cadSettingsSaving}
            className="px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
          >
            {cadSettingsSaving ? 'Saving…' : 'Save CAD tools'}
          </button>
          <button
            type="button"
            onClick={() => void testCadOpenSCAD()}
            className="px-4 py-2 text-sm border border-slack-border rounded text-slack-text hover:bg-slack-bgHover"
          >
            Test OpenSCAD
          </button>
        </div>
        {cadSettingsErr && <p className="text-sm text-red-600 mt-2">{cadSettingsErr}</p>}
        {cadSettingsOk && <p className="text-sm text-green-600 mt-2">{cadSettingsOk}</p>}
        {cadTestResult && <p className="text-sm text-slack-textMuted mt-2 font-mono whitespace-pre-wrap">{cadTestResult}</p>}
      </div>
    )}
  </div>
  );
}
