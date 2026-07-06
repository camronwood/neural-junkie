import { useCallback, useEffect, useState } from 'react';
import { ChatAPI } from '../../../api/chatAPI';
import type {
  SlackConnectionResponse,
  SlackDiagnoseResult,
  SlackSmokeResult,
  SlackStatus,
} from '../../../types/protocol';

function statusIcon(status: string): string {
  switch (status) {
    case 'pass':
      return '✓';
    case 'warn':
      return '⚠';
    case 'fail':
      return '✗';
    case 'skip':
      return '–';
    default:
      return '•';
  }
}

function statusClass(status: string): string {
  switch (status) {
    case 'pass':
      return 'text-green-700';
    case 'warn':
      return 'text-amber-600';
    case 'fail':
      return 'text-red-600';
    default:
      return 'text-slack-textMuted';
  }
}

export interface SlackSetupChecklistProps {
  hubHttp: string;
  botTokenSet: boolean;
  slackStatus: SlackStatus | null;
  slackConnection: SlackConnectionResponse | null;
  bindingsCount: number;
  smokeChannelId?: string;
}

export function SlackSetupChecklist({
  hubHttp,
  botTokenSet,
  slackStatus,
  slackConnection,
  bindingsCount,
  smokeChannelId,
}: SlackSetupChecklistProps) {
  const [diagnose, setDiagnose] = useState<SlackDiagnoseResult | null>(null);
  const [diagnoseLoading, setDiagnoseLoading] = useState(false);
  const [smokeBusy, setSmokeBusy] = useState(false);
  const [smokeResult, setSmokeResult] = useState<SlackSmokeResult | null>(null);
  const [smokeError, setSmokeError] = useState<string | null>(null);

  const loadDiagnose = useCallback(async () => {
    if (!botTokenSet) {
      setDiagnose(null);
      return;
    }
    setDiagnoseLoading(true);
    try {
      const api = new ChatAPI(hubHttp);
      setDiagnose(await api.getSlackDiagnose());
    } catch {
      setDiagnose(null);
    } finally {
      setDiagnoseLoading(false);
    }
  }, [botTokenSet, hubHttp]);

  useEffect(() => {
    void loadDiagnose();
  }, [loadDiagnose, slackStatus?.configured, slackConnection?.bridge_connected, bindingsCount]);

  const runSmoke = async () => {
    setSmokeBusy(true);
    setSmokeError(null);
    setSmokeResult(null);
    try {
      const api = new ChatAPI(hubHttp);
      const channelId =
        smokeChannelId?.trim() || undefined;
      const result = await api.runSlackSmoke({
        channel_id: channelId,
        outbound: false,
      });
      setSmokeResult(result);
      if (!result.ok) {
        setSmokeError('Bridge test did not pass — see checks below.');
      }
    } catch (e) {
      setSmokeError(e instanceof Error ? e.message : 'Bridge test failed');
    } finally {
      setSmokeBusy(false);
    }
  };

  const workspaceOk = Boolean(slackConnection?.bot_token_set);
  const bridgeOk = Boolean(slackConnection?.bridge_connected ?? slackStatus?.connected);

  return (
    <div className="mb-4 p-4 bg-slack-bgHover rounded-lg border border-slack-border space-y-3">
      <div className="flex items-center justify-between gap-2">
        <h4 className="text-sm font-semibold text-slack-text">Setup checklist</h4>
        <button
          type="button"
          onClick={() => void loadDiagnose()}
          disabled={diagnoseLoading || !botTokenSet}
          className="px-2 py-1 text-xs border border-slack-border rounded hover:bg-slack-bg text-slack-text disabled:opacity-50"
        >
          {diagnoseLoading ? 'Checking…' : 'Refresh'}
        </button>
      </div>

      <ul className="space-y-1.5 text-sm">
        <li className={workspaceOk ? 'text-green-700' : 'text-slack-textMuted'}>
          {workspaceOk ? '✓' : '○'} Workspace connected
          {slackConnection?.team_name ? ` (${slackConnection.team_name})` : ''}
        </li>
        <li className={bridgeOk ? 'text-green-700' : 'text-slack-textMuted'}>
          {bridgeOk ? '✓' : '○'} Bridge connected
        </li>
        {diagnose?.checks?.map((c) => (
          <li key={c.id} className={statusClass(c.status)}>
            <span className="mr-1">{statusIcon(c.status)}</span>
            {c.label}
            {c.fix && c.status !== 'pass' ? (
              <span className="block text-xs text-slack-textMuted ml-4">{c.fix}</span>
            ) : null}
          </li>
        ))}
      </ul>

      {!botTokenSet && (
        <p className="text-xs text-slack-textMuted">Connect Slack to run the setup checklist.</p>
      )}

      {slackStatus?.bot_user_id && (diagnose?.channels_found ?? 0) === 0 && (
        <p className="text-xs text-amber-600">
          Invite the bot:{' '}
          <code className="bg-slack-bg px-1 rounded">/invite @Neural Junkie</code> in your channel
        </p>
      )}

      <button
        type="button"
        onClick={() => void runSmoke()}
        disabled={smokeBusy || !botTokenSet}
        className="px-3 py-1.5 text-sm border border-slack-border rounded hover:bg-slack-bg text-slack-text disabled:opacity-50"
      >
        {smokeBusy ? 'Testing…' : 'Test bridge (no Slack message)'}
      </button>

      {smokeError && <p className="text-xs text-red-600">{smokeError}</p>}
      {smokeResult?.ok && (
        <p className="text-xs text-green-700">
          Bridge test passed in {smokeResult.duration_ms}ms
          {smokeResult.outbound_skipped ? ' (no Slack messages sent)' : ''}.
        </p>
      )}
      {smokeResult && !smokeResult.ok && (
        <ul className="text-xs space-y-1">
          {smokeResult.checks.map((c) => (
            <li key={c.id} className={statusClass(c.status)}>
              {statusIcon(c.status)} {c.id}: {c.detail || c.status}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
