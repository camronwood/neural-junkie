import { useEffect, useState } from 'react';
import { putSystemSecurity, type SettingsTabProps } from './settingsShared';

export function AutomationSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const [form, setForm] = useState({
    scenario_repo: '',
    scenario_allow_file_fallback: false,
    deliverable_judge_provider: 'claude',
    deliverable_judge_mode: 'hub',
    deliverable_judge_model: 'qwen2.5-coder:14b',
    deliverable_judge_gemini_model: '',
    deliverable_judge_agent: '',
    deliverable_judge_timeout: 180,
    deliverable_judge_skip: false,
    deliverable_judge_fallback_ollama: true,
    deliverable_judge_min_interval_s: 13,
    agent_poll: false,
    human_name: '',
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [ok, setOk] = useState(false);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/system/security`);
        if (!r.ok) throw new Error(await r.text());
        const data = await r.json();
        const a = data.automation ?? {};
        if (!cancelled) {
          setForm({
            scenario_repo: String(a.scenario_repo ?? ''),
            scenario_allow_file_fallback: !!a.scenario_allow_file_fallback,
            deliverable_judge_provider: String(a.deliverable_judge_provider ?? 'claude'),
            deliverable_judge_mode: String(a.deliverable_judge_mode ?? 'hub'),
            deliverable_judge_model: String(a.deliverable_judge_model ?? 'qwen2.5-coder:14b'),
            deliverable_judge_gemini_model: String(a.deliverable_judge_gemini_model ?? ''),
            deliverable_judge_agent: String(a.deliverable_judge_agent ?? ''),
            deliverable_judge_timeout: Number(a.deliverable_judge_timeout ?? 180),
            deliverable_judge_skip: !!a.deliverable_judge_skip,
            deliverable_judge_fallback_ollama: a.deliverable_judge_fallback_ollama !== false,
            deliverable_judge_min_interval_s: Number(a.deliverable_judge_min_interval_s ?? 13),
            agent_poll: !!a.agent_poll,
            human_name: String(a.human_name ?? ''),
          });
        }
      } catch (e) {
        if (!cancelled) setErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp]);

  const save = async () => {
    setBusy(true);
    setErr(null);
    setOk(false);
    try {
      await putSystemSecurity(hubHttp, { automation: form });
      setOk(true);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  if (!isActive) return null;

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h3 className="text-lg font-semibold text-slack-text mb-1">Automation & testing</h3>
        <p className="text-sm text-slack-textMuted">
          Scenario harness and deliverable judge defaults (used by collab-scenarios.py when env vars
          are unset). Environment variables still override.
        </p>
      </div>
      {err && <p className="text-sm text-red-400">{err}</p>}
      {ok && <p className="text-sm text-emerald-400">Automation settings saved.</p>}

      <label className="block text-sm">
        <span className="text-slack-textMuted">Scenario repo root</span>
        <input
          className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text font-mono"
          value={form.scenario_repo}
          onChange={(e) => setForm((f) => ({ ...f, scenario_repo: e.target.value }))}
        />
      </label>
      <label className="flex gap-2 items-center text-sm">
        <input
          type="checkbox"
          checked={form.scenario_allow_file_fallback}
          onChange={(e) => setForm((f) => ({ ...f, scenario_allow_file_fallback: e.target.checked }))}
        />
        Allow scenario file-write fallback (dev only)
      </label>
      <label className="block text-sm">
        <span className="text-slack-textMuted">Human display name (messages API)</span>
        <input
          className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
          value={form.human_name}
          onChange={(e) => setForm((f) => ({ ...f, human_name: e.target.value }))}
        />
      </label>
      <label className="flex gap-2 items-center text-sm">
        <input
          type="checkbox"
          checked={form.agent_poll}
          onChange={(e) => setForm((f) => ({ ...f, agent_poll: e.target.checked }))}
        />
        Standalone agent poll transport
      </label>

      <h4 className="text-sm font-medium text-slack-text pt-2">Deliverable judge</h4>
      <div className="grid grid-cols-2 gap-3 text-sm">
        <label className="block">
          <span className="text-slack-textMuted">Provider</span>
          <input
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            value={form.deliverable_judge_provider}
            onChange={(e) => setForm((f) => ({ ...f, deliverable_judge_provider: e.target.value }))}
          />
        </label>
        <label className="block">
          <span className="text-slack-textMuted">Mode</span>
          <input
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            value={form.deliverable_judge_mode}
            onChange={(e) => setForm((f) => ({ ...f, deliverable_judge_mode: e.target.value }))}
          />
        </label>
        <label className="block col-span-2">
          <span className="text-slack-textMuted">Gemini model</span>
          <input
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text font-mono"
            value={form.deliverable_judge_gemini_model}
            onChange={(e) => setForm((f) => ({ ...f, deliverable_judge_gemini_model: e.target.value }))}
          />
        </label>
        <label className="block">
          <span className="text-slack-textMuted">Ollama fallback model</span>
          <input
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text font-mono"
            value={form.deliverable_judge_model}
            onChange={(e) => setForm((f) => ({ ...f, deliverable_judge_model: e.target.value }))}
          />
        </label>
        <label className="block">
          <span className="text-slack-textMuted">Timeout (s)</span>
          <input
            type="number"
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            value={form.deliverable_judge_timeout}
            onChange={(e) =>
              setForm((f) => ({ ...f, deliverable_judge_timeout: Number(e.target.value) }))
            }
          />
        </label>
      </div>
      <label className="flex gap-2 items-center text-sm">
        <input
          type="checkbox"
          checked={form.deliverable_judge_skip}
          onChange={(e) => setForm((f) => ({ ...f, deliverable_judge_skip: e.target.checked }))}
        />
        Skip LLM judge (regex only)
      </label>
      <label className="flex gap-2 items-center text-sm">
        <input
          type="checkbox"
          checked={form.deliverable_judge_fallback_ollama}
          onChange={(e) =>
            setForm((f) => ({ ...f, deliverable_judge_fallback_ollama: e.target.checked }))
          }
        />
        Fallback to Ollama when cloud judge fails
      </label>

      <button
        type="button"
        disabled={busy}
        onClick={() => void save()}
        className="px-4 py-2 rounded bg-slack-accent text-white text-sm hover:opacity-90 disabled:opacity-50"
      >
        {busy ? 'Saving…' : 'Save automation settings'}
      </button>
    </div>
  );
}
