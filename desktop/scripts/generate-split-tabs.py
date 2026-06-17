#!/usr/bin/env python3
"""Generate split settings tab components from monolithic sources."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SETTINGS = ROOT / "src" / "components" / "settings"
AI = (SETTINGS / "AIProvidersSettingsTab.tsx").read_text()
INT = (SETTINGS / "IntegrationsSettingsTab.tsx").read_text()


def write(name: str, body: str) -> None:
    path = SETTINGS / name
    path.write_text(body)
    print(f"wrote {name}: {len(body.splitlines())} lines")


def between(src: str, start: str, end: str) -> str:
    a = src.index(start)
    b = src.index(end, a)
    return src[a:b].strip()


def slice_lines(src: str, start: int, end: int) -> str:
    lines = src.splitlines()
    return "\n".join(lines[start - 1 : end]).strip()


# --- Providers (already hand-authored; regenerate from sections) ---
ollama_block = slice_lines(AI, 1370, 1476)
lm_block = slice_lines(AI, 1478, 1576)
global_block = slice_lines(AI, 1578, 1635)
cli_block = slice_lines(AI, 1211, 1214)
prov_block = slice_lines(AI, 1216, 1219)

# Handlers from AI file lines 556-752
provider_handlers = slice_lines(AI, 556, 752)

write(
    "ProvidersSettingsTab.tsx",
    f"""import {{ useState, useEffect }} from 'react';
import {{ shallow }} from 'zustand/shallow';
import {{ useSettingsStore }} from '../../stores/settingsStore';
import {{ useChatStore }} from '../../stores/chatStore';
import {{ ProviderManager }} from '../ProviderManager';
import {{ CLIAgentsManager }} from '../CLIAgentsManager';
import {{
  fetchHardwareSnapshot,
  fetchModelLookup,
  formatModelResourceHint,
  type HardwareSnapshot,
  type ModelLookup,
}} from '../../utils/hardwareRecommendations';
import type {{ OllamaSettings, LMStudioSettings }} from '../../types/protocol';
import type {{ SettingsTabProps }} from './settingsShared';

export function ProvidersSettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
  const {{
    integrations,
    loadIntegrations,
    updateOllamaSettings,
    updateLMStudioSettings,
    fetchOllamaModels,
    fetchLMStudioModels,
    testOllamaConnection,
    testLMStudioConnection,
  }} = useSettingsStore();
  const {{ switchAllAgentProviders }} = useChatStore(
    (s) => ({{ switchAllAgentProviders: s.switchAllAgentProviders }}),
    shallow
  );

  const [ollamaForm, setOllamaForm] = useState<OllamaSettings>(integrations.ollama);
  const [hardwareSnapshot, setHardwareSnapshot] = useState<HardwareSnapshot | null>(null);
  const [defaultModelLookup, setDefaultModelLookup] = useState<ModelLookup | null>(null);
  const [lmstudioForm, setLMStudioForm] = useState<LMStudioSettings>(integrations.lmstudio);
  const [testResults, setTestResults] = useState<Record<string, {{ success: boolean; message: string }}>>({{}});
  const [isSwitching, setIsSwitching] = useState(false);

  useEffect(() => {{
    if (!isActive) return;
    loadIntegrations();
  }}, [isActive, loadIntegrations]);

  useEffect(() => {{
    setOllamaForm(integrations.ollama);
    setLMStudioForm(integrations.lmstudio);
  }}, [integrations]);

  useEffect(() => {{
    if (!isActive) return;
    let cancelled = false;
    const loadModels = async () => {{
      try {{
        const ollamaModels = await fetchOllamaModels();
        if (!cancelled) setOllamaForm((prev) => ({{ ...prev, availableModels: ollamaModels }}));
      }} catch {{ /* Ollama may not be running */ }}
      try {{
        const lmModels = await fetchLMStudioModels();
        if (!cancelled) setLMStudioForm((prev) => ({{ ...prev, availableModels: lmModels }}));
      }} catch {{ /* LM Studio may not be running */ }}
    }};
    void loadModels();
    return () => {{ cancelled = true; }};
  }}, [isActive, fetchOllamaModels, fetchLMStudioModels]);

  useEffect(() => {{
    if (!isActive) return;
    let cancelled = false;
    void fetchHardwareSnapshot(hubHttp).then((snap) => {{
      if (!cancelled) setHardwareSnapshot(snap);
    }});
    return () => {{ cancelled = true; }};
  }}, [isActive, hubHttp]);

  useEffect(() => {{
    if (!isActive) return;
    const model = ollamaForm.defaultModel?.trim();
    if (!model) {{
      setDefaultModelLookup(null);
      return;
    }}
    let cancelled = false;
    void fetchModelLookup(hubHttp, model).then((row) => {{
      if (!cancelled) setDefaultModelLookup(row);
    }});
    return () => {{ cancelled = true; }};
  }}, [isActive, hubHttp, ollamaForm.defaultModel]);

