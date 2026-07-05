import { useEffect, useState } from 'react';
import { useChatStore } from '../../stores/chatStore';
import {
  loadConnectionSettings,
  saveConnectionSettings,
  type ConnectionSettings,
} from '../../stores/connectionStore';
import { DEFAULT_HUB_HTTP } from '../../config/hubUrl';
import type { SettingsTabProps } from './settingsShared';

export function ConnectionSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const setServerAddr = useChatStore((s) => s.setServerAddr);
  const [form, setForm] = useState<ConnectionSettings>({
    hubUrl: DEFAULT_HUB_HTTP,
    hubToken: '',
  });
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!isActive) return;
    void loadConnectionSettings().then(setForm);
  }, [isActive]);

  const save = async () => {
    setBusy(true);
    setErr(null);
    setMsg(null);
    try {
      await saveConnectionSettings(form);
      setServerAddr(form.hubUrl);
      setMsg('Connection settings saved. Reconnect if the hub URL changed.');
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
        <h3 className="text-lg font-semibold text-slack-text mb-1">Hub connection</h3>
        <p className="text-sm text-slack-textMuted">
          Desktop hub URL and client token. Overrides build-time{' '}
          <code className="font-mono text-xs">VITE_NJ_HUB_URL</code> /{' '}
          <code className="font-mono text-xs">VITE_NJ_HUB_TOKEN</code> when set. Current API base:{' '}
          <code className="font-mono text-xs">{hubHttp}</code>
        </p>
      </div>
      {err && <p className="text-sm text-red-400">{err}</p>}
      {msg && <p className="text-sm text-emerald-400">{msg}</p>}
      <label className="block text-sm">
        <span className="text-slack-textMuted">Hub URL</span>
        <input
          className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text font-mono"
          value={form.hubUrl}
          onChange={(e) => setForm((f) => ({ ...f, hubUrl: e.target.value }))}
          placeholder={DEFAULT_HUB_HTTP}
        />
      </label>
      <label className="block text-sm">
        <span className="text-slack-textMuted">Hub token (client)</span>
        <input
          type="password"
          className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text font-mono"
          value={form.hubToken}
          onChange={(e) => setForm((f) => ({ ...f, hubToken: e.target.value }))}
          placeholder="Optional — matches hub security hub_token"
        />
      </label>
      <button
        type="button"
        disabled={busy}
        onClick={() => void save()}
        className="px-4 py-2 rounded bg-slack-accent text-white text-sm hover:opacity-90 disabled:opacity-50"
      >
        {busy ? 'Saving…' : 'Save connection'}
      </button>
    </div>
  );
}
