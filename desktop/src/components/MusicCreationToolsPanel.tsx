import { useCallback, useEffect, useState } from 'react';
import type { ACEStepStatus } from '../api/chatAPI';
import { ChatAPI } from '../api/chatAPI';
import { mergeSettingsPut } from './settings/settingsShared';

export interface MusicCreationToolsPanelProps {
  hubHttp: string;
  isActive: boolean;
  packEnabled?: boolean;
}

const MODEL_VARIANTS = [
  { id: 'sft', label: 'SFT — balanced quality (~50 steps)' },
  { id: 'turbo', label: 'Turbo — fast preview (8 steps)' },
  { id: 'xl-sft', label: 'XL SFT — highest quality, more VRAM (~50 steps)' },
  { id: 'xl-turbo', label: 'XL Turbo — fast XL (8 steps)' },
] as const;

const VARIANT_DEFAULTS: Record<string, { steps: number; guidance: string }> = {
  sft: { steps: 50, guidance: '7' },
  turbo: { steps: 8, guidance: '1' },
  'xl-sft': { steps: 50, guidance: '7' },
  'xl-turbo': { steps: 8, guidance: '1' },
};

function statusLabel(st: ACEStepStatus | null): string {
  if (!st) return 'Checking…';
  if (st.demo_mode) return 'Demo mode (NJ_MUSIC_DEMO=1)';
  if (st.installing) {
    const phase = st.install_progress?.phase;
    const detail = st.install_progress?.detail;
    if (phase) return `Installing ACE-Step (${phase}${detail ? `: ${detail}` : ''})…`;
    return 'Installing ACE-Step…';
  }
  if (st.ready) return `Ready (${st.model_variant ?? 'turbo'})`;
  return 'Weights not installed for selected model';
}

function statusClass(st: ACEStepStatus | null): string {
  if (!st) return 'text-slack-textMuted';
  if (st.demo_mode || st.ready) return 'text-green-400';
  if (st.installing) return 'text-amber-300';
  return 'text-amber-400';
}