{provider_handlers}

  if (!isActive) return null;

  return (
    <div className="space-y-8">
{cli_block}
{prov_block}
{ollama_block}
{lm_block}
{global_block}
    </div>
  );
}}
""",
)

# Models & performance
layout_toggles = """    <div className="mb-4">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Agent &amp; editor behavior</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        Controls for memory monitoring, routing visibility, inline completion, and IDE file-change trust.
      </p>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Memory monitor</div>
        <div className="text-sm text-slack-textMuted">
          Show live system RAM and Ollama loaded-model usage in the toolbar
        </div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.memoryMonitorEnabled !== false}
          onChange={(e) => updateLayoutSettings({ memoryMonitorEnabled: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Routing badges on messages</div>
        <div className="text-sm text-slack-textMuted">
          Show which model ran on each agent reply (chat model, tool model, routing reason)
        </div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.showRoutingOnMessages !== false}
          onChange={(e) => updateLayoutSettings({ showRoutingOnMessages: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Inline completion (ghost text)</div>
        <div className="text-sm text-slack-textMuted">Ollama FIM via hub when Software development pack is on</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.inlineCompletionEnabled ?? false}
          onChange={(e) => updateLayoutSettings({ inlineCompletionEnabled: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Editor agent trust</div>
        <div className="text-sm text-slack-textMuted">
          How file changes from IDE-mode chat are applied (Ask/Agent toggle on the main composer)
        </div>
      </div>
      <select
        value={layoutSettings.editorAgentTrust ?? 'interactive'}
        onChange={(e) =>
          updateLayoutSettings({
            editorAgentTrust: e.target.value as 'interactive' | 'auto_apply_edits' | 'yolo',
          })
        }
        className="text-sm bg-slack-bg border border-slack-border rounded px-2 py-1"
      >
        <option value="interactive">Interactive (approve each)</option>
        <option value="auto_apply_edits">Auto-apply edits</option>
        <option value="yolo">Yolo (tools)</option>
      </select>
    </div>"""

hf_block = slice_lines(AI, 757, 790)
specialist_block = slice_lines(AI, 1146, 1209)
library_block = slice_lines(AI, 1221, 1229)
perf_block = slice_lines(AI, 1231, 1368)

models_perf_state_handlers = """
  const { layoutSettings, updateLayoutSettings, integrations, loadIntegrations, fetchOllamaModels } =
    useSettingsStore();
  const [ollamaModels, setOllamaModels] = useState<string[]>(integrations.ollama.availableModels ?? []);
  const [hfHubToken, setHfHubToken] = useState('');
  const [hfHubTokenPersisted, setHfHubTokenPersisted] = useState('');
  const [hfTokenSaving, setHfTokenSaving] = useState(false);
  const [hfTokenErr, setHfTokenErr] = useState<string | null>(null);
  const [hfTokenOk, setHfTokenOk] = useState<string | null>(null);
  const [configuredAgents, setConfiguredAgents] = useState<
    { type: string; name: string; enabled: boolean; model?: string }[]
  >([]);
  const [agentModelsSaving, setAgentModelsSaving] = useState(false);
  const [agentModelsErr, setAgentModelsErr] = useState<string | null>(null);
  const [agentModelsOk, setAgentModelsOk] = useState<string | null>(null);
  const [specialistModelsAdvancedOpen, setSpecialistModelsAdvancedOpen] = useState(false);
  const [perfForm, setPerfForm] = useState({
    contextBudgetKB: 32,
    ideContextBudgetKB: 48,
    implSessionBudgetKB: 64,
    maxHistoryMessages: 10,
    ollamaNumCtx: 0,
    ollamaNumPredict: 0,
    ollamaKeepAlive: '',
  });
  const [perfSaving, setPerfSaving] = useState(false);
  const [perfFeedback, setPerfFeedback] = useState<{ success: boolean; message: string } | null>(null);

  const { config, save, reload } = useHubSettingsSnapshot(hubHttp, isActive);

  useEffect(() => {
    if (!isActive) return;
    loadIntegrations();
  }, [isActive, loadIntegrations]);

  useEffect(() => {
    setOllamaModels(integrations.ollama.availableModels ?? []);
  }, [integrations.ollama.availableModels]);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    void fetchOllamaModels()
      .then((models) => {
        if (!cancelled) setOllamaModels(models);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [isActive, fetchOllamaModels]);

  useEffect(() => {
    if (!isActive || !config) return;
    const hfTok = typeof config.hf === 'object' && config.hf && typeof (config.hf as { token?: string }).token === 'string'
      ? (config.hf as { token: string }).token
      : '';
    const redacted = hfTok.includes('...') || hfTok === '***';
    setHfHubToken(redacted ? '' : hfTok);
    setHfHubTokenPersisted(redacted ? '***' : hfTok);
    const perf = config.performance as Record<string, number> | undefined;
    const ollamaCfg = config.ollama as Record<string, unknown> | undefined;
    setPerfForm({
      contextBudgetKB: perf?.context_budget_kb || 32,
      ideContextBudgetKB: perf?.ide_context_budget_kb || 48,
      implSessionBudgetKB: perf?.impl_session_budget_kb || 64,
      maxHistoryMessages: perf?.max_history_messages || 10,
      ollamaNumCtx: typeof ollamaCfg?.num_ctx === 'number' ? ollamaCfg.num_ctx : 0,
      ollamaNumPredict: typeof ollamaCfg?.num_predict === 'number' ? ollamaCfg.num_predict : 0,
      ollamaKeepAlive: typeof ollamaCfg?.keep_alive === 'string' ? ollamaCfg.keep_alive : '',
    });
  }, [isActive, config]);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/agents/configured`);
        if (!r.ok) throw new Error(await r.text());
        const rows = (await r.json()) as { type: string; name: string; enabled: boolean; model?: string }[];
        if (!cancelled) setConfiguredAgents(rows);
      } catch (e) {
        if (!cancelled) setAgentModelsErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp]);

  const saveHfHubToken = async () => {
    setHfTokenSaving(true);
    setHfTokenErr(null);
    setHfTokenOk(null);
    try {
      const trimmed = hfHubToken.trim();
      await save((cfg) => ({
        ...cfg,
        hf: { ...(cfg.hf as object | undefined), token: trimmed },
      }));
      setHfHubTokenPersisted(trimmed ? '***' : '');
      setHfTokenOk(trimmed ? 'Hugging Face token saved.' : 'Cleared hub Hugging Face token.');
    } catch (e) {
      setHfTokenErr(e instanceof Error ? e.message : String(e));
    } finally {
      setHfTokenSaving(false);
    }
  };

  const savePerformanceSettings = async () => {
    setPerfSaving(true);
    setPerfFeedback(null);
    try {
      await save((cfg) => ({
        ...cfg,
        performance: {
          context_budget_kb: perfForm.contextBudgetKB,
          ide_context_budget_kb: perfForm.ideContextBudgetKB,
          impl_session_budget_kb: perfForm.implSessionBudgetKB,
          max_history_messages: perfForm.maxHistoryMessages,
        },
        ollama: {
          ...(typeof cfg.ollama === 'object' && cfg.ollama ? cfg.ollama : {}),
          num_ctx: perfForm.ollamaNumCtx > 0 ? perfForm.ollamaNumCtx : 0,
          num_predict: perfForm.ollamaNumPredict > 0 ? perfForm.ollamaNumPredict : 0,
          keep_alive: perfForm.ollamaKeepAlive.trim(),
        },
      }));
      setPerfFeedback({ success: true, message: 'Performance settings saved.' });
    } catch (e) {
      setPerfFeedback({
        success: false,
        message: e instanceof Error ? e.message : 'Failed to save performance settings',
      });
    } finally {
      setPerfSaving(false);
    }
  };

  const saveConfiguredAgentModels = async () => {
    setAgentModelsSaving(true);
    setAgentModelsErr(null);
    setAgentModelsOk(null);
    try {
      await save((cfg) => {
        const existing = Array.isArray(cfg.agents) ? cfg.agents : [];
        const byKey = new Map(configuredAgents.map((a) => [`${a.type}\\x00${a.name}`, a]));
        const agents = existing.map((row: { type: string; name: string; model?: string }) => {
          const hit = byKey.get(`${row.type}\\x00${row.name}`);
          if (!hit) return row;
          return { ...row, model: hit.model?.trim() || undefined };
        });
        return { ...cfg, agents };
      });
      await fetch(`${hubHttp}/api/agents/restart`, { method: 'POST' });
      setAgentModelsOk('Specialist models saved. Agents restarted.');
      await reload();
    } catch (e) {
      setAgentModelsErr(e instanceof Error ? e.message : String(e));
    } finally {
      setAgentModelsSaving(false);
    }
  };
"""

# Fix specialist block to use ollamaModels instead of ollamaForm
specialist_block = specialist_block.replace("ollamaForm.availableModels", "ollamaModels")
specialist_block = specialist_block.replace(
    "placeholder={ollamaForm.defaultModel || 'qwen2.5-coder:14b'}",
    "placeholder={integrations.ollama.defaultModel || 'qwen2.5-coder:14b'}",
)

write(
    "ModelsPerformanceSettingsTab.tsx",
    f"""import {{ useState, useEffect }} from 'react';
import {{ useSettingsStore }} from '../../stores/settingsStore';
import {{ mergeSettingsPut, type SettingsTabProps }} from './settingsShared';
import {{ useHubSettingsSnapshot }} from './useHubSettingsSnapshot';

export function ModelsPerformanceSettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
{models_perf_state_handlers}
  if (!isActive) return null;

  return (
    <div className="space-y-8">
{layout_toggles}
{hf_block}
{specialist_block}
{library_block}
{perf_block}
    </div>
  );
}}
""".replace("mergeSettingsPut", "mergeSettingsPut").replace(
    "import { mergeSettingsPut, type SettingsTabProps }", "import { type SettingsTabProps }"
),
)

# Collab routing tab - extract handlers and JSX
collab_jsx = "\n".join(
    [
        slice_lines(AI, 792, 844),
        slice_lines(AI, 846, 862),
        slice_lines(AI, 864, 894),
        slice_lines(AI, 896, 925),
        slice_lines(AI, 927, 966),
    ]
)

collab_handlers = slice_lines(AI, 302, 554)

write(
    "CollabRoutingSettingsTab.tsx",
    f"""import {{ useState, useEffect }} from 'react';
import {{ open }} from '@tauri-apps/api/dialog';
import type {{ SettingsTabProps }} from './settingsShared';

export function CollabRoutingSettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
  const [collabSmartRouting, setCollabSmartRouting] = useState(false);
  const [collabPlanningProviderId, setCollabPlanningProviderId] = useState('');
  const [configuredProviders, setConfiguredProviders] = useState<Array<{{ id: string; name: string }}>>([]);
  const [implRoutingEnabled, setImplRoutingEnabled] = useState(true);
  const [implRoutingEnabledPersisted, setImplRoutingEnabledPersisted] = useState(true);
  const [implLocalToolModel, setImplLocalToolModel] = useState('qwen2.5-coder:7b');
  const [implLocalToolModelPersisted, setImplLocalToolModelPersisted] = useState('qwen2.5-coder:7b');
  const [collabAutoApproveDeliverables, setCollabAutoApproveDeliverables] = useState(true);
  const [collabRoutingSaving, setCollabRoutingSaving] = useState(false);
  const [collabRoutingErr, setCollabRoutingErr] = useState<string | null>(null);
  const [delegationEnabled, setDelegationEnabled] = useState(false);
  const [delegationSaving, setDelegationSaving] = useState(false);
  const [collabAssetsRoot, setCollabAssetsRoot] = useState('');
  const [collabAssetsPersisted, setCollabAssetsPersisted] = useState('');
  const [collabAssetsSaving, setCollabAssetsSaving] = useState(false);
  const [collabAssetsErr, setCollabAssetsErr] = useState<string | null>(null);
  const [collabAssetsOk, setCollabAssetsOk] = useState<string | null>(null);

  useEffect(() => {{
    if (!isActive) return;
    let cancelled = false;
    setCollabRoutingErr(null);
    (async () => {{
      try {{
        const r = await fetch(`${{hubHttp}}/api/settings`);
        if (!r.ok) throw new Error(await r.text());
        const cfg = await r.json();
        if (!cancelled) {{
          setCollabSmartRouting(!!cfg.collaboration?.smart_routing_enabled);
          setCollabPlanningProviderId(
            typeof cfg.collaboration?.planning_provider_id === 'string'
              ? cfg.collaboration.planning_provider_id
              : ''
          );
          const provRows = Array.isArray(cfg.ai?.providers) ? cfg.ai.providers : [];
          setConfiguredProviders(
            provRows
              .map((p: {{ id?: string; name?: string }}) => ({{
                id: String(p.id ?? ''),
                name: String(p.name ?? p.id ?? ''),
              }}))
              .filter((p: {{ id: string }}) => p.id)
          );
          setImplRoutingEnabled(cfg.implementation?.routing_enabled !== false);
          setImplRoutingEnabledPersisted(cfg.implementation?.routing_enabled !== false);
          const toolModel =
            typeof cfg.implementation?.local_tool_model === 'string' &&
            cfg.implementation.local_tool_model.trim()
              ? cfg.implementation.local_tool_model.trim()
              : 'qwen2.5-coder:7b';
          setImplLocalToolModel(toolModel);
          setImplLocalToolModelPersisted(toolModel);
          setCollabAutoApproveDeliverables(cfg.collaboration?.auto_approve_deliverables !== false);
          setDelegationEnabled(!!cfg.delegation?.enabled);
          const root =
            typeof cfg.collaboration?.assets_root === 'string' ? cfg.collaboration.assets_root : '';
          setCollabAssetsRoot(root);
          setCollabAssetsPersisted(root);
          setCollabAssetsOk(null);
        }}
      }} catch (e) {{
        if (!cancelled) setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      }}
    }})();
    return () => {{
      cancelled = true;
    }};
  }}, [isActive, hubHttp]);

{collab_handlers}

  if (!isActive) return null;

  return (
    <div className="space-y-8">
{collab_jsx}
    </div>
  );
}}
""",
)

# Memory learning
memory_jsx = slice_lines(AI, 968, 984)
personal_block = slice_lines(AI, 986, 1144)
memory_handlers = slice_lines(AI, 230, 300)
memory_effects = slice_lines(AI, 176, 181)

write(
    "MemoryLearningSettingsTab.tsx",
    f"""import {{ useState, useEffect }} from 'react';
import {{ usePacksStore }} from '../../stores/packsStore';
import {{ PACK_CAP }} from '../../stores/packCapabilities';
import {{ ChatAPI, type UserLearning }} from '../../api/chatAPI';
import {{ mergeSettingsPut, type SettingsTabProps }} from './settingsShared';

export function MemoryLearningSettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
  const hasPersonalLearning = usePacksStore((s) => s.hasCapability(PACK_CAP.PERSONAL_LEARNING));
  const hasLoRATraining = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_TRAINING));
  const [personalLearningEnabled, setPersonalLearningEnabled] = useState(false);
  const [personalLearningSuggestEnabled, setPersonalLearningSuggestEnabled] = useState(false);
  const [conversationMemoryEnabled, setConversationMemoryEnabled] = useState(true);
  const [conversationMemorySaving, setConversationMemorySaving] = useState(false);
  const [personalLearningSaving, setPersonalLearningSaving] = useState(false);
  const [personalLearningsOpen, setPersonalLearningsOpen] = useState(false);
  const [allLearnings, setAllLearnings] = useState<UserLearning[]>([]);
  const [allLearningsLoading, setAllLearningsLoading] = useState(false);
  const [allLearningsErr, setAllLearningsErr] = useState<string | null>(null);
  const [collabRoutingErr, setCollabRoutingErr] = useState<string | null>(null);

  useEffect(() => {{
    if (!isActive) return;
    let cancelled = false;
    (async () => {{
      try {{
        const r = await fetch(`${{hubHttp}}/api/settings`);
        if (!r.ok) throw new Error(await r.text());
        const cfg = await r.json();
        if (!cancelled) {{
          setPersonalLearningEnabled(!!cfg.features?.personal_learning_enabled);
          setPersonalLearningSuggestEnabled(!!cfg.features?.personal_learning_suggest_enabled);
          setConversationMemoryEnabled(cfg.features?.conversation_memory_enabled !== false);
        }}
      }} catch (e) {{
        if (!cancelled) setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      }}
    }})();
    return () => {{ cancelled = true; }};
  }}, [isActive, hubHttp]);

{memory_effects}

{memory_handlers}

  if (!isActive) return null;

  return (
    <div className="space-y-8">
{memory_jsx}
      {{collabRoutingErr && <p className="text-sm text-red-600">{{collabRoutingErr}}</p>}}
{personal_block}
    </div>
  );
}}
""",
)

# Integrations splits - use regex to extract big sections from return JSX
int_return = between(INT, "  return (\n", "\n  );\n}")

api_jsx = "\n".join(
    [
        slice_lines(INT, 856, 964),
        slice_lines(INT, 966, 1033),
        slice_lines(INT, 2030, 2132),
    ]
)

assistant_jsx = "\n".join([slice_lines(INT, 1035, 1176), slice_lines(INT, 1178, 1349)])

slack_jsx = slice_lines(INT, 1351, 2028)

# Extract integration handlers (lines 730-834) + credential-specific from 103-140
cred_handlers = slice_lines(INT, 730, 834)
int_state_top = slice_lines(INT, 52, 60)
int_effects_cred = """    useEffect(() => {
      if (!isActive) return;
      loadIntegrations();
    }, [isActive, loadIntegrations]);

    useEffect(() => {
      setAnthropicForm(integrations.anthropic);
      setGitHubForm(integrations.github);
      setConfluenceForm(integrations.confluence);
    }, [integrations]);"""

assistant_handlers = slice_lines(INT, 103, 729)
assistant_state = slice_lines(INT, 56, 102)

slack_handlers = slice_lines(INT, 221, 621)
slack_state = slice_lines(INT, 64, 92)

write(
    "ApiCredentialsSettingsTab.tsx",
    f"""import {{ useState, useEffect }} from 'react';
import {{ useSettingsStore }} from '../../stores/settingsStore';
import type {{
  AnthropicSettings,
  GitHubSettings,
  ConfluenceSettings,
}} from '../../types/protocol';
import {{ openExternalLink, type SettingsTabProps }} from './settingsShared';

export function ApiCredentialsSettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
  const {{
    integrations,
    loadIntegrations,
    updateAnthropicSettings,
    updateGitHubSettings,
    updateConfluenceSettings,
    clearIntegrationSettings,
    testAnthropicConnection,
    testGitHubConnection,
    testConfluenceConnection,
  }} = useSettingsStore();

{int_state_top.replace('googleOAuthForm', '/* removed */').split(chr(10))[0]}
    const [anthropicForm, setAnthropicForm] = useState<AnthropicSettings>(integrations.anthropic);
    const [githubForm, setGitHubForm] = useState<GitHubSettings>(integrations.github);
    const [confluenceForm, setConfluenceForm] = useState<ConfluenceSettings>(integrations.confluence);
    const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({{}});
    const [testResults, setTestResults] = useState<Record<string, {{ success: boolean; message: string }}>>({{}});

{int_effects_cred}

{cred_handlers}

  if (!isActive) return null;

  return (
    <div className="space-y-8 nj-settings-integrations text-slack-text">
{api_jsx}
    </div>
  );
}}
""",
)

write(
    "AssistantToolsSettingsTab.tsx",
    f"""import {{ useState, useEffect }} from 'react';
import {{ useSettingsStore }} from '../../stores/settingsStore';
import {{ ChatAPI }} from '../../api/chatAPI';
import type {{
  GoogleMeetNotesSettings,
  GoogleMeetNotesStatus,
  WebSearchConfigResponse,
}} from '../../types/protocol';
import {{ openExternalLink, type SettingsTabProps }} from './settingsShared';

export function AssistantToolsSettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
  const {{ integrations, updateGoogleMeetNotesSettings }} = useSettingsStore();
{assistant_state}
{assistant_handlers}

  if (!isActive) return null;

  return (
    <div className="space-y-8 nj-settings-integrations text-slack-text">
{assistant_jsx}
    </div>
  );
}}
""",
)

write(
    "SlackSettingsTab.tsx",
    f"""import {{ useState, useEffect }} from 'react';
import {{ shallow }} from 'zustand/shallow';
import {{ useChatStore }} from '../../stores/chatStore';
import {{ ChatAPI }} from '../../api/chatAPI';
import type {{
  SlackStatus,
  SlackBinding,
  SlackChannelInfo,
  SlackPolicy,
  SlackConfigResponse,
  SlackConnectionResponse,
  SlackInboxConfig,
}} from '../../types/protocol';
import {{
  defaultSlackInboxForm,
  mergeSlackInboxForm,
  slackCanListChannelsFrom,
  updateForwardRule,
}} from './slackInboxHelpers';
import {{ openExternalLink, type SettingsTabProps }} from './settingsShared';

export function SlackSettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
  const {{ agents, setChannels }} = useChatStore(
    (s) => ({{ agents: s.agents, setChannels: s.setChannels }}),
    shallow
  );
{slack_state}
{slack_handlers}

    const humanDMStatusLabel = (status?: string) => {{
      switch (status) {{
        case 'monitoring_active':
          return 'Monitoring active';
        case 'not_authorized':
          return 'Not authorized — click Authorize Slack DM access';
        case 'inside_work_hours':
          return 'Inside work hours (schedule)';
        case 'away_off':
          return 'Away mode off';
        case 'inbox_not_ready':
          return 'Enable personal inbox and pick an agent first';
        case 'disabled':
          return 'Disabled';
        default:
          return status ?? '';
      }}
    }};

    const toggleMentionWatchChannel = (channelId: string) => {{
      setSlackInbox((prev) => {{
        const rules = prev.forward_rules ?? [];
        const mention = rules.find((r) => r.id === 'mentions' || r.type === 'mention_of_me');
        const ids = new Set(mention?.slack_channel_ids ?? []);
        if (ids.has(channelId)) ids.delete(channelId);
        else ids.add(channelId);
        return updateForwardRule(prev, mention?.id ?? 'mentions', {{
          slack_channel_ids: Array.from(ids),
        }});
      }});
    }};

    useEffect(() => {{
      if (!isActive) return;
      void refreshSlackIntegration();
    }}, [isActive, hubHttp]);

  if (!isActive) return null;

  return (
    <div className="space-y-8 nj-settings-integrations text-slack-text">
{slack_jsx}
    </div>
  );
}}
""",
)

print("done")
