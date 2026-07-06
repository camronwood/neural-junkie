import { useEffect, useMemo, useState } from 'react';
import { usePacksStore } from '../../stores/packsStore';
import { PACK_CAP } from '../../stores/packCapabilities';
import { ChatAPI } from '../../api/chatAPI';
import { mergeSettingsPut, openExternalLink, type SettingsTabProps } from './settingsShared';
import { ConnectorsSettingsTab } from './ConnectorsSettingsTab';

type AWSForm = {
  default_region: string;
  profile: string;
  sso_start_url: string;
  read_only: boolean;
  write_enabled: boolean;
  allowed_profiles: string;
  allowed_accounts: string;
  org_root_id: string;
  write_audit_path: string;
};

type JiraForm = {
  base_url: string;
  email: string;
  api_token: string;
  default_project_key: string;
};

export function IntegrationsSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const connectorsApi = useMemo(() => new ChatAPI(hubHttp), [hubHttp]);
  const hasAWS = usePacksStore((s) => s.hasCapability(PACK_CAP.AWS_SSO));
  const hasJira = usePacksStore((s) => s.hasCapability(PACK_CAP.JIRA_INTEGRATION));
  const hasGitHub = usePacksStore((s) => s.hasCapability(PACK_CAP.GITHUB_ISSUES_INTEGRATION));
  const hasLinear = usePacksStore((s) => s.hasCapability(PACK_CAP.LINEAR_INTEGRATION));

  const [incidentForm, setIncidentForm] = useState({
    default_provider: 'jira',
    write_mode: false,
    require_approval: true,
  });
  const [githubForm, setGithubForm] = useState({ token: '', default_repo: '' });
  const [linearForm, setLinearForm] = useState({ api_key: '', default_team_id: '' });
  const [pagerdutyForm, setPagerdutyForm] = useState({ api_key: '', default_service_id: '' });
  const [sentryForm, setSentryForm] = useState({ auth_token: '', default_org: '', default_project: '' });

  const [awsForm, setAwsForm] = useState<AWSForm>({
    default_region: 'us-east-2',
    profile: '',
    sso_start_url: '',
    read_only: true,
    write_enabled: false,
    allowed_profiles: '',
    allowed_accounts: '',
    org_root_id: '',
    write_audit_path: '~/.neural-junkie/aws-audit.log',
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
          write_enabled: aws.write_enabled === true,
          allowed_profiles: Array.isArray(aws.allowed_profiles)
            ? (aws.allowed_profiles as string[]).join(', ')
            : String(aws.allowed_profiles ?? ''),
          allowed_accounts: Array.isArray(aws.allowed_accounts)
            ? (aws.allowed_accounts as string[]).join(', ')
            : String(aws.allowed_accounts ?? ''),
          org_root_id: String(aws.org_root_id ?? ''),
          write_audit_path: String(aws.write_audit_path ?? '~/.neural-junkie/aws-audit.log'),
        });
        setJiraForm({
          base_url: String(jira.base_url ?? ''),
          email: String(jira.email ?? ''),
          api_token: String(jira.api_token ?? ''),
          default_project_key: String(jira.default_project_key ?? ''),
        });
        const incident = (cfg.incident ?? {}) as Record<string, unknown>;
        setIncidentForm({
          default_provider: String(incident.default_provider ?? 'jira'),
          write_mode: incident.write_mode === true,
          require_approval: incident.require_approval !== false,
        });
        const gh = (cfg.github_issues ?? {}) as Record<string, unknown>;
        setGithubForm({
          token: String(gh.token ?? ''),
          default_repo: String(gh.default_repo ?? ''),
        });
        const linear = (cfg.linear ?? {}) as Record<string, unknown>;
        setLinearForm({
          api_key: String(linear.api_key ?? ''),
          default_team_id: String(linear.default_team_id ?? ''),
        });
        const pd = (cfg.pagerduty ?? {}) as Record<string, unknown>;
        setPagerdutyForm({
          api_key: String(pd.api_key ?? ''),
          default_service_id: String(pd.default_service_id ?? ''),
        });
        const sentry = (cfg.sentry ?? {}) as Record<string, unknown>;
        setSentryForm({
          auth_token: String(sentry.auth_token ?? ''),
          default_org: String(sentry.default_org ?? ''),
          default_project: String(sentry.default_project ?? ''),
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

  const splitCSV = (raw: string) =>
    raw
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);

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
          write_enabled: awsForm.write_enabled,
          allowed_profiles: splitCSV(awsForm.allowed_profiles),
          allowed_accounts: splitCSV(awsForm.allowed_accounts),
          org_root_id: awsForm.org_root_id.trim(),
          write_audit_path: awsForm.write_audit_path.trim(),
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

  const testAWSOrgAccounts = async () => {
    setAwsErr(null);
    setAwsMsg(null);
    try {
      await saveAWS();
      const r = await fetch(`${hubHttp}/api/integrations/aws/org-accounts`, { method: 'POST' });
      const data = await r.json();
      if (!r.ok) throw new Error(data.error ?? 'List org accounts failed');
      const count = Array.isArray(data.items) ? data.items.length : 0;
      setAwsMsg(`Organization accounts: ${count} listed.`);
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

  const saveIncidentSettings = async () => {
    await mergeSettingsPut(hubHttp, (cfg) => ({
      ...cfg,
      incident: {
        default_provider: incidentForm.default_provider.trim() || 'jira',
        write_mode: incidentForm.write_mode,
        require_approval: incidentForm.require_approval,
      },
      github_issues: {
        token: githubForm.token.trim(),
        default_repo: githubForm.default_repo.trim(),
      },
      linear: {
        api_key: linearForm.api_key.trim(),
        default_team_id: linearForm.default_team_id.trim(),
      },
      pagerduty: {
        api_key: pagerdutyForm.api_key.trim(),
        default_service_id: pagerdutyForm.default_service_id.trim(),
      },
      sentry: {
        auth_token: sentryForm.auth_token.trim(),
        default_org: sentryForm.default_org.trim(),
        default_project: sentryForm.default_project.trim(),
      },
    }));
  };

  if (!isActive) return null;

  return (
    <div className="max-w-3xl space-y-8">
      <div className="rounded-lg border border-slack-border p-6">
        <h4 className="text-base font-semibold text-slack-text">Runbook connectors</h4>
        <p className="mt-1 mb-4 text-sm text-slack-textMuted">
          Store webhook tokens and auth secrets outside runbook JSON. Reference connectors by ID in action tasks.
        </p>
        <ConnectorsSettingsTab api={connectorsApi} />
      </div>

      {!hasAWS && !hasJira ? (
        <div className="text-sm text-slack-textMuted">
          Install and enable the <strong className="text-slack-text">AWS</strong> or{' '}
          <strong className="text-slack-text">Incident management</strong> domain pack to configure additional integrations below.
        </div>
      ) : null}

      {hasAWS || hasJira ? (
        <>
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
            <label className="block text-sm sm:col-span-2">
              <span className="text-slack-textMuted">Allowed profiles (comma-separated, optional)</span>
              <input
                type="text"
                value={awsForm.allowed_profiles}
                onChange={(e) => setAwsForm((f) => ({ ...f, allowed_profiles: e.target.value }))}
                placeholder="prod-admin, dev-readonly"
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
              />
            </label>
            <label className="block text-sm sm:col-span-2">
              <span className="text-slack-textMuted">Allowed account IDs (comma-separated, optional)</span>
              <input
                type="text"
                value={awsForm.allowed_accounts}
                onChange={(e) => setAwsForm((f) => ({ ...f, allowed_accounts: e.target.value }))}
                placeholder="123456789012, 210987654321"
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 font-mono text-sm text-slack-text"
              />
            </label>
            <label className="block text-sm">
              <span className="text-slack-textMuted">Org root ID (optional)</span>
              <input
                type="text"
                value={awsForm.org_root_id}
                onChange={(e) => setAwsForm((f) => ({ ...f, org_root_id: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 font-mono text-sm text-slack-text"
              />
            </label>
            <label className="block text-sm">
              <span className="text-slack-textMuted">Write audit log path</span>
              <input
                type="text"
                value={awsForm.write_audit_path}
                onChange={(e) => setAwsForm((f) => ({ ...f, write_audit_path: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 font-mono text-sm text-slack-text"
              />
            </label>
          </div>
          <label className="mt-3 flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={awsForm.read_only}
              onChange={(e) => setAwsForm((f) => ({ ...f, read_only: e.target.checked }))}
            />
            Read-only AWS tools (recommended)
          </label>
          <label className="mt-2 flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={awsForm.write_enabled}
              onChange={(e) => setAwsForm((f) => ({ ...f, write_enabled: e.target.checked }))}
            />
            Enable write operations (requires confirm_token per mutation; audit log appended)
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
            <button
              type="button"
              onClick={() => void testAWSOrgAccounts()}
              className="rounded border border-slack-border px-4 py-2 text-sm text-slack-text"
            >
              List org accounts
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

      {hasJira && (
        <div className="rounded-lg border border-slack-border p-6">
          <h4 className="text-base font-semibold text-slack-text">Incident settings</h4>
          <p className="mt-1 mb-4 text-sm text-slack-textMuted">
            Default ticketing provider and write-mode gates for IncidentManager mutating tools.
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="text-slack-textMuted">Default provider</span>
              <select
                value={incidentForm.default_provider}
                onChange={(e) => setIncidentForm((f) => ({ ...f, default_provider: e.target.value }))}
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
              >
                <option value="jira">Jira</option>
                <option value="github">GitHub Issues</option>
                <option value="linear">Linear</option>
              </select>
            </label>
          </div>
          <label className="mt-3 flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={incidentForm.write_mode}
              onChange={(e) => setIncidentForm((f) => ({ ...f, write_mode: e.target.checked }))}
            />
            Allow ticket mutations (create, assign, transition, comment)
          </label>
          <label className="mt-2 flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={incidentForm.require_approval}
              onChange={(e) => setIncidentForm((f) => ({ ...f, require_approval: e.target.checked }))}
            />
            Require approval for each mutating tool call
          </label>
          {hasGitHub && (
            <div className="mt-4 grid gap-3">
              <h5 className="text-sm font-medium text-slack-text">GitHub Issues</h5>
              <input
                type="password"
                placeholder="PAT"
                value={githubForm.token}
                onChange={(e) => setGithubForm((f) => ({ ...f, token: e.target.value }))}
                className="w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm"
              />
              <input
                type="text"
                placeholder="owner/repo"
                value={githubForm.default_repo}
                onChange={(e) => setGithubForm((f) => ({ ...f, default_repo: e.target.value }))}
                className="w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm"
              />
            </div>
          )}
          {hasLinear && (
            <div className="mt-4 grid gap-3">
              <h5 className="text-sm font-medium text-slack-text">Linear</h5>
              <input
                type="password"
                placeholder="API key"
                value={linearForm.api_key}
                onChange={(e) => setLinearForm((f) => ({ ...f, api_key: e.target.value }))}
                className="w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm"
              />
              <input
                type="text"
                placeholder="Default team ID"
                value={linearForm.default_team_id}
                onChange={(e) => setLinearForm((f) => ({ ...f, default_team_id: e.target.value }))}
                className="w-full rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm"
              />
            </div>
          )}
          <div className="mt-4 flex gap-2">
            <button
              type="button"
              onClick={() => void saveIncidentSettings()}
              className="rounded bg-slack-accent px-4 py-2 text-sm text-white"
            >
              Save incident integrations
            </button>
          </div>
        </div>
      )}
        </>
      ) : null}
    </div>
  );
}
