import { useState, useEffect } from 'react';
import { open } from '@tauri-apps/api/dialog';
import type { SettingsTabProps } from './settingsShared';

export function CollabRoutingSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const [collabSmartRouting, setCollabSmartRouting] = useState(false);
  const [modelCapabilityRouting, setModelCapabilityRouting] = useState(true);
  const [capabilityProfileMeta, setCapabilityProfileMeta] = useState<{
    updated_at?: string;
    source_run_id?: string;
    source_suite?: string;
  } | null>(null);
  const [collabPlanningProviderId, setCollabPlanningProviderId] = useState('');
  const [configuredProviders, setConfiguredProviders] = useState<Array<{ id: string; name: string }>>([]);
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

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    setCollabRoutingErr(null);
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) throw new Error(await r.text());
        const cfg = await r.json();
        if (!cancelled) {
          setCollabSmartRouting(!!cfg.collaboration?.smart_routing_enabled);
          setModelCapabilityRouting(cfg.routing?.model_capability_routing_enabled !== false);
          if (cfg.capability_profiles && typeof cfg.capability_profiles === 'object') {
            setCapabilityProfileMeta(cfg.capability_profiles as {
              updated_at?: string;
              source_run_id?: string;
              source_suite?: string;
            });
          } else {
            setCapabilityProfileMeta(null);
          }
          setCollabPlanningProviderId(
            typeof cfg.collaboration?.planning_provider_id === 'string'
              ? cfg.collaboration.planning_provider_id
              : ''
          );
          const provRows = Array.isArray(cfg.ai?.providers) ? cfg.ai.providers : [];
          setConfiguredProviders(
            provRows
              .map((p: { id?: string; name?: string }) => ({
                id: String(p.id ?? ''),
                name: String(p.name ?? p.id ?? ''),
              }))
              .filter((p: { id: string }) => p.id)
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
        }
      } catch (e) {
        if (!cancelled) setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp]);

