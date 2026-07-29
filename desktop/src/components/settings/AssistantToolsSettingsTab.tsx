import { useState, useEffect } from 'react';
import { useSettingsStore } from '../../stores/settingsStore';
import { ChatAPI } from '../../api/chatAPI';
import type {
  GoogleMeetNotesSettings,
  GoogleMeetNotesStatus,
} from '../../types/protocol';
import { openExternalLinkAsync, type SettingsTabProps } from './settingsShared';

export function AssistantToolsSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const { integrations, updateGoogleMeetNotesSettings } = useSettingsStore();

    const [googleOAuthForm, setGoogleOAuthForm] = useState<GoogleMeetNotesSettings>(integrations.googleMeetNotes);
    const [googleOAuthSecretSet, setGoogleOAuthSecretSet] = useState(false);
    const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});
    const [googleMeetNotes, setGoogleMeetNotes] = useState<GoogleMeetNotesStatus | null>(null);
    const [googleMeetNotesLoading, setGoogleMeetNotesLoading] = useState(false);
    const [googleMeetNotesBusy, setGoogleMeetNotesBusy] = useState(false);
    const [googleAdvancedOpen, setGoogleAdvancedOpen] = useState(false);
    const refreshGoogleMeetNotesStatus = async () => {
      setGoogleMeetNotesLoading(true);
      try {
        const api = new ChatAPI(hubHttp);
        const [status, appConfig] = await Promise.all([
          api.getGoogleMeetNotesStatus(),
          api.getGoogleMeetNotesAppConfig().catch(() => null),
        ]);
        setGoogleMeetNotes({
          ...status,
          connect_ready: status.connect_ready ?? appConfig?.connect_ready ?? appConfig?.configured ?? false,
          oauth_source: status.oauth_source ?? appConfig?.oauth_source,
          oauth_configured: status.oauth_configured || appConfig?.configured === true,
        });
        if (appConfig) {
          setGoogleOAuthForm((prev) => ({
            ...prev,
            clientId: appConfig.client_id || prev.clientId,
            redirectUrl: appConfig.redirect_url || prev.redirectUrl,
          }));
          setGoogleOAuthSecretSet(appConfig.secret_set);
        }
      } catch (e) {
        setGoogleMeetNotes({
          connected: false,
          oauth_configured: false,
        });
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: {
            success: false,
            message: e instanceof Error ? e.message : 'Failed to load status',
          },
        }));
      } finally {
        setGoogleMeetNotesLoading(false);
      }
    };
  useEffect(() => {
    setGoogleOAuthForm(integrations.googleMeetNotes);
  }, [integrations]);

  useEffect(() => {
    if (!isActive) return;
    void refreshGoogleMeetNotesStatus();
  }, [isActive, hubHttp]);

    const saveGoogleOAuthSettings = async () => {
      setGoogleMeetNotesBusy(true);
      try {
        const api = new ChatAPI(hubHttp);
        await api.saveGoogleMeetNotesAppConfig(
          googleOAuthForm.clientId,
          googleOAuthForm.clientSecret,
          googleOAuthForm.redirectUrl
        );
        await updateGoogleMeetNotesSettings(googleOAuthForm);
        setGoogleOAuthSecretSet(true);
        setGoogleOAuthForm((prev) => ({ ...prev, clientSecret: '' }));
        await refreshGoogleMeetNotesStatus();
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: {
            success: true,
            message: 'OAuth app credentials saved on the hub.',
          },
        }));
      } catch (e) {
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: {
            success: false,
            message: e instanceof Error ? e.message : 'Failed to save OAuth credentials',
          },
        }));
      } finally {
        setGoogleMeetNotesBusy(false);
      }
    };

    const connectGoogleMeetNotes = async () => {
      setGoogleMeetNotesBusy(true);
      try {
        const api = new ChatAPI(hubHttp);
        const url = await api.getGoogleMeetNotesAuthURL();
        const opened = await openExternalLinkAsync(url);
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: {
            success: true,
            message: opened
              ? 'Complete sign-in in your browser, then refresh status.'
              : `Browser did not open automatically. Open this URL manually, then refresh status:\n${url}`,
          },
        }));
      } catch (e) {
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: {
            success: false,
            message: e instanceof Error ? e.message : 'Connect failed',
          },
        }));
      } finally {
        setGoogleMeetNotesBusy(false);
      }
    };

    const disconnectGoogleMeetNotes = async () => {
      setGoogleMeetNotesBusy(true);
      try {
        const api = new ChatAPI(hubHttp);
        await api.disconnectGoogleMeetNotes();
        await refreshGoogleMeetNotesStatus();
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: { success: true, message: 'Disconnected from Google.' },
        }));
      } catch (e) {
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: {
            success: false,
            message: e instanceof Error ? e.message : 'Disconnect failed',
          },
        }));
      } finally {
        setGoogleMeetNotesBusy(false);
      }
    };

    const syncGoogleMeetNotesNow = async () => {
      setGoogleMeetNotesBusy(true);
      try {
        const api = new ChatAPI(hubHttp);
        const n = await api.syncGoogleMeetNotes();
        await refreshGoogleMeetNotesStatus();
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: {
            success: true,
            message: `Synced ${n} meeting note(s).`,
          },
        }));
      } catch (e) {
        setTestResults((prev) => ({
          ...prev,
          googleMeetNotes: {
            success: false,
            message: e instanceof Error ? e.message : 'Sync failed',
          },
        }));
      } finally {
        setGoogleMeetNotesBusy(false);
      }
    };
    const googleOAuthSourceLabel = (source?: string) => {
      switch (source) {
        case 'vendor':
          return 'Using Neural Junkie Google app';
        case 'env':
          return 'Using environment Google OAuth config';
        case 'config':
          return 'Using custom Google OAuth client';
        default:
          return 'Google OAuth unavailable';
      }
    };

    const googleConnectReady =
      googleMeetNotes?.connect_ready ?? googleMeetNotes?.oauth_configured ?? false;

  if (!isActive) return null;

  return (
    <div className="space-y-8 nj-settings-integrations text-slack-text">
    <div className="rounded border border-slack-border bg-slack-bgHover/40 p-3 text-sm text-slack-textMuted mb-2">
      Web search is now a shared hub setting under <span className="font-medium text-slack-text">External → Web search</span>.
    </div>
    {/* Google Meet notes (Assistant) */}
    <div className="border border-slack-border rounded-lg p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-slack-text">Google Meet notes</h3>
        <button
          type="button"
          onClick={() => void refreshGoogleMeetNotesStatus()}
          disabled={googleMeetNotesLoading}
          className="px-3 py-1 text-sm border border-slack-border rounded hover:bg-slack-bgHover text-slack-text"
        >
          Refresh
        </button>
      </div>
      <p className="text-sm text-slack-textMuted mb-4">
        Connect your Google account to sync Gemini meeting notes from Gmail into Assistant.
      </p>
      <div
        className={`mb-4 rounded border p-3 text-sm ${
          googleConnectReady
            ? 'border-green-200 bg-green-50 text-green-800'
            : 'border-yellow-200 bg-yellow-50 text-yellow-800'
        }`}
      >
        {googleOAuthSourceLabel(googleMeetNotes?.oauth_source)}
        {!googleConnectReady && (
          <span className="block mt-1">
            Use a release build with bundled credentials, set env vars, or configure
            Advanced Google OAuth.
          </span>
        )}
      </div>
      {testResults.googleMeetNotes && (
        <div
          className={`mb-4 p-3 rounded text-sm whitespace-pre-wrap break-all ${
            testResults.googleMeetNotes.success
              ? 'bg-green-100 text-green-800 border border-green-200'
              : 'bg-red-100 text-red-800 border border-red-200'
          }`}
        >
          {testResults.googleMeetNotes.message}
        </div>
      )}
      <div className="flex flex-wrap gap-2 mb-4">
        <button
          type="button"
          onClick={() => void connectGoogleMeetNotes()}
          disabled={googleMeetNotesBusy || !googleConnectReady}
          className="px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
        >
          Connect Google
        </button>
        <button
          type="button"
          onClick={() => void syncGoogleMeetNotesNow()}
          disabled={googleMeetNotesBusy || !googleMeetNotes?.connected}
          className="px-4 py-2 border border-slack-border rounded hover:bg-slack-bgHover text-slack-text disabled:opacity-50"
        >
          Sync now
        </button>
        <button
          type="button"
          onClick={() => void disconnectGoogleMeetNotes()}
          disabled={googleMeetNotesBusy || !googleMeetNotes?.connected}
          className="px-4 py-2 text-red-600 border border-red-300 rounded hover:bg-red-50 disabled:opacity-50"
        >
          Disconnect
        </button>
      </div>
      {googleMeetNotesLoading && !googleMeetNotes ? (
        <p className="text-sm text-slack-textMuted">Loading status…</p>
      ) : googleMeetNotes ? (
        <div className="space-y-3 text-sm text-slack-text">
          <p>
            <span className="font-medium">Account:</span>{' '}
            {googleMeetNotes.connected
              ? googleMeetNotes.email || 'connected'
              : 'not connected'}
          </p>
          {googleMeetNotes.connected && (
            <>
              <p>
                <span className="font-medium">Stored notes:</span>{' '}
                {googleMeetNotes.notes_count ?? 0}
              </p>
              {googleMeetNotes.last_sync_at && (
                <p>
                  <span className="font-medium">Last sync:</span>{' '}
                  {new Date(googleMeetNotes.last_sync_at).toLocaleString()}
                </p>
              )}
            </>
          )}
        </div>
      ) : null}
      <details
        open={googleAdvancedOpen}
        onToggle={(e) => setGoogleAdvancedOpen(e.currentTarget.open)}
        className="mt-4 border border-slack-border rounded-lg p-4"
      >
        <summary className="cursor-pointer text-sm font-medium text-slack-text">
          Advanced (bring your own Google OAuth client)
        </summary>
        <div className="space-y-4 mt-4">
          <p className="text-sm text-slack-textMuted">
            Create a Google Cloud OAuth web client, add the redirect URI below, then save
            your Client ID and Secret.
          </p>
          <div>
            <label className="block text-sm font-medium text-slack-text mb-2">
              OAuth Client ID
            </label>
            <input
              type="text"
              value={googleOAuthForm.clientId}
              onChange={(e) =>
                setGoogleOAuthForm((prev) => ({ ...prev, clientId: e.target.value }))
              }
              placeholder="xxxx.apps.googleusercontent.com"
              className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slack-text mb-2">
              OAuth Client Secret
              {googleOAuthSecretSet && !googleOAuthForm.clientSecret && (
                <span className="ml-2 text-xs text-green-600">(saved)</span>
              )}
            </label>
            <input
              type="password"
              value={googleOAuthForm.clientSecret}
              onChange={(e) =>
                setGoogleOAuthForm((prev) => ({ ...prev, clientSecret: e.target.value }))
              }
              placeholder={
                googleOAuthSecretSet ? 'Leave blank to keep existing secret' : 'Client secret'
              }
              className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slack-text mb-2">
              Redirect URI
            </label>
            <input
              type="text"
              value={googleOAuthForm.redirectUrl}
              onChange={(e) =>
                setGoogleOAuthForm((prev) => ({ ...prev, redirectUrl: e.target.value }))
              }
              className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text font-mono text-xs focus:outline-none focus:ring-2 focus:ring-slack-accent"
            />
            <p className="text-xs text-slack-textMuted mt-1">
              Add this exact URI in Google Cloud Console → Credentials → your OAuth
              client.
            </p>
          </div>
          <button
            type="button"
            onClick={() => void saveGoogleOAuthSettings()}
            disabled={
              googleMeetNotesBusy ||
              !googleOAuthForm.clientId.trim() ||
              (!googleOAuthSecretSet && !googleOAuthForm.clientSecret.trim())
            }
            className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
          >
            Save OAuth credentials
          </button>
        </div>
      </details>
    </div>
    </div>
  );
}