export function MusicCreationToolsPanel({ hubHttp, isActive, packEnabled = true }: MusicCreationToolsPanelProps) {
  const [status, setStatus] = useState<ACEStepStatus | null>(null);
  const [modelVariant, setModelVariant] = useState('turbo');
  const [inferenceSteps, setInferenceSteps] = useState('50');
  const [guidanceScale, setGuidanceScale] = useState('7');
  const [inferMethod, setInferMethod] = useState<'ode' | 'sde'>('ode');
  const [defaultSeed, setDefaultSeed] = useState('-1');
  const [loading, setLoading] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);

  const loadSettings = useCallback(async () => {
    const r = await fetch(`${hubHttp}/api/settings`);
    if (!r.ok) return;
    const cfg = await r.json();
    const music = (cfg.mcp?.music ?? {}) as Record<string, unknown>;
    const variant = String(music.ace_step_model_variant || 'turbo');
    setModelVariant(variant);
    const defs = VARIANT_DEFAULTS[variant] ?? VARIANT_DEFAULTS.sft;
    setInferenceSteps(String(music.inference_steps ?? defs.steps));
    setGuidanceScale(String(music.guidance_scale ?? defs.guidance));
    setInferMethod(music.infer_method === 'sde' ? 'sde' : 'ode');
    setDefaultSeed(String(music.default_seed ?? -1));
  }, [hubHttp]);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const api = new ChatAPI(hubHttp);
      const [st] = await Promise.all([api.fetchACEStepStatus(), loadSettings()]);
      setStatus(st);
      if (st.model_variant) {
        setModelVariant(st.model_variant);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [hubHttp, loadSettings]);

  useEffect(() => {
    if (!isActive) return;
    void refresh();
  }, [isActive, refresh]);

  useEffect(() => {
    if (!isActive || !status?.installing) return;
    const id = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(id);
  }, [isActive, status?.installing, refresh]);

  const onVariantChange = (variant: string) => {
    setModelVariant(variant);
    const defs = VARIANT_DEFAULTS[variant] ?? VARIANT_DEFAULTS.sft;
    setInferenceSteps(String(defs.steps));
    setGuidanceScale(defs.guidance);
  };

  const restartMusicSidecar = async () => {
    const api = new ChatAPI(hubHttp);
    await api.restartMusicSidecar('music-creation');
  };

  const persistMusicSettings = async (restartSidecar: boolean) => {
    const steps = parseInt(inferenceSteps, 10);
    const guidance = parseFloat(guidanceScale);
    const seed = parseInt(defaultSeed, 10);
    if (!Number.isFinite(steps) || steps < 1 || steps > 600) {
      throw new Error('Inference steps must be 1–600');
    }
    if (!Number.isFinite(guidance) || guidance <= 0) {
      throw new Error('Guidance scale must be positive');
    }
    await mergeSettingsPut(hubHttp, (cfg) => ({
      ...cfg,
      mcp: {
        ...(cfg.mcp as object | undefined),
        music: {
          ace_step_model_variant: modelVariant,
          inference_steps: steps,
          guidance_scale: guidance,
          infer_method: inferMethod,
          default_seed: Number.isFinite(seed) ? seed : -1,
        },
      },
    }));
    if (restartSidecar) {
      await restartMusicSidecar();
    }
  };

  const saveSettings = async () => {
    if (!packEnabled) {
      setError('Enable the Music creation pack from Domain packs → Store before saving.');
      return;
    }
    setSaving(true);
    setError(null);
    setOk(null);
    try {
      await persistMusicSettings(true);
      setOk('Music generation settings saved. Sidecar restarted.');
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  const runInstall = async () => {
    const variantLabel = MODEL_VARIANTS.find((v) => v.id === modelVariant)?.label ?? modelVariant;
    const confirmed = window.confirm(
      `Download and install ACE-Step 1.5 (${variantLabel})?\n\n` +
        'This clones the ACE-Step repo, creates a Python 3.12 venv, and downloads model weights (~several GB). ' +
        'It can take 10–30 minutes depending on your connection.\n\n' +
        'Requires Python 3.11 or 3.12 (pyenv or Homebrew).',
    );
    if (!confirmed) return;
    setInstalling(true);
    setError(null);
    setOk(null);
    try {
      await persistMusicSettings(false);
      const api = new ChatAPI(hubHttp);
      const resp = await api.installACEStep('music-creation', modelVariant);
      setStatus(resp.acestep);
      await restartMusicSidecar();
      setOk('ACE-Step installed. Try /generate-music in chat.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      await refresh();
    } finally {
      setInstalling(false);
    }
  };

  const busy = loading || installing || saving || status?.installing;

  return (
    <div className="rounded-lg border border-slack-border p-4">
      <h3 className="text-base font-semibold text-slack-text mb-2">Music creation — ACE-Step</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        Full song generation uses a local ACE-Step 1.5 sidecar. Pick a model variant, tune inference
        settings, then install weights for that variant. Ollama models for lyrics (
        <code className="font-mono text-xs">qwen2.5:7b</code>,{' '}
        <code className="font-mono text-xs">qwen3.5:9b</code>) pull when Ollama is running.
      </p>

      <div className="mb-4 grid gap-3 sm:grid-cols-2">
        <label className="block text-sm sm:col-span-2">
          <span className="text-slack-textMuted">Model variant</span>
          <select
            value={modelVariant}
            onChange={(e) => onVariantChange(e.target.value)}
            className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
          >
            {MODEL_VARIANTS.map((v) => (
              <option key={v.id} value={v.id}>
                {v.label}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="text-slack-textMuted">Inference steps</span>
          <input
            type="number"
            min={1}
            max={600}
            value={inferenceSteps}
            onChange={(e) => setInferenceSteps(e.target.value)}
            className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 font-mono text-sm text-slack-text"
          />
        </label>
        <label className="block text-sm">
          <span className="text-slack-textMuted">Guidance scale</span>
          <input
            type="number"
            min={0.1}
            step={0.1}
            value={guidanceScale}
            onChange={(e) => setGuidanceScale(e.target.value)}
            className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 font-mono text-sm text-slack-text"
          />
        </label>
        <label className="block text-sm">
          <span className="text-slack-textMuted">Infer method</span>
          <select
            value={inferMethod}
            onChange={(e) => setInferMethod(e.target.value === 'sde' ? 'sde' : 'ode')}
            className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
          >
            <option value="ode">ODE (default)</option>
            <option value="sde">SDE (more variation)</option>
          </select>
        </label>
        <label className="block text-sm">
          <span className="text-slack-textMuted">Default seed (-1 = random)</span>
          <input
            type="number"
            value={defaultSeed}
            onChange={(e) => setDefaultSeed(e.target.value)}
            className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 font-mono text-sm text-slack-text"
          />
        </label>
      </div>

      <p className={`text-sm font-medium mb-3 ${statusClass(status)}`}>{statusLabel(status)}</p>
      {status && !status.demo_mode && (
        <ul className="mb-3 space-y-1 text-xs font-mono text-slack-textMuted">
          <li>Variant: {status.model_variant ?? modelVariant}</li>
          <li>Python venv: {status.venv_ready ? '✓' : '—'} {status.paths.venv}</li>
          <li>ACE-Step project: {status.project_ready ? '✓' : '—'} {status.paths.project}</li>
          <li>Model weights: {status.checkpoint_ready ? '✓' : '—'} {status.paths.checkpoint}</li>
          {status.python_version && <li>{status.python_version}</li>}
        </ul>
      )}
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void saveSettings()}
          className="rounded bg-slack-accent px-4 py-2 text-sm text-white hover:bg-slack-accentHover disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save settings'}
        </button>
        {!status?.ready && !status?.demo_mode && (
          <button
            type="button"
            disabled={busy}
            onClick={() => void runInstall()}
            className="rounded bg-slack-accent px-4 py-2 text-sm text-white hover:bg-slack-accentHover disabled:opacity-50"
          >
            {installing ? 'Installing…' : `Install ${modelVariant} weights`}
          </button>
        )}
        <button
          type="button"
          disabled={busy}
          onClick={() => void refresh()}
          className="rounded border border-slack-border px-4 py-2 text-sm text-slack-text hover:bg-slack-bgHover disabled:opacity-50"
        >
          Refresh status
        </button>
      </div>
      {status?.last_error && (
        <p className="mt-2 whitespace-pre-wrap text-sm text-red-400">{status.last_error}</p>
      )}
      {error && <p className="mt-2 text-sm text-red-500">{error}</p>}
      {ok && <p className="mt-2 text-sm text-green-600">{ok}</p>}
      {!status?.ready && !status?.demo_mode && (
        <p className="mt-3 text-xs text-slack-textMuted">
          XL variants need more disk and VRAM. Try <strong>turbo</strong> for quick previews,{' '}
          <strong>xl-sft</strong> for best quality. UI smoke test:{' '}
          <code className="font-mono">NJ_MUSIC_DEMO=1</code> + hub restart.
        </p>
      )}
    </div>
  );
}
