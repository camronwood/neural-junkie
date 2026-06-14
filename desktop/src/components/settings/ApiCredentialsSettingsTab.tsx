import { useState, useEffect } from 'react';
import { useSettingsStore } from '../../stores/settingsStore';
import type {
  AnthropicSettings,
  GitHubSettings,
  ConfluenceSettings,
} from '../../types/protocol';
import { openExternalLink, type SettingsTabProps } from './settingsShared';

export function ApiCredentialsSettingsTab({ hubHttp: _hubHttp, isActive }: SettingsTabProps) {
  const {
    integrations,
    loadIntegrations,
    updateAnthropicSettings,
    updateGitHubSettings,
    updateConfluenceSettings,
    clearIntegrationSettings,
    testAnthropicConnection,
    testGitHubConnection,
    testConfluenceConnection,
  } = useSettingsStore();

// Integration form states
    const [anthropicForm, setAnthropicForm] = useState<AnthropicSettings>(integrations.anthropic);
    const [githubForm, setGitHubForm] = useState<GitHubSettings>(integrations.github);
    const [confluenceForm, setConfluenceForm] = useState<ConfluenceSettings>(integrations.confluence);
    const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({});
    const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});

    useEffect(() => {
      if (!isActive) return;
      loadIntegrations();
    }, [isActive, loadIntegrations]);

    useEffect(() => {
      setAnthropicForm(integrations.anthropic);
      setGitHubForm(integrations.github);
      setConfluenceForm(integrations.confluence);
    }, [integrations]);

