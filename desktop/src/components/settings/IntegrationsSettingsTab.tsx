import { useEffect, useState } from 'react';
import { usePacksStore } from '../../stores/packsStore';
import { PACK_CAP } from '../../stores/packCapabilities';
import { mergeSettingsPut, openExternalLink, type SettingsTabProps } from './settingsShared';

type AWSForm = {
  default_region: string;
  profile: string;
  sso_start_url: string;
  read_only: boolean;
};

type JiraForm = {
  base_url: string;
  email: string;
  api_token: string;
  default_project_key: string;
};

export function IntegrationsSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const hasAWS = usePacksStore((s) => s.hasCapability(PACK_CAP.AWS_SSO));
  const hasJira = usePacksStore((s) => s.hasCapability(PACK_CAP.JIRA_INTEGRATION));

  const [awsForm, setAwsForm] = useState<AWSForm>({
    default_region: 'us-east-2',
    profile: '',
    sso_start_url: '',
    read_only: true,
  });
  const [jiraForm, setJiraForm] = useState<JiraForm>({
    base_url: '',
    email: '',
    api_token: '',
    default_project_key: '',
  });
  const [profiles, setProfiles] = useState<string[]>([]);
  const [awsSaving, setAwsSaving] = useState(false);
  const [jiraSaving, setJiraSaving] = useState(false);
  const [awsMsg, setAwsMsg] = useState<string | null>(null);
  const [jiraMsg, setJiraMsg] = useState<string | null>(null);
  const [awsErr, setAwsErr] = useState<string | null>(null);
  const [jiraErr, setJiraErr] = useState<string | null>(null);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) throw new Error(await r.text());
        const cfg = await r.json();
        if (cancelled) return;
        const aws = (cfg.aws ?? {}) as Record<string, unknown>;
        const jira = (cfg.jira ?? {}) as Record<string, unknown>;
        setAwsForm({
          default_region: String(aws.default_region ?? 'us-east-2'),
          profile: String(aws.profile ?? ''),
          sso_start_url: String(aws.sso_start_url ?? ''),
          read_only: aws.read_only !== false,
        });
        setJiraForm({
          base_url: String(jira.base_url ?? ''),
          email: String(jira.email ?? ''),
          api_token: String(jira.api_token ?? ''),
          default_project_key: String(jira.default_project_key ?? ''),
        });
      } catch (e) {
        if (!cancelled) setAwsErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp]);

  useEffect(() => {
    if (!isActive || !hasAWS) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/integrations/aws/profiles`);
        if (!r.ok) throw new Error(await r.text());
        const data = (await r.json()) as { profiles?: string[] };
        if (!cancelled) setProfiles(data.profiles ?? []);
      } catch {
        if (!cancelled) setProfiles([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hasAWS, hubHttp]);

  const saveAWS = async () => {
    setAwsSaving(true);
    setAwsErr(null);
    setAwsMsg(null);
    try {
      await mergeSettingsPut(hubHttp, (cfg) => ({
        ...cfg,
        aws: {
          default_region: awsForm.default_region.trim(),
          profile: awsForm.profile.trim(),
          sso_start_url: awsForm.sso_start_url.trim(),
          read_only: awsForm.read_only,
        },
      }));
      setAwsMsg('Saved AWS integration settings.');
    } catch (e) {
      setAwsErr(e instanceof Error ? e.message : String(e));
    } finally {
      setAwsSaving(false);
    }
  };

  const testAWS = async () => {
    setAwsErr(null);
    setAwsMsg(null);
    try {
      await saveAWS();
      const r = await fetch(`${hubHttp}/api/integrations/aws/test`, { method: 'POST' });
      const data = await r.json();
      if (!r.ok) throw new Error(data.error ?? 'AWS test failed');
      setAwsMsg(`Connected: ${data.output ?? data.status}`);
    } catch (e) {
      setAwsErr(e instanceof Error ? e.message : String(e));
    }
  };

  const saveJira = async () => {
    setJiraSaving(true);
    setJiraErr(null);
    setJiraMsg(null);
    try {
      await mergeSettingsPut(hubHttp, (cfg) => ({
        ...cfg,
        jira: {
          base_url: jiraForm.base_url.trim(),
          email: jiraForm.email.trim(),
          api_token: jiraForm.api_token.trim(),
          default_project_key: jiraForm.default_project_key.trim(),
        },
      }));
      setJiraMsg('Saved Jira integration settings.');
    } catch (e) {
      setJiraErr(e instanceof Error ? e.message : String(e));
    } finally {
      setJiraSaving(false);
    }
  };

  const testJira = async () => {
    setJiraErr(null);
    setJiraMsg(null);
    try {
      await saveJira();
      const r = await fetch(`${hubHttp}/api/integrations/jira/test`, { method: 'POST' });
      const data = await r.json();
      if (!r.ok) throw new Error(data.error ?? 'Jira test failed');
      setJiraMsg('Jira connection OK.');
    } catch (e) {
      setJiraErr(e instanceof Error ? e.message : String(e));
    }
  };

  if (!isActive) return null;

  if (!hasAWS && !hasJira) {
    return (
      <div className="max-w-2xl text-sm text-slack-textMuted">
        Install and enable the <strong className="text-slack-text">AWS</strong> or{' '}
        <strong className="text-slack-text">Incident management</strong> domain pack to configure integrations here.
      </div>
    );
  }

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h3 className="text-lg font-semibold text-slack-text">Integrations</h3>
        <p className="mt-1 text-sm text-slack-textMuted">
          Connect external services for domain pack specialists. Credentials are stored in hub config on this machine.
        </p>
      </div>

      {hasAWS && (
        <div className="rounded-lg border border-slack-border p-6">
          <h4 className="text-base font-semibold text-slack-text">AWS (SSO profiles)</h4>
          <p className="mt-1 mb-4 text-sm text-slack-textMuted">
            Uses named profiles from <code className="rounded bg-slack-bgHover px-1">~/.aws/config</code>. Run{' '}
            <code className="rounded bg-slack-bgHover px-1">aws sso login --profile …</code> in a terminal before testing.
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="text-slack-textMuted">Default region</span>
              <input
                type="text"
                value={awsForm.default_region}
                onChange={(e) => setAwsForm((f) => ({ ...f, default_region: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 font-mono text-sm text-slack-text"
              />
            </label>
            <label className="block text-sm">
              <span className="text-slack-textMuted">Active profile</span>
              <select
                value={awsForm.profile}
                onChange={(e) => setAwsForm((f) => ({ ...f, profile: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
              >
                <option value="">Select profile…</option>
                {profiles.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm sm:col-span-2">
              <span className="text-slack-textMuted">SSO start URL (reference)</span>
              <input
                type="text"
                value={awsForm.sso_start_url}
                onChange={(e) => setAwsForm((f) => ({ ...f, sso_start_url: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
              />
            </label>
          </div>
          <label className="mt-3 flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={awsForm.read_only}
              onChange={(e) => setAwsForm((f) => ({ ...f, read_only: e.target.checked }))}
            />
            Read-only AWS CLI (recommended)
          </label>
          <div className="mt-4 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => void saveAWS()}
              disabled={awsSaving}
              className="rounded bg-slack-accent px-4 py-2 text-sm text-white disabled:opacity-50"
            >
              Save
            </button>
            <button
              type="button"
              onClick={() => void testAWS()}
              className="rounded border border-slack-border px-4 py-2 text-sm text-slack-text"
            >
              Test connection
            </button>
            {awsForm.profile && (
              <button
                type="button"
                onClick={() =>
                  navigator.clipboard?.writeText(`aws sso login --profile ${awsForm.profile}`)
                }
                className="rounded border border-slack-border px-4 py-2 text-sm text-slack-text"
              >
                Copy SSO login command
              </button>
            )}
          </div>
          {awsMsg && <p className="mt-2 text-sm text-green-600">{awsMsg}</p>}
          {awsErr && <p className="mt-2 text-sm text-red-500">{awsErr}</p>}
        </div>
      )}

      {hasJira && (
        <div className="rounded-lg border border-slack-border p-6">
          <h4 className="text-base font-semibold text-slack-text">Jira Cloud</h4>
          <p className="mt-1 mb-4 text-sm text-slack-textMuted">
            API token auth for IncidentManager.{' '}
            <button
              type="button"
              className="text-slack-accent underline"
              onClick={() =>
                openExternalLink('https://id.atlassian.com/manage-profile/security/api-tokens')
              }
            >
              Create an API token
            </button>
          </p>
          <div className="grid gap-3">
            <label className="block text-sm">
              <span className="text-slack-textMuted">Site URL</span>
              <input
                type="url"
                placeholder="https://yourorg.atlassian.net"
                value={jiraForm.base_url}
                onChange={(e) => setJiraForm((f) => ({ ...f, base_url: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
              />
            </label>
            <label className="block text-sm">
              <span className="text-slack-textMuted">Email</span>
              <input
                type="email"
                value={jiraForm.email}
                onChange={(e) => setJiraForm((f) => ({ ...f, email: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
              />
            </label>
            <label className="block text-sm">
              <span className="text-slack-textMuted">API token</span>
              <input
                type="password"
                value={jiraForm.api_token}
                onChange={(e) => setJiraForm((f) => ({ ...f, api_token: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 font-mono text-sm text-slack-text"
              />
            </label>
            <label className="block text-sm">
              <span className="text-slack-textMuted">Default project key</span>
              <input
                type="text"
                placeholder="ENG"
                value={jiraForm.default_project_key}
                onChange={(e) => setJiraForm((f) => ({ ...f, default_project_key: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
              />
            </label>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => void saveJira()}
              disabled={jiraSaving}
              className="rounded bg-slack-accent px-4 py-2 text-sm text-white disabled:opacity-50"
            >
              Save
            </button>
            <button
              type="button"
              onClick={() => void testJira()}
              className="rounded border border-slack-border px-4 py-2 text-sm text-slack-text"
            >
              Test connection
            </button>
          </div>
          {jiraMsg && <p className="mt-2 text-sm text-green-600">{jiraMsg}</p>}
          {jiraErr && <p className="mt-2 text-sm text-red-500">{jiraErr}</p>}
        </div>
      )}
    </div>
  );
}