const handleDelegationToggle = async (enabled: boolean) => {
      setDelegationSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          delegation: {
            ...(cfg.delegation ?? {}),
            enabled,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setDelegationEnabled(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setDelegationSaving(false);
      }
    };

    const saveImplementationSettings = async () => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          implementation: {
            ...(cfg.implementation ?? {}),
            routing_enabled: implRoutingEnabled,
            local_tool_model: implLocalToolModel.trim() || 'qwen2.5-coder:7b',
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setImplLocalToolModelPersisted(implLocalToolModel.trim() || 'qwen2.5-coder:7b');
        setImplRoutingEnabledPersisted(implRoutingEnabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

    const handleCollabPlanningProviderChange = async (providerId: string) => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          collaboration: {
            ...(cfg.collaboration ?? {}),
            planning_provider_id: providerId.trim(),
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setCollabPlanningProviderId(providerId.trim());
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

    const handleModelCapabilityRoutingToggle = async (enabled: boolean) => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          routing: {
            ...(cfg.routing ?? {}),
            model_capability_routing_enabled: enabled,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setModelCapabilityRouting(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

    const handleCollabSmartRoutingToggle = async (enabled: boolean) => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          collaboration: {
            ...(cfg.collaboration ?? {}),
            smart_routing_enabled: enabled,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setCollabSmartRouting(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

    const handleCollabAutoApproveToggle = async (enabled: boolean) => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          collaboration: {
            ...(cfg.collaboration ?? {}),
            auto_approve_deliverables: enabled,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setCollabAutoApproveDeliverables(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

      const persistCollabAssetsRoot = async (path: string): Promise<boolean> => {
      setCollabAssetsSaving(true);
      setCollabAssetsErr(null);
      setCollabAssetsOk(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const trimmed = path.trim();
        const next = {
          ...cfg,
          collaboration: {
            ...(cfg.collaboration ?? {}),
            assets_root: trimmed,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setCollabAssetsRoot(trimmed);
        setCollabAssetsPersisted(trimmed);
        setCollabAssetsOk(
          trimmed
            ? 'Saved to hub. New collaborations will use this folder.'
            : 'Saved. New collaborations will use the default ~/.neural-junkie/collaborations.'
        );
        return true;
      } catch (e) {
        setCollabAssetsErr(e instanceof Error ? e.message : String(e));
        return false;
      } finally {
        setCollabAssetsSaving(false);
      }
    };

    const handleCollabAssetsRootSave = async () => {
      await persistCollabAssetsRoot(collabAssetsRoot);
    };

    const handleCollabAssetsRootBlur = () => {
      if (collabAssetsSaving) return;
      if (collabAssetsRoot.trim() === collabAssetsPersisted.trim()) return;
      void persistCollabAssetsRoot(collabAssetsRoot);
    };

    const handleBrowseCollabAssetsRoot = async () => {
      setCollabAssetsErr(null);
      setCollabAssetsOk(null);
      if (!(typeof window !== 'undefined' && (window as { __TAURI__?: unknown }).__TAURI__)) {
        setCollabAssetsErr('Folder picker requires the desktop app');
        return;
      }
      try {
        const selected = await open({
          directory: true,
          multiple: false,
          title: 'Collaboration output folder',
        });
        if (selected && typeof selected === 'string') {
          setCollabAssetsRoot(selected);
          await persistCollabAssetsRoot(selected);
        }
      } catch (e) {
        setCollabAssetsErr(e instanceof Error ? e.message : String(e));
      }
    };

  if (!isActive) return null;

  return (
    <div className="space-y-8">
<div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Collaboration output folder</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        When a plan is approved, each collaboration gets a sandbox at{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">&lt;folder&gt;/&lt;collaboration-id&gt;/</code>.
        Leave empty to use{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">~/.neural-junkie/collaborations</code>.
        <strong className="text-slack-text"> Browse saves immediately.</strong> Typed paths save when you click Save or leave the field.
        Hub env <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">NEURAL_JUNKIE_COLLAB_ASSETS_DIR</code> overrides this if set at server start.
      </p>
      <div className="flex flex-col sm:flex-row gap-2 mb-3">
        <input
          type="text"
          value={collabAssetsRoot}
          onChange={(e) => {
            setCollabAssetsRoot(e.target.value);
            setCollabAssetsOk(null);
          }}
          onBlur={handleCollabAssetsRootBlur}
          placeholder="~/development/collab-output"
          disabled={collabAssetsSaving}
          className="flex-1 px-3 py-2 text-sm border border-slack-border rounded bg-slack-bg text-slack-text font-mono"
          spellCheck={false}
        />
        <button
          type="button"
          onClick={() => void handleBrowseCollabAssetsRoot()}
          disabled={collabAssetsSaving}
          className="px-3 py-2 text-sm border border-slack-border rounded text-slack-text hover:bg-slack-bgHover disabled:opacity-50"
        >
          Browse…
        </button>
        <button
          type="button"
          onClick={() => void handleCollabAssetsRootSave()}
          disabled={collabAssetsSaving}
          className="px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
        >
          {collabAssetsSaving ? 'Saving…' : 'Save'}
        </button>
      </div>
      {collabAssetsErr && (
        <p className="text-sm text-red-600">{collabAssetsErr}</p>
      )}
      {collabAssetsOk && !collabAssetsErr && (
        <p className="text-sm text-green-600">{collabAssetsOk}</p>
      )}
      {!collabAssetsSaving &&
        !collabAssetsErr &&
        collabAssetsRoot.trim() !== collabAssetsPersisted.trim() && (
          <p className="text-sm text-amber-600">Unsaved changes — Save or tab out of the field.</p>
        )}
    </div>
<div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Agent delegation</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        When enabled, any in-process agent may consult other specialists via the hub (by relevance), then
        synthesize one reply. Applies to normal chat and DMs — not collaboration task orchestration.
      </p>
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={delegationEnabled}
          disabled={delegationSaving}
          onChange={(e) => void handleDelegationToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Enable cross-agent delegation</span>
      </label>
    </div>
<div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Collaboration planning provider</h3>
      <p className="text-sm text-slack-textMuted mb-2">
        Local Ollama models vary in plan quality. For harder collaborations, route <strong>planning discussion</strong>{' '}
        turns through a cloud or larger local provider. Execution tasks still use smart routing / agent defaults.
      </p>
      <p className="text-sm text-slack-textMuted mb-4">
        Recommended: 14B+ local or a configured Claude/OpenAI provider. See{' '}
        <a href="https://github.com/camronwood/neural-junkie/blob/main/docs/HARDWARE.md" className="text-slack-accent hover:underline" target="_blank" rel="noreferrer">
          HARDWARE.md
        </a>{' '}
        for RAM tiers.
      </p>
      <label className="block text-sm text-slack-text mb-1" htmlFor="collab-planning-provider">
        Planning provider
      </label>
      <select
        id="collab-planning-provider"
        className="w-full max-w-md rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text mb-2"
        value={collabPlanningProviderId}
        disabled={collabRoutingSaving}
        onChange={(e) => void handleCollabPlanningProviderChange(e.target.value)}
      >
        <option value="">Use each agent&apos;s default</option>
        {configuredProviders.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name} ({p.id})
          </option>
        ))}
      </select>
    </div>
<div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Benchmark model routing</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        When enabled, the hub picks local Ollama models from benchmark-derived capability profiles for
        collaboration tasks, implementation sessions, and normal chat/DMs. Profiles refresh after{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">make model-benchmark</code>.
      </p>
      {capabilityProfileMeta?.source_run_id && (
        <p className="text-xs text-slack-textMuted mb-3 font-mono">
          Profiles: {capabilityProfileMeta.source_run_id}
          {capabilityProfileMeta.updated_at ? ` · updated ${capabilityProfileMeta.updated_at}` : ''}
        </p>
      )}
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={modelCapabilityRouting}
          disabled={collabRoutingSaving}
          onChange={(e) => void handleModelCapabilityRoutingToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Enable benchmark model routing</span>
      </label>
    </div>
<div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Collaboration smart routing</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        When enabled, the hub picks a configured AI provider for each <strong>collaboration execution task</strong>{' '}
        (assigned task messages after the plan is approved). Normal chat and DMs still use each agent's configured provider.
      </p>
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={collabSmartRouting}
          disabled={collabRoutingSaving}
          onChange={(e) => void handleCollabSmartRoutingToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Enable smart routing for collaboration tasks</span>
      </label>
      <label className="flex items-center gap-3 cursor-pointer mt-4">
        <input
          type="checkbox"
          checked={collabAutoApproveDeliverables}
          disabled={collabRoutingSaving}
          onChange={(e) => void handleCollabAutoApproveToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Auto-approve deliverables under collabs/&lt;id&gt;/</span>
      </label>
      {collabRoutingErr && (
        <p className="text-sm text-red-600 mt-2">{collabRoutingErr}</p>
      )}
    </div>
<div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Implementation sessions</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        IDE Agent mode runs multi-step implementation sessions (read → edit → verify). Local Ollama is
        preferred; fallbacks use configured cloud providers when local tool calling is unavailable.
      </p>
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={implRoutingEnabled}
          disabled={collabRoutingSaving}
          onChange={(e) => setImplRoutingEnabled(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Enable local-first implementation routing</span>
      </label>
      <div className="mt-4">
        <label className="block text-sm text-slack-textMuted mb-1">Implementation tool model (Ollama tag)</label>
        <input
          type="text"
          value={implLocalToolModel}
          disabled={collabRoutingSaving}
          onChange={(e) => setImplLocalToolModel(e.target.value)}
          className="w-full max-w-md px-3 py-2 rounded border border-slack-border bg-slack-bg text-slack-text text-sm"
          placeholder="qwen2.5-coder:7b"
        />
      </div>
      <button
        type="button"
        disabled={
          collabRoutingSaving ||
          (implRoutingEnabled === implRoutingEnabledPersisted &&
            (implLocalToolModel.trim() || 'qwen2.5-coder:7b') === implLocalToolModelPersisted)
        }
        onClick={() => void saveImplementationSettings()}
        className="mt-4 px-4 py-2 text-sm rounded bg-slack-accent text-white disabled:opacity-50"
      >
        Save implementation settings
      </button>
    </div>
    </div>
  );
}