// Integration handlers
    const handleAnthropicChange = (field: keyof AnthropicSettings, value: string | boolean) => {
      setAnthropicForm(prev => ({ ...prev, [field]: value }));
    };

    const handleGitHubChange = (field: keyof GitHubSettings, value: string) => {
      setGitHubForm(prev => ({ ...prev, [field]: value }));
    };

    const handleConfluenceChange = (field: keyof ConfluenceSettings, value: string) => {
      setConfluenceForm(prev => ({ ...prev, [field]: value }));
    };

    const saveAnthropicSettings = async () => {
      try {
        await updateAnthropicSettings(anthropicForm);
        setTestResults(prev => ({ ...prev, anthropic: { success: true, message: 'Settings saved successfully!' } }));
      } catch (error) {
        setTestResults(prev => ({ 
          ...prev, 
          anthropic: { 
            success: false, 
            message: error instanceof Error ? error.message : 'Failed to save settings' 
          } 
        }));
      }
    };

    const saveGitHubSettings = async () => {
      try {
        await updateGitHubSettings(githubForm);
        setTestResults(prev => ({ ...prev, github: { success: true, message: 'Settings saved successfully!' } }));
      } catch (error) {
        setTestResults(prev => ({ 
          ...prev, 
          github: { 
            success: false, 
            message: error instanceof Error ? error.message : 'Failed to save settings' 
          } 
        }));
      }
    };

    const saveConfluenceSettings = async () => {
      try {
        await updateConfluenceSettings(confluenceForm);
        setTestResults(prev => ({ ...prev, confluence: { success: true, message: 'Settings saved successfully!' } }));
      } catch (error) {
        setTestResults(prev => ({ 
          ...prev, 
          confluence: { 
            success: false, 
            message: error instanceof Error ? error.message : 'Failed to save settings' 
          } 
        }));
      }
    };

    const togglePasswordVisibility = (field: string) => {
      setShowPasswords(prev => ({ ...prev, [field]: !prev[field] }));
    };

    const testConnection = async (service: string) => {
      setTestResults(prev => ({ ...prev, [service]: { success: false, message: 'Testing...' } }));
    
      try {
        let result = false;
        switch (service) {
          case 'anthropic':
            result = await testAnthropicConnection();
            break;
          case 'github':
            result = await testGitHubConnection();
            break;
          case 'confluence':
            result = await testConfluenceConnection();
            break;
        }
      
        setTestResults(prev => ({ 
          ...prev, 
          [service]: { 
            success: result, 
            message: result ? 'Connection successful!' : 'Connection failed. Check your credentials.' 
          } 
        }));
      } catch (error) {
        setTestResults(prev => ({ 
          ...prev, 
          [service]: { 
            success: false, 
            message: `Error: ${error instanceof Error ? error.message : 'Unknown error'}` 
          } 
        }));
      }
    };

    const clearAllIntegrations = async () => {
      if (confirm('Are you sure you want to clear all integration settings? This action cannot be undone.')) {
        await clearIntegrationSettings();
        setAnthropicForm(integrations.anthropic);
        setGitHubForm(integrations.github);
        setConfluenceForm(integrations.confluence);
      }
    };

  if (!isActive) return null;

  return (
    <div className="space-y-8 nj-settings-integrations text-slack-text">
{/* Anthropic Settings */}
    <div className="border border-slack-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-slack-text">Anthropic API</h3>
        <div className="flex items-center space-x-2">
          {anthropicForm.apiKey && (
            <span className="text-green-500 text-sm">✓ Configured</span>
          )}
          <button
            onClick={() => testConnection('anthropic')}
            className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
          >
            Test
          </button>
        </div>
      </div>
      
      {testResults.anthropic && (
        <div className={`mb-4 p-3 rounded text-sm ${
          testResults.anthropic.success 
            ? 'bg-green-100 text-green-800 border border-green-200' 
            : 'bg-red-100 text-red-800 border border-red-200'
        }`}>
          {testResults.anthropic.message}
        </div>
      )}

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            API Key
          </label>
          <div className="relative">
            <input
              type={showPasswords.anthropic ? 'text' : 'password'}
              value={anthropicForm.apiKey}
              onChange={(e) => handleAnthropicChange('apiKey', e.target.value)}
              placeholder="sk-ant-..."
              className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
            />
            <button
              type="button"
              onClick={() => togglePasswordVisibility('anthropic')}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slack-textMuted hover:text-slack-text"
            >
              {showPasswords.anthropic ? '👁️' : '👁️‍🗨️'}
            </button>
          </div>
          <p className="text-xs text-slack-textMuted mt-1">
            Get your API key from{' '}
            <button
              onClick={() => openExternalLink('https://console.anthropic.com/')}
              className="text-slack-accent hover:underline"
            >
              Anthropic Console
            </button>
          </p>
        </div>

        <div className="flex items-center space-x-2">
          <input
            type="checkbox"
            id="useAIHub"
            checked={anthropicForm.useAIHub}
            onChange={(e) => handleAnthropicChange('useAIHub', e.target.checked)}
            className="text-slack-accent focus:ring-slack-accent"
          />
          <label htmlFor="useAIHub" className="text-sm text-slack-text">
            Use AI Hub (recommended)
          </label>
        </div>

        {anthropicForm.useAIHub && (
          <>
            <div>
              <label className="block text-sm font-medium text-slack-text mb-2">
                AI Hub Endpoint
              </label>
              <input
                type="text"
                value={anthropicForm.aiHubEndpoint}
                onChange={(e) => handleAnthropicChange('aiHubEndpoint', e.target.value)}
                className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slack-text mb-2">
                Model
              </label>
              <select
                value={anthropicForm.aiHubModel}
                onChange={(e) => handleAnthropicChange('aiHubModel', e.target.value)}
                className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
              >
                <option value="claude-sonnet">Claude Sonnet (recommended)</option>
                <option value="claude-haiku">Claude Haiku (faster)</option>
              </select>
            </div>
          </>
        )}

        <button
          onClick={saveAnthropicSettings}
          className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
        >
          Save Anthropic Settings
        </button>
      </div>
    </div>
{/* GitHub Settings */}
    <div className="border border-slack-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-slack-text">GitHub</h3>
        <div className="flex items-center space-x-2">
          {githubForm.personalAccessToken && (
            <span className="text-green-500 text-sm">✓ Configured</span>
          )}
          <button
            onClick={() => testConnection('github')}
            className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
          >
            Test
          </button>
        </div>
      </div>
      
      {testResults.github && (
        <div className={`mb-4 p-3 rounded text-sm ${
          testResults.github.success 
            ? 'bg-green-100 text-green-800 border border-green-200' 
            : 'bg-red-100 text-red-800 border border-red-200'
        }`}>
          {testResults.github.message}
        </div>
      )}

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            Personal Access Token
          </label>
          <div className="relative">
            <input
              type={showPasswords.github ? 'text' : 'password'}
              value={githubForm.personalAccessToken}
              onChange={(e) => handleGitHubChange('personalAccessToken', e.target.value)}
              placeholder="ghp_..."
              className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
            />
            <button
              type="button"
              onClick={() => togglePasswordVisibility('github')}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slack-textMuted hover:text-slack-text"
            >
              {showPasswords.github ? '👁️' : '👁️‍🗨️'}
            </button>
          </div>
          <p className="text-xs text-slack-textMuted mt-1">
            Create a token at{' '}
            <button
              onClick={() => openExternalLink('https://github.com/settings/tokens')}
              className="text-slack-accent hover:underline"
            >
              GitHub Settings
            </button>
            {' '}with repo, read:org permissions
          </p>
        </div>

        <button
          onClick={saveGitHubSettings}
          className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
        >
          Save GitHub Settings
        </button>
      </div>
    </div>
{/* Confluence Settings */}
    <div className="border border-slack-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-slack-text">Confluence</h3>
        <div className="flex items-center space-x-2">
          {confluenceForm.domain && confluenceForm.email && confluenceForm.apiToken && (
            <span className="text-green-500 text-sm">✓ Configured</span>
          )}
          <button
            onClick={() => testConnection('confluence')}
            className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
          >
            Test
          </button>
        </div>
      </div>
      
      {testResults.confluence && (
        <div className={`mb-4 p-3 rounded text-sm ${
          testResults.confluence.success 
            ? 'bg-green-100 text-green-800 border border-green-200' 
            : 'bg-red-100 text-red-800 border border-red-200'
        }`}>
          {testResults.confluence.message}
        </div>
      )}

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            Domain
          </label>
          <input
            type="text"
            value={confluenceForm.domain}
            onChange={(e) => handleConfluenceChange('domain', e.target.value)}
            placeholder="yourcompany.atlassian.net"
            className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            Email
          </label>
          <input
            type="email"
            value={confluenceForm.email}
            onChange={(e) => handleConfluenceChange('email', e.target.value)}
            placeholder="your.email@company.com"
            className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            API Token
          </label>
          <div className="relative">
            <input
              type={showPasswords.confluence ? 'text' : 'password'}
              value={confluenceForm.apiToken}
              onChange={(e) => handleConfluenceChange('apiToken', e.target.value)}
              placeholder="Your API token"
              className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
            />
            <button
              type="button"
              onClick={() => togglePasswordVisibility('confluence')}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slack-textMuted hover:text-slack-text"
            >
              {showPasswords.confluence ? '👁️' : '👁️‍🗨️'}
            </button>
          </div>
          <p className="text-xs text-slack-textMuted mt-1">
            Get your API token from{' '}
            <button
              onClick={() => openExternalLink('https://id.atlassian.com/manage-profile/security/api-tokens')}
              className="text-slack-accent hover:underline"
            >
              Atlassian Account Settings
            </button>
          </p>
        </div>

        <button
          onClick={saveConfluenceSettings}
          className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
        >
          Save Confluence Settings
        </button>
      </div>
    </div>

    {/* Clear All Button */}
    <div className="pt-4 border-t border-slack-border">
      <button
        onClick={clearAllIntegrations}
        className="px-4 py-2 text-red-600 border border-red-300 rounded hover:bg-red-50 transition-colors"
      >
        Clear All Integration Settings
      </button>
    </div>
    </div>
  );
}
