import { useCallback, useEffect, useMemo, useState } from 'react';
import { ChatAPI } from '../../api/chatAPI';
import type { CapabilityPolicyResponse } from '../../types/protocol';
import { CapabilityPolicyEditor } from '../CapabilityPolicyEditor';
import type { SettingsTabProps } from './settingsShared';

export function CapabilitiesSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const [policy, setPolicy] = useState<CapabilityPolicyResponse | null>(null);
  const [selectedAgent, setSelectedAgent] = useState('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const next = await new ChatAPI(hubHttp).fetchCapabilityPolicy();
      setPolicy(next);
      setSelectedAgent((current) =>
        next.agents.some((row) => row.agent.id === current)
          ? current
          : next.agents[0]?.agent.id ?? '',
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load capability policy');
    } finally {
      setLoading(false);
    }
  }, [hubHttp]);

  useEffect(() => {
    if (isActive) void load();
  }, [isActive, load]);

  const row = useMemo(
    () => policy?.agents.find((item) => item.agent.id === selectedAgent),
    [policy, selectedAgent],
  );

  const save = async (update: Parameters<ChatAPI['updateCapabilityPolicy']>[0]) => {
    setSaving(true);
    setError(null);
    try {
      await new ChatAPI(hubHttp).updateCapabilityPolicy(update);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save capability policy');
    } finally {
      setSaving(false);
    }
  };

  if (!isActive) return null;

  return (
    <div className="max-w-4xl space-y-6">
      <div>
        <h3 className="text-lg font-semibold text-slack-text">Agent capabilities</h3>
        <p className="mt-1 text-sm text-slack-textMuted">
          Broad safe capabilities are inherited automatically. Sensitive capabilities require an explicit
          grant unless you change the global default below.
        </p>
      </div>

      {error && (
        <div className="rounded border border-red-500/40 bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {error}
        </div>
      )}

      {loading && !policy ? (
        <p className="text-sm text-slack-textMuted">Loading capabilities…</p>
      ) : policy ? (
        <>
          <label className="flex items-start justify-between gap-4 rounded border border-slack-border bg-slack-bgHover p-4">
            <span>
              <span className="block text-sm font-medium text-slack-text">
                Allow sensitive capabilities by default
              </span>
              <span className="mt-1 block text-xs text-slack-textMuted">
                Applies broadly unless an agent has a revoke override. Keeping this off is recommended.
              </span>
            </span>
            <input
              type="checkbox"
              checked={policy.allow_sensitive_by_default}
              disabled={saving}
              onChange={(event) =>
                void save({ allow_sensitive_by_default: event.target.checked })
              }
              className="mt-1 h-4 w-4 accent-slack-accent"
            />
          </label>
          <label className="flex items-start justify-between gap-4 rounded border border-slack-border bg-slack-bgHover p-4">
            <span>
              <span className="block text-sm font-medium text-slack-text">
                Capability handoff channels
              </span>
              <span className="mt-1 block text-xs text-slack-textMuted">
                Let an agent open a temporary room with one capable helper, return the result, and archive the room.
              </span>
            </span>
            <input
              type="checkbox"
              checked={policy.handoffs_enabled}
              disabled={saving}
              onChange={(event) => void save({ handoffs_enabled: event.target.checked })}
              className="mt-1 h-4 w-4 accent-slack-accent"
            />
          </label>

          <section className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-slack-textMuted">Agent policy</label>
              <select
                value={selectedAgent}
                onChange={(event) => setSelectedAgent(event.target.value)}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bgHover px-3 py-2 text-sm text-slack-text"
              >
                {policy.agents.map(({ agent }) => (
                  <option key={agent.id} value={agent.id}>
                    {agent.name} · {agent.type}
                  </option>
                ))}
              </select>
            </div>

            {row ? (
              <CapabilityPolicyEditor
                capabilities={row.state.discoverable}
                state={row.state}
                disabled={saving}
                onChange={(override) =>
                  void save({ agent_key: row.agent.id, override })
                }
              />
            ) : (
              <p className="text-sm text-slack-textMuted">No active agents are available to configure.</p>
            )}
          </section>
        </>
      ) : null}
    </div>
  );
}
