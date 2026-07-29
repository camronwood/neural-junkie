import { useState, useEffect } from 'react';
import { ChatAPI } from '../../api/chatAPI';
import type { WebSearchConfigResponse } from '../../types/protocol';
import { openExternalLink, type SettingsTabProps } from './settingsShared';

export function WebSearchSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const [webSearchConfig, setWebSearchConfig] = useState<WebSearchConfigResponse | null>(null);
  const [webSearchForm, setWebSearchForm] = useState({
    enabled: false,
    provider: 'tavily' as 'tavily' | 'brave',
    apiKey: '',
    maxResults: 5,
    keyless: false,
  });
  const [webSearchBusy, setWebSearchBusy] = useState(false);
  const [webSearchFeedback, setWebSearchFeedback] = useState<{ success: boolean; message: string } | null>(null);
  const [showApiKey, setShowApiKey] = useState(false);

  const refreshWebSearchConfig = async () => {
    try {
      const api = new ChatAPI(hubHttp);
      const cfg = await api.getWebSearchConfig();
      setWebSearchConfig(cfg);
      setWebSearchForm((prev) => ({
        ...prev,
        enabled: cfg.enabled,
        provider: cfg.provider === 'brave' ? 'brave' : 'tavily',
        maxResults: cfg.max_results || 5,
        keyless: cfg.keyless ?? false,
      }));
    } catch (e) {
      setWebSearchFeedback({
        success: false,
        message: e instanceof Error ? e.message : 'Failed to load web search settings',
      });
    }
  };

  useEffect(() => {
    if (!isActive) return;
    void refreshWebSearchConfig();
  }, [isActive, hubHttp]);

  const saveWebSearchSettings = async () => {
    setWebSearchBusy(true);
    setWebSearchFeedback(null);
    try {
      const api = new ChatAPI(hubHttp);
      await api.saveWebSearchConfig({
        enabled: webSearchForm.enabled,
        provider: webSearchForm.provider,
        api_key: webSearchForm.apiKey || undefined,
        max_results: webSearchForm.maxResults,
        keyless: webSearchForm.keyless,
      });
      setWebSearchForm((prev) => ({ ...prev, apiKey: '' }));
      await refreshWebSearchConfig();
      setWebSearchFeedback({
        success: true,
        message: 'Web search settings saved.',
      });
    } catch (e) {
      setWebSearchFeedback({
        success: false,
        message: e instanceof Error ? e.message : 'Failed to save web search settings',
      });
    } finally {
      setWebSearchBusy(false);
    }
  };

  const testWebSearchSettings = async () => {
    setWebSearchBusy(true);
    setWebSearchFeedback(null);
    try {
      const api = new ChatAPI(hubHttp);
      const result = await api.testWebSearchConnection();
      const title = result.results?.[0]?.title ?? 'connection ok';
      setWebSearchFeedback({ success: true, message: `Web search test succeeded (${title}).` });
    } catch (e) {
      setWebSearchFeedback({
        success: false,
        message: e instanceof Error ? e.message : 'Web search test failed',
      });
    } finally {
      setWebSearchBusy(false);
    }
  };

  if (!isActive) return null;

  return (
    <div className="space-y-8 text-slack-text">
      <div className="border border-slack-border rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-slack-text">Web search</h3>
          <div className="flex items-center space-x-2">
            {webSearchConfig?.ready && <span className="text-green-500 text-sm">✓ Ready</span>}
            <button
              type="button"
              onClick={() => void testWebSearchSettings()}
              disabled={webSearchBusy || !webSearchConfig?.ready}
              className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
            >
              Test
            </button>
          </div>
        </div>
        <p className="text-sm text-slack-textMuted mb-4">
          Shared hub setting for every agent. When enabled, specialists can use{' '}
          <code className="text-xs">web_search</code> and <code className="text-xs">fetch_url</code>{' '}
          for current public web facts. Default provider is{' '}
          <button
            type="button"
            onClick={() => openExternalLink('https://tavily.com')}
            className="text-slack-accent hover:underline"
          >
            Tavily
          </button>{' '}
          (1,000 free searches/month, no card).{' '}
          <button
            type="button"
            onClick={() => openExternalLink('https://brave.com/search/api/')}
            className="text-slack-accent hover:underline"
          >
            Brave
          </button>{' '}
          is also supported.
        </p>
        {webSearchFeedback && (
          <div
            className={`mb-4 p-3 rounded text-sm ${
              webSearchFeedback.success
                ? 'bg-green-100 text-green-800 border border-green-200'
                : 'bg-red-100 text-red-800 border border-red-200'
            }`}
          >
            {webSearchFeedback.message}
          </div>
        )}
        <div className="space-y-4">
          <label className="flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={webSearchForm.enabled}
              onChange={(e) => setWebSearchForm((prev) => ({ ...prev, enabled: e.target.checked }))}
            />
            Enable web search tools for all agents
          </label>
          <div>
            <label className="block text-sm font-medium text-slack-text mb-2">Provider</label>
            <select
              value={webSearchForm.provider}
              onChange={(e) =>
                setWebSearchForm((prev) => ({
                  ...prev,
                  provider: e.target.value === 'brave' ? 'brave' : 'tavily',
                }))
              }
              className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
            >
              <option value="tavily">Tavily (recommended — free tier, no card)</option>
              <option value="brave">Brave Search API</option>
            </select>
          </div>
          {webSearchForm.provider === 'tavily' && (
            <label className="flex items-center gap-2 text-sm text-slack-text">
              <input
                type="checkbox"
                checked={webSearchForm.keyless}
                onChange={(e) => setWebSearchForm((prev) => ({ ...prev, keyless: e.target.checked }))}
              />
              Keyless mode (no API key — rate limited, good for trying locally)
            </label>
          )}
          <div>
            <label className="block text-sm font-medium text-slack-text mb-2">
              {webSearchForm.provider === 'brave' ? 'Brave API key' : 'Tavily API key (optional with keyless)'}
            </label>
            <div className="relative">
              <input
                type={showApiKey ? 'text' : 'password'}
                value={webSearchForm.apiKey}
                onChange={(e) => setWebSearchForm((prev) => ({ ...prev, apiKey: e.target.value }))}
                placeholder={
                  webSearchConfig?.api_key_set
                    ? '•••••••• (saved — enter to replace)'
                    : webSearchForm.provider === 'brave'
                      ? 'BSA...'
                      : 'tvly-...'
                }
                className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
              />
              <button
                type="button"
                onClick={() => setShowApiKey((v) => !v)}
                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slack-textMuted hover:text-slack-text"
              >
                {showApiKey ? '👁️' : '👁️‍🗨️'}
              </button>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-slack-text mb-2">Max results per search</label>
            <input
              type="number"
              min={1}
              max={20}
              value={webSearchForm.maxResults}
              onChange={(e) =>
                setWebSearchForm((prev) => ({
                  ...prev,
                  maxResults: Math.max(1, Math.min(20, Number(e.target.value) || 5)),
                }))
              }
              className="w-32 px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
            />
          </div>
          <button
            type="button"
            onClick={() => void saveWebSearchSettings()}
            disabled={webSearchBusy}
            className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
          >
            Save web search settings
          </button>
        </div>
      </div>
    </div>
  );
}
